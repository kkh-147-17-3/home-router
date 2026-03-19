package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	Blocker  *Blocker
	Cache    *Cache
	QueryLog *QueryLog
	upstream []string
	listenAddr string
}

func NewServer(listenAddr string, upstream []string, blocker *Blocker, cache *Cache, queryLog *QueryLog) *Server {
	// 업스트림 주소에 포트가 없으면 :53 추가
	for i, addr := range upstream {
		if !strings.Contains(addr, ":") {
			upstream[i] = addr + ":53"
		}
	}

	return &Server{
		Blocker:    blocker,
		Cache:      cache,
		QueryLog:   queryLog,
		upstream:   upstream,
		listenAddr: listenAddr,
	}
}

func RunServer(ctx context.Context, listenAddr string, upstream []string, blocker *Blocker, cache *Cache, queryLog *QueryLog) error {
	s := NewServer(listenAddr, upstream, blocker, cache, queryLog)

	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleQuery)

	udpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: mux,
	}
	tcpServer := &dns.Server{
		Addr:    listenAddr,
		Net:     "tcp",
		Handler: mux,
	}

	errCh := make(chan error, 2)

	go func() {
		log.Printf("[DNS Server] UDP 서버 시작: %s", listenAddr)
		errCh <- udpServer.ListenAndServe()
	}()

	go func() {
		log.Printf("[DNS Server] TCP 서버 시작: %s", listenAddr)
		errCh <- tcpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		udpServer.Shutdown()
		tcpServer.Shutdown()
		return nil
	case err := <-errCh:
		udpServer.Shutdown()
		tcpServer.Shutdown()
		return fmt.Errorf("DNS 서버 오류: %w", err)
	}
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		return
	}

	start := time.Now()
	q := r.Question[0]
	domain := normalizeDomain(q.Name)
	qtype := dns.TypeToString[q.Qtype]

	clientIP := ""
	if addr, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		clientIP = addr.IP.String()
	} else if addr, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		clientIP = addr.IP.String()
	}

	// 1. 블록리스트 확인
	if s.Blocker.IsBlocked(domain) {
		msg := s.blockedResponse(r)
		w.WriteMsg(msg)

		s.QueryLog.Add(QueryEntry{
			Timestamp:    time.Now(),
			ClientIP:     clientIP,
			Domain:       domain,
			QueryType:    qtype,
			Blocked:      true,
			Cached:       false,
			ResponseTime: float64(time.Since(start).Microseconds()) / 1000,
		})
		return
	}

	// 2. 캐시 확인
	if cached := s.Cache.Get(q.Name, q.Qtype); cached != nil {
		cached.Id = r.Id
		w.WriteMsg(cached)

		s.QueryLog.Add(QueryEntry{
			Timestamp:    time.Now(),
			ClientIP:     clientIP,
			Domain:       domain,
			QueryType:    qtype,
			Blocked:      false,
			Cached:       true,
			ResponseTime: float64(time.Since(start).Microseconds()) / 1000,
		})
		return
	}

	// 3. 업스트림 포워딩
	resp, err := s.forwardQuery(r)
	if err != nil {
		log.Printf("[DNS Server] 업스트림 쿼리 실패 (%s %s): %v", domain, qtype, err)
		msg := new(dns.Msg)
		msg.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(msg)
		return
	}

	// 캐시 저장
	s.Cache.Put(q.Name, q.Qtype, resp)

	resp.Id = r.Id
	w.WriteMsg(resp)

	s.QueryLog.Add(QueryEntry{
		Timestamp:    time.Now(),
		ClientIP:     clientIP,
		Domain:       domain,
		QueryType:    qtype,
		Blocked:      false,
		Cached:       false,
		ResponseTime: float64(time.Since(start).Microseconds()) / 1000,
	})
}

func (s *Server) blockedResponse(r *dns.Msg) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	q := r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		msg.Answer = append(msg.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: net.IPv4(0, 0, 0, 0),
		})
	case dns.TypeAAAA:
		msg.Answer = append(msg.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			AAAA: net.IPv6zero,
		})
	default:
		msg.SetRcode(r, dns.RcodeNameError)
	}

	return msg
}

func (s *Server) forwardQuery(r *dns.Msg) (*dns.Msg, error) {
	client := &dns.Client{
		Timeout: 5 * time.Second,
	}

	for _, upstream := range s.upstream {
		resp, _, err := client.Exchange(r, upstream)
		if err == nil {
			return resp, nil
		}
		log.Printf("[DNS Server] 업스트림 %s 실패: %v, 다음 시도...", upstream, err)
	}

	return nil, fmt.Errorf("모든 업스트림 DNS 서버 응답 없음")
}
