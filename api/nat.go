package api

import (
	"encoding/json"
	"home-router/internal/config"
	"home-router/nat"
	"log"
	"net/http"
)

type portForwardResponse struct {
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"externalPort"`
	InternalIP   string `json:"internalIp"`
	InternalPort int    `json:"internalPort"`
}

func (s *Server) handleNATPortForwards(w http.ResponseWriter, r *http.Request) {
	forwards := make([]portForwardResponse, 0, len(s.cfg.PortForwarding))
	for _, pf := range s.cfg.PortForwarding {
		forwards = append(forwards, portForwardResponse{
			Name:         pf.Name,
			Protocol:     pf.Protocol,
			ExternalPort: pf.ExternalPort,
			InternalIP:   pf.InternalIP,
			InternalPort: pf.InternalPort,
		})
	}
	writeJSON(w, forwards)
}

type addPortForwardRequest struct {
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"externalPort"`
	InternalIP   string `json:"internalIp"`
	InternalPort int    `json:"internalPort"`
}

func (s *Server) handleNATAddPortForward(w http.ResponseWriter, r *http.Request) {
	var req addPortForwardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Protocol == "" || req.ExternalPort == 0 || req.InternalIP == "" || req.InternalPort == 0 {
		http.Error(w, `{"error":"all fields required"}`, http.StatusBadRequest)
		return
	}

	if req.Protocol != "tcp" && req.Protocol != "udp" {
		http.Error(w, `{"error":"protocol must be tcp or udp"}`, http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	wanIP := s.wanIP
	s.mu.RUnlock()

	pf := nat.PortForward{
		Name:         req.Name,
		Protocol:     req.Protocol,
		ExternalPort: req.ExternalPort,
		InternalIP:   req.InternalIP,
		InternalPort: req.InternalPort,
	}

	if err := nat.AddPortForward(s.wanIface, wanIP, pf); err != nil {
		http.Error(w, `{"error":"failed to add port forward: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	s.cfg.PortForwarding = append(s.cfg.PortForwarding, config.PortForwardEntry{
		Name:         req.Name,
		Protocol:     req.Protocol,
		ExternalPort: req.ExternalPort,
		InternalIP:   req.InternalIP,
		InternalPort: req.InternalPort,
	})

	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 포트포워딩 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleNATRemovePortForward(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	wanIP := s.wanIP
	s.mu.RUnlock()

	found := false
	for i, pf := range s.cfg.PortForwarding {
		if pf.Name == name {
			nat.RemovePortForward(s.wanIface, wanIP, nat.PortForward{
				Name:         pf.Name,
				Protocol:     pf.Protocol,
				ExternalPort: pf.ExternalPort,
				InternalIP:   pf.InternalIP,
				InternalPort: pf.InternalPort,
			})
			s.cfg.PortForwarding = append(s.cfg.PortForwarding[:i], s.cfg.PortForwarding[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error":"port forward not found"}`, http.StatusNotFound)
		return
	}

	if err := s.cfg.Save(); err != nil {
		log.Printf("[API] 포트포워딩 설정 저장 실패: %v", err)
	}

	writeJSON(w, map[string]string{"status": "ok"})
}
