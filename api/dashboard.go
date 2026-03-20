package api

import "net/http"

type dashboardResponse struct {
	WANIP          string  `json:"wanIp"`
	ActiveLeases   int     `json:"activeLeases"`
	TotalQueries   int     `json:"totalQueries"`
	BlockedQueries int     `json:"blockedQueries"`
	BlockRate      float64 `json:"blockRate"`
	CacheHitRatio  float64 `json:"cacheHitRatio"`
	DNSEnabled     bool    `json:"dnsEnabled"`
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
