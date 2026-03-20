package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (s *Server) handleDDNSStatus(w http.ResponseWriter, r *http.Request) {
	if s.ddns == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	status := s.ddns.Status()
	// Mask token
	resp := map[string]interface{}{
		"enabled":     status.Enabled,
		"provider":    status.Provider,
		"domain":      status.Domain,
		"lastIp":     status.LastIP,
		"lastUpdate": status.LastUpdate,
		"lastError":  status.LastError,
	}
	writeJSON(w, resp)
}

func (s *Server) handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	if s.ddns == nil {
		http.Error(w, `{"error":"ddns not configured"}`, http.StatusServiceUnavailable)
		return
	}

	if err := s.ddns.ForceUpdate(); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

type ddnsConfigRequest struct {
	Enabled   bool   `json:"enabled"`
	Provider  string `json:"provider"`
	Domain    string `json:"domain"`
	Token     string `json:"token"`
	ZoneID    string `json:"zoneId"`
	RecordID  string `json:"recordId"`
	Proxied   bool   `json:"proxied"`
	UpdateURL string `json:"updateUrl"`
}

func (s *Server) handleDDNSConfig(w http.ResponseWriter, r *http.Request) {
	var req ddnsConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Enabled {
		valid := []string{"cloudflare", "duckdns", "custom"}
		found := false
		for _, v := range valid {
			if req.Provider == v {
				found = true
				break
			}
		}
		if !found {
			http.Error(w, `{"error":"provider must be cloudflare, duckdns, or custom"}`, http.StatusBadRequest)
			return
		}
		if req.Domain == "" {
			http.Error(w, `{"error":"domain required"}`, http.StatusBadRequest)
			return
		}
	}

	// Update config
	s.cfg.Ddns.Enabled = req.Enabled
	s.cfg.Ddns.Provider = req.Provider
	s.cfg.Ddns.Domain = req.Domain
	s.cfg.Ddns.ZoneID = req.ZoneID
	s.cfg.Ddns.RecordID = req.RecordID
	s.cfg.Ddns.Proxied = req.Proxied
	s.cfg.Ddns.UpdateURL = req.UpdateURL
	if req.Token != "" && !strings.HasPrefix(req.Token, "***") {
		s.cfg.Ddns.Token = req.Token
	}

	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] DDNS 설정 저장 실패: %v", err)
		http.Error(w, `{"error":"failed to save config"}`, http.StatusInternalServerError)
		return
	}

	// Reconfigure manager
	if s.ddns != nil {
		s.ddns.Reconfigure(req.Enabled, req.Provider, req.Domain,
			s.cfg.Ddns.Token, req.ZoneID, req.RecordID, req.Proxied, req.UpdateURL)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}
