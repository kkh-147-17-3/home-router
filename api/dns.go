package api

import (
	"encoding/json"
	hdns "home-router/dns"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleDNSStats(w http.ResponseWriter, r *http.Request) {
	if s.queryLog == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}
	stats := s.queryLog.Stats()
	s.enrichTopClients(stats.TopClients)
	writeJSON(w, stats)
}

func (s *Server) handleDNSQueryLog(w http.ResponseWriter, r *http.Request) {
	if s.queryLog == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	entries := s.queryLog.Recent(limit)

	// 필터링
	blockedOnly := r.URL.Query().Get("blocked") == "true"
	domainFilter := strings.ToLower(r.URL.Query().Get("domain"))
	clientFilter := r.URL.Query().Get("client")

	if blockedOnly || domainFilter != "" || clientFilter != "" {
		filtered := make([]interface{}, 0)
		for _, e := range entries {
			if blockedOnly && !e.Blocked {
				continue
			}
			if domainFilter != "" && !strings.Contains(strings.ToLower(e.Domain), domainFilter) {
				continue
			}
			if clientFilter != "" && e.ClientIP != clientFilter {
				continue
			}
			filtered = append(filtered, e)
		}
		writeJSON(w, filtered)
		return
	}

	writeJSON(w, entries)
}

func (s *Server) handleDNSCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, s.cache.Stats())
}

func (s *Server) handleDNSBlockerStats(w http.ResponseWriter, r *http.Request) {
	if s.blocker == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, s.blocker.Stats())
}

func (s *Server) handleDNSBlockerReload(w http.ResponseWriter, r *http.Request) {
	if s.blocker == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}
	go s.blocker.Reload()
	writeJSON(w, map[string]string{"status": "ok", "message": "reload started"})
}

func (s *Server) handleDNSWhitelist(w http.ResponseWriter, r *http.Request) {
	if s.blocker == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	type whitelistResponse struct {
		Entries []string `json:"entries"`
		Sources []string `json:"sources"`
	}
	writeJSON(w, whitelistResponse{
		Entries: s.blocker.WhitelistEntries(),
		Sources: s.blocker.BlocklistSources(),
	})
}

type addWhitelistRequest struct {
	Domain string `json:"domain"`
}

func (s *Server) handleDNSAddWhitelist(w http.ResponseWriter, r *http.Request) {
	if s.blocker == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	var req addWhitelistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, `{"error":"domain required"}`, http.StatusBadRequest)
		return
	}

	s.blocker.AddWhitelist(req.Domain)

	// Sync to config and save
	s.cfg.Dns.Whitelist = append(s.cfg.Dns.Whitelist, req.Domain)
	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 화이트리스트 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleDNSRemoveWhitelist(w http.ResponseWriter, r *http.Request) {
	if s.blocker == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	domain := r.PathValue("domain")
	if domain == "" {
		http.Error(w, `{"error":"domain required"}`, http.StatusBadRequest)
		return
	}

	s.blocker.RemoveWhitelist(domain)

	// Sync to config and save
	for i, d := range s.cfg.Dns.Whitelist {
		if d == domain {
			s.cfg.Dns.Whitelist = append(s.cfg.Dns.Whitelist[:i], s.cfg.Dns.Whitelist[i+1:]...)
			break
		}
	}
	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 화이트리스트 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

// enrichTopClients adds hostname labels to TopClients entries (IP → hostname).
func (s *Server) enrichTopClients(clients []hdns.TopEntry) {
	hostnames := make(map[string]string)
	for _, sl := range s.cfg.Dhcp.StaticLeases {
		hostnames[sl.IP] = sl.Name
	}
	if s.pool != nil {
		for _, l := range s.pool.ActiveLeases() {
			if l.Hostname != "" {
				hostnames[l.IP] = l.Hostname
			}
		}
	}
	for i := range clients {
		if name, ok := hostnames[clients[i].Name]; ok {
			clients[i].Label = name
		}
	}
}
