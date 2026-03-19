package dns

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Blocker struct {
	domains   map[string]bool
	whitelist map[string]bool
	sources   []string
	mu        sync.RWMutex

	TotalDomains int
}

func NewBlocker(sources []string, whitelist []string) *Blocker {
	wl := make(map[string]bool, len(whitelist))
	for _, d := range whitelist {
		wl[normalizeDomain(d)] = true
	}

	b := &Blocker{
		domains:   make(map[string]bool),
		whitelist: wl,
		sources:   sources,
	}

	if err := b.Reload(); err != nil {
		log.Printf("[DNS Blocker] 블록리스트 초기 로드 실패: %v", err)
	}

	return b
}

func (b *Blocker) IsBlocked(domain string) bool {
	domain = normalizeDomain(domain)

	b.mu.RLock()
	defer b.mu.RUnlock()

	// 화이트리스트 우선 확인
	if b.whitelist[domain] {
		return false
	}

	// 정확한 도메인 매칭 + 상위 도메인 매칭
	parts := strings.Split(domain, ".")
	for i := range parts {
		candidate := strings.Join(parts[i:], ".")
		if b.whitelist[candidate] {
			return false
		}
		if b.domains[candidate] {
			return true
		}
	}

	return false
}

func (b *Blocker) Reload() error {
	domains := make(map[string]bool)
	var totalErr error

	for _, source := range b.sources {
		var reader io.ReadCloser
		var err error

		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			reader, err = fetchURL(source)
		} else {
			reader, err = os.Open(source)
		}
		if err != nil {
			log.Printf("[DNS Blocker] 소스 로드 실패 (%s): %v", source, err)
			totalErr = fmt.Errorf("%v; %w", totalErr, err)
			continue
		}

		count := parseBlocklist(reader, domains)
		reader.Close()
		log.Printf("[DNS Blocker] 로드 완료: %s (%d개 도메인)", source, count)
	}

	b.mu.Lock()
	b.domains = domains
	b.TotalDomains = len(domains)
	b.mu.Unlock()

	log.Printf("[DNS Blocker] 총 %d개 도메인 차단 중", len(domains))
	return totalErr
}

func (b *Blocker) Stats() BlockerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BlockerStats{
		TotalDomains: len(b.domains),
		Sources:      len(b.sources),
		Whitelist:    len(b.whitelist),
	}
}

type BlockerStats struct {
	TotalDomains int `json:"total_domains"`
	Sources      int `json:"sources"`
	Whitelist    int `json:"whitelist"`
}

func fetchURL(url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func parseBlocklist(r io.Reader, domains map[string]bool) int {
	count := 0
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 빈 줄, 주석 건너뛰기
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		var domain string

		// hosts 파일 형식: "0.0.0.0 ads.example.com" 또는 "127.0.0.1 ads.example.com"
		if strings.HasPrefix(line, "0.0.0.0") || strings.HasPrefix(line, "127.0.0.1") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				domain = fields[1]
			}
		} else if !strings.Contains(line, " ") && strings.Contains(line, ".") {
			// 도메인 리스트 형식: "ads.example.com"
			domain = line
		}

		if domain == "" {
			continue
		}

		domain = normalizeDomain(domain)
		if domain == "" || domain == "localhost" {
			continue
		}

		domains[domain] = true
		count++
	}

	return count
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, ".")
	return domain
}
