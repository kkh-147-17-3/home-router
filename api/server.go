package api

import (
	"context"
	"home-router/dhcp"
	hdns "home-router/dns"
	"home-router/internal/config"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	pool      *dhcp.Pool
	cache     *hdns.Cache
	queryLog  *hdns.QueryLog
	blocker   *hdns.Blocker
	cfg       *config.Config
	startTime time.Time
	wanIP     string
	wanIface  string
	lanIface  string
	mu        sync.RWMutex // wanIP 보호
	sessions  map[string]time.Time
	sessMu    sync.RWMutex
	staticFS  fs.FS
}

func NewServer(cfg *config.Config, pool *dhcp.Pool, cache *hdns.Cache,
	queryLog *hdns.QueryLog, blocker *hdns.Blocker,
	wanIface, lanIface string, staticFS fs.FS) *Server {
	return &Server{
		pool:      pool,
		cache:     cache,
		queryLog:  queryLog,
		blocker:   blocker,
		cfg:       cfg,
		startTime: time.Now(),
		wanIface:  wanIface,
		lanIface:  lanIface,
		sessions:  make(map[string]time.Time),
		staticFS:  staticFS,
	}
}

func (s *Server) SetWANIP(ip string) {
	s.mu.Lock()
	s.wanIP = ip
	s.mu.Unlock()
}

func (s *Server) Start(ctx context.Context, addr string) {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

	mux.HandleFunc("GET /api/dashboard", s.auth(s.handleDashboard))

	mux.HandleFunc("GET /api/dhcp/leases", s.auth(s.handleDHCPLeases))
	mux.HandleFunc("GET /api/dhcp/pool", s.auth(s.handleDHCPPool))
	mux.HandleFunc("GET /api/dhcp/static-leases", s.auth(s.handleDHCPStaticLeases))
	mux.HandleFunc("POST /api/dhcp/static-leases", s.auth(s.handleDHCPAddStaticLease))
	mux.HandleFunc("DELETE /api/dhcp/static-leases/{mac}", s.auth(s.handleDHCPRemoveStaticLease))

	mux.HandleFunc("GET /api/dns/stats", s.auth(s.handleDNSStats))
	mux.HandleFunc("GET /api/dns/querylog", s.auth(s.handleDNSQueryLog))
	mux.HandleFunc("GET /api/dns/cache/stats", s.auth(s.handleDNSCacheStats))
	mux.HandleFunc("GET /api/dns/blocker/stats", s.auth(s.handleDNSBlockerStats))
	mux.HandleFunc("POST /api/dns/blocker/reload", s.auth(s.handleDNSBlockerReload))
	mux.HandleFunc("GET /api/dns/blocker/whitelist", s.auth(s.handleDNSWhitelist))
	mux.HandleFunc("POST /api/dns/blocker/whitelist", s.auth(s.handleDNSAddWhitelist))
	mux.HandleFunc("DELETE /api/dns/blocker/whitelist/{domain}", s.auth(s.handleDNSRemoveWhitelist))

	mux.HandleFunc("GET /api/nat/port-forwards", s.auth(s.handleNATPortForwards))
	mux.HandleFunc("POST /api/nat/port-forwards", s.auth(s.handleNATAddPortForward))
	mux.HandleFunc("DELETE /api/nat/port-forwards/{name}", s.auth(s.handleNATRemovePortForward))

	mux.HandleFunc("GET /api/network/wan", s.auth(s.handleNetworkWAN))
	mux.HandleFunc("GET /api/network/lan", s.auth(s.handleNetworkLAN))

	mux.HandleFunc("GET /api/system/config", s.auth(s.handleSystemConfig))
	mux.HandleFunc("GET /api/system/uptime", s.auth(s.handleSystemUptime))
	mux.HandleFunc("GET /api/system/logs", s.auth(s.handleSystemLogs))

	mux.HandleFunc("GET /api/sse/dns-querylog", s.auth(s.handleSSEDNSQueryLog))
	mux.HandleFunc("GET /api/sse/system-logs", s.auth(s.handleSSESystemLogs))

	// SPA — embedded frontend
	if s.staticFS != nil {
		mux.Handle("/", spaHandler{fs: http.FileServerFS(s.staticFS), staticFS: s.staticFS})
	}

	server := &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutCtx)
	}()

	log.Printf("[Web UI] HTTP 서버 시작: %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("[Web UI] HTTP 서버 오류: %v", err)
	}
}

// spaHandler serves static files and falls back to index.html for SPA routing
type spaHandler struct {
	fs       http.Handler
	staticFS fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to serve the file directly
	path := r.URL.Path
	if path == "/" {
		path = "index.html"
	} else {
		path = path[1:] // remove leading /
	}

	// Check if file exists
	if _, err := fs.Stat(h.staticFS, path); err == nil {
		h.fs.ServeHTTP(w, r)
		return
	}

	// Fall back to index.html for SPA routing
	r.URL.Path = "/"
	h.fs.ServeHTTP(w, r)
}
