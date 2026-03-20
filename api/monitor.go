package api

import (
	"home-router/monitor"
	"net/http"
	"strconv"
)

func (s *Server) handleMonitorAccessLog(w http.ResponseWriter, r *http.Request) {
	if s.accessLog == nil {
		writeJSON(w, []struct{}{})
		return
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	sourceIP := r.URL.Query().Get("source_ip")
	var destPort int
	if dp := r.URL.Query().Get("dest_port"); dp != "" {
		destPort, _ = strconv.Atoi(dp)
	}

	entries := s.accessLog.Filter(limit, sourceIP, destPort)
	if entries == nil {
		writeJSON(w, []struct{}{})
		return
	}
	writeJSON(w, entries)
}

func (s *Server) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	if s.accessLog == nil {
		writeJSON(w, map[string]interface{}{
			"total_events":      0,
			"unique_source_ips": 0,
			"top_source_ips":    []struct{}{},
			"top_ports":         []struct{}{},
			"hourly":            map[string]int{},
		})
		return
	}

	writeJSON(w, s.accessLog.Stats())
}

func (s *Server) handleMonitorTraffic(w http.ResponseWriter, r *http.Request) {
	lanSubnet := s.cfg.Network.Lan.Subnet
	entries, err := monitor.ReadConntrack(lanSubnet)
	if err != nil {
		http.Error(w, `{"error":"failed to read conntrack: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Build hostname map from DHCP leases + static config
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

	summary := monitor.BuildTrafficSummary(entries, hostnames)

	// ReverseDNS로 목적지 도메인 채우기
	if s.dnsServer != nil && s.dnsServer.ReverseDNS != nil {
		for i := range summary.Hosts {
			for j := range summary.Hosts[i].TopDests {
				summary.Hosts[i].TopDests[j].Domain = s.dnsServer.ReverseDNS.Lookup(summary.Hosts[i].TopDests[j].IP)
			}
		}
		for i := range summary.TopDestinations {
			summary.TopDestinations[i].Domain = s.dnsServer.ReverseDNS.Lookup(summary.TopDestinations[i].IP)
		}
	}

	writeJSON(w, summary)
}

func (s *Server) handleMonitorConnections(w http.ResponseWriter, r *http.Request) {
	lanSubnet := s.cfg.Network.Lan.Subnet
	entries, err := monitor.ReadConntrack(lanSubnet)
	if err != nil {
		http.Error(w, `{"error":"failed to read conntrack: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Optional filter by host
	host := r.URL.Query().Get("host")
	if host != "" {
		filtered := make([]monitor.ConnEntry, 0)
		for _, e := range entries {
			if e.SrcIP == host {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	// ReverseDNS로 목적지 도메인 채우기
	if s.dnsServer != nil && s.dnsServer.ReverseDNS != nil {
		for i := range entries {
			entries[i].DstDomain = s.dnsServer.ReverseDNS.Lookup(entries[i].DstIP)
		}
	}

	if entries == nil {
		entries = make([]monitor.ConnEntry, 0)
	}
	writeJSON(w, entries)
}
