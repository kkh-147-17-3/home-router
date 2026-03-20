package api

import (
	"encoding/json"
	"home-router/internal/config"
	"log"
	"net"
	"net/http"
	"regexp"
)

func (s *Server) handleDHCPLeases(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeJSON(w, []struct{}{})
		return
	}
	writeJSON(w, s.pool.ActiveLeases())
}

func (s *Server) handleDHCPPool(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		http.Error(w, `{"error":"pool not available"}`, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, s.pool.PoolInfo())
}

func (s *Server) handleDHCPStaticLeases(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		writeJSON(w, []struct{}{})
		return
	}
	writeJSON(w, s.pool.StaticLeaseList())
}

type addStaticLeaseRequest struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IP   string `json:"ip"`
}

var macRegex = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)

func (s *Server) handleDHCPAddStaticLease(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		http.Error(w, `{"error":"pool not available"}`, http.StatusServiceUnavailable)
		return
	}

	var req addStaticLeaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if !macRegex.MatchString(req.MAC) {
		http.Error(w, `{"error":"invalid MAC address"}`, http.StatusBadRequest)
		return
	}

	ip := net.ParseIP(req.IP)
	if ip == nil {
		http.Error(w, `{"error":"invalid IP address"}`, http.StatusBadRequest)
		return
	}

	s.pool.AddStaticLease(req.Name, req.MAC, ip)

	// Sync to config and save
	s.cfg.Dhcp.StaticLeases = append(s.cfg.Dhcp.StaticLeases, config.StaticLeaseEntry{
		Name:       req.Name,
		MacAddress: req.MAC,
		IP:         req.IP,
	})
	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 정적 임대 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleDHCPRemoveStaticLease(w http.ResponseWriter, r *http.Request) {
	if s.pool == nil {
		http.Error(w, `{"error":"pool not available"}`, http.StatusServiceUnavailable)
		return
	}

	mac := r.PathValue("mac")
	if mac == "" {
		http.Error(w, `{"error":"mac required"}`, http.StatusBadRequest)
		return
	}

	s.pool.RemoveStaticLease(mac)

	// Sync to config and save
	for i, sl := range s.cfg.Dhcp.StaticLeases {
		if sl.MacAddress == mac {
			s.cfg.Dhcp.StaticLeases = append(s.cfg.Dhcp.StaticLeases[:i], s.cfg.Dhcp.StaticLeases[i+1:]...)
			break
		}
	}
	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 정적 임대 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}
