package ddns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DDNSProvider updates a DNS record with the given IP.
type DDNSProvider interface {
	Update(ip string) error
	Name() string
}

// CloudflareProvider uses Cloudflare API v4 to update a DNS A record.
type CloudflareProvider struct {
	Token    string
	ZoneID   string
	RecordID string
	Domain   string
	Proxied  bool
}

func (c *CloudflareProvider) Name() string { return "cloudflare" }

func (c *CloudflareProvider) Update(ip string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.ZoneID, c.RecordID)

	body := map[string]interface{}{
		"type":    "A",
		"name":    c.Domain,
		"content": ip,
		"proxied": c.Proxied,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloudflare API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GenericProvider uses a simple GET request with placeholder substitution.
// Supports DuckDNS and similar services.
type GenericProvider struct {
	UpdateURL string // URL template with {{ip}} and {{domain}} placeholders
	Domain    string
	Token     string
}

func (g *GenericProvider) Name() string { return "generic" }

func (g *GenericProvider) Update(ip string) error {
	url := g.UpdateURL
	url = strings.ReplaceAll(url, "{{ip}}", ip)
	url = strings.ReplaceAll(url, "{{domain}}", g.Domain)
	url = strings.ReplaceAll(url, "{{token}}", g.Token)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("generic ddns request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("generic ddns error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Status holds the current DDNS state.
type Status struct {
	Enabled    bool      `json:"enabled"`
	Provider   string    `json:"provider"`
	Domain     string    `json:"domain"`
	LastIP     string    `json:"lastIp"`
	LastUpdate time.Time `json:"lastUpdate"`
	LastError  string    `json:"lastError,omitempty"`
}

// Manager manages DDNS updates.
type Manager struct {
	provider DDNSProvider
	mu       sync.RWMutex
	lastIP   string
	lastTime time.Time
	lastErr  string
	enabled  bool
	domain   string
	provName string
}

// NewManager creates a DDNSManager from config values.
func NewManager(enabled bool, provider, domain, token, zoneID, recordID string, proxied bool, updateURL string) *Manager {
	m := &Manager{
		enabled:  enabled,
		domain:   domain,
		provName: provider,
	}

	if !enabled {
		return m
	}

	switch provider {
	case "cloudflare":
		m.provider = &CloudflareProvider{
			Token:    token,
			ZoneID:   zoneID,
			RecordID: recordID,
			Domain:   domain,
			Proxied:  proxied,
		}
	case "duckdns":
		m.provider = &GenericProvider{
			UpdateURL: "https://www.duckdns.org/update?domains={{domain}}&token={{token}}&ip={{ip}}",
			Domain:    domain,
			Token:     token,
		}
	case "custom":
		if updateURL != "" {
			m.provider = &GenericProvider{
				UpdateURL: updateURL,
				Domain:    domain,
				Token:     token,
			}
		}
	}

	return m
}

// Reconfigure updates the provider configuration.
func (m *Manager) Reconfigure(enabled bool, provider, domain, token, zoneID, recordID string, proxied bool, updateURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = enabled
	m.domain = domain
	m.provName = provider
	m.provider = nil

	if !enabled {
		return
	}

	switch provider {
	case "cloudflare":
		m.provider = &CloudflareProvider{
			Token:    token,
			ZoneID:   zoneID,
			RecordID: recordID,
			Domain:   domain,
			Proxied:  proxied,
		}
	case "duckdns":
		m.provider = &GenericProvider{
			UpdateURL: "https://www.duckdns.org/update?domains={{domain}}&token={{token}}&ip={{ip}}",
			Domain:    domain,
			Token:     token,
		}
	case "custom":
		if updateURL != "" {
			m.provider = &GenericProvider{
				UpdateURL: updateURL,
				Domain:    domain,
				Token:     token,
			}
		}
	}
}

// UpdateIP is called when the WAN IP changes.
func (m *Manager) UpdateIP(ip string) {
	m.mu.Lock()
	if !m.enabled || m.provider == nil {
		m.mu.Unlock()
		return
	}
	m.lastIP = ip
	provider := m.provider
	m.mu.Unlock()

	log.Printf("[DDNS] IP 변경 감지, 업데이트 중: %s", ip)
	if err := provider.Update(ip); err != nil {
		log.Printf("[DDNS] 업데이트 실패: %v", err)
		m.mu.Lock()
		m.lastErr = err.Error()
		m.lastTime = time.Now()
		m.mu.Unlock()
		return
	}

	log.Printf("[DDNS] 업데이트 성공: %s", ip)
	m.mu.Lock()
	m.lastErr = ""
	m.lastTime = time.Now()
	m.mu.Unlock()
}

// ForceUpdate triggers a DDNS update with the current known IP.
func (m *Manager) ForceUpdate() error {
	m.mu.RLock()
	if !m.enabled || m.provider == nil {
		m.mu.RUnlock()
		return fmt.Errorf("DDNS not enabled or not configured")
	}
	ip := m.lastIP
	provider := m.provider
	m.mu.RUnlock()

	if ip == "" {
		return fmt.Errorf("no WAN IP available")
	}

	if err := provider.Update(ip); err != nil {
		m.mu.Lock()
		m.lastErr = err.Error()
		m.lastTime = time.Now()
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	m.lastErr = ""
	m.lastTime = time.Now()
	m.mu.Unlock()
	return nil
}

// Status returns the current DDNS status.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{
		Enabled:    m.enabled,
		Provider:   m.provName,
		Domain:     m.domain,
		LastIP:     m.lastIP,
		LastUpdate: m.lastTime,
		LastError:  m.lastErr,
	}
}
