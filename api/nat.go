package api

import (
	"encoding/json"
	"home-router/nat"
	"net/http"
)

type portForwardResponse struct {
	Name         string `json:"name"`
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"external_port"`
	InternalIP   string `json:"internal_ip"`
	InternalPort int    `json:"internal_port"`
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
	ExternalPort int    `json:"external_port"`
	InternalIP   string `json:"internal_ip"`
	InternalPort int    `json:"internal_port"`
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

	// config에 추가 (메모리만, 파일 저장 안 함)
	s.cfg.PortForwarding = append(s.cfg.PortForwarding, struct {
		Name         string `yaml:"name"`
		Protocol     string `yaml:"protocol"`
		ExternalPort int    `yaml:"external_port"`
		InternalIP   string `yaml:"internal_ip"`
		InternalPort int    `yaml:"internal_port"`
	}{
		Name:         req.Name,
		Protocol:     req.Protocol,
		ExternalPort: req.ExternalPort,
		InternalIP:   req.InternalIP,
		InternalPort: req.InternalPort,
	})

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

	writeJSON(w, map[string]string{"status": "ok"})
}
