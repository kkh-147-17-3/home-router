package api

import (
	"net"
	"net/http"
)

type wanResponse struct {
	IP        string `json:"ip"`
	Interface string `json:"interface"`
	MAC       string `json:"mac"`
}

type lanResponse struct {
	Subnet    string `json:"subnet"`
	Interface string `json:"interface"`
	MAC       string `json:"mac"`
	Gateway   string `json:"gateway"`
}

func (s *Server) handleNetworkWAN(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	wanIP := s.wanIP
	s.mu.RUnlock()

	mac := ""
	if iface, err := net.InterfaceByName(s.wanIface); err == nil {
		mac = iface.HardwareAddr.String()
	}

	writeJSON(w, wanResponse{
		IP:        wanIP,
		Interface: s.wanIface,
		MAC:       mac,
	})
}

func (s *Server) handleNetworkLAN(w http.ResponseWriter, r *http.Request) {
	mac := ""
	if iface, err := net.InterfaceByName(s.lanIface); err == nil {
		mac = iface.HardwareAddr.String()
	}

	writeJSON(w, lanResponse{
		Subnet:    s.cfg.Network.Lan.Subnet,
		Interface: s.lanIface,
		MAC:       mac,
		Gateway:   s.cfg.Dhcp.Server.Gateway,
	})
}
