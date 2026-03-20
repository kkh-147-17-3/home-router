package api

import "net/http"

type dashboardResponse struct {
	WANIP          string  `json:"wan_ip"`
	ActiveLeases   int     `json:"active_leases"`
	TotalQueries   int     `json:"total_queries"`
	BlockedQueries int     `json:"blocked_queries"`
	BlockRate      float64 `json:"block_rate"`
	CacheHitRatio  float64 `json:"cache_hit_ratio"`
	DNSEnabled     bool    `json:"dns_enabled"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	wanIP := s.wanIP
	s.mu.RUnlock()

	resp := dashboardResponse{
		WANIP:      wanIP,
		DNSEnabled: s.cfg.Dns.Enabled,
	}

	if s.pool != nil {
		info := s.pool.PoolInfo()
		resp.ActiveLeases = info.TotalLeases
	}

	if s.queryLog != nil {
		stats := s.queryLog.Stats()
		resp.TotalQueries = stats.TotalQueries
		resp.BlockedQueries = stats.BlockedQueries
		resp.BlockRate = stats.BlockPercentage
	}

	if s.cache != nil {
		cacheStats := s.cache.Stats()
		resp.CacheHitRatio = cacheStats.HitRatio
	}

	writeJSON(w, resp)
}
