package dns

import (
	"net"
	"sync"

	"github.com/miekg/dns"
)

// ReverseDNS 는 IP→도메인 역방향 매핑을 유지한다.
// DNS 응답의 A/AAAA 레코드에서 자동으로 채워진다.
type ReverseDNS struct {
	m       map[string]string
	mu      sync.RWMutex
	maxSize int
}

func NewReverseDNS(maxSize int) *ReverseDNS {
	if maxSize <= 0 {
		maxSize = 50000
	}
	return &ReverseDNS{
		m:       make(map[string]string),
		maxSize: maxSize,
	}
}

// LearnFromResponse 는 DNS 응답에서 IP→도메인 매핑을 추출한다.
func (r *ReverseDNS) LearnFromResponse(msg *dns.Msg) {
	if msg == nil || len(msg.Question) == 0 {
		return
	}
	domain := normalizeDomain(msg.Question[0].Name)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rr := range msg.Answer {
		switch a := rr.(type) {
		case *dns.A:
			r.put(a.A.String(), domain)
		case *dns.AAAA:
			r.put(a.AAAA.String(), domain)
		case *dns.CNAME:
			// CNAME 타겟도 같은 도메인으로 매핑
			r.put(normalizeDomain(a.Target), domain)
		}
	}
}

func (r *ReverseDNS) put(ip, domain string) {
	if len(r.m) >= r.maxSize {
		// 가득 차면 절반 제거
		i := 0
		for k := range r.m {
			delete(r.m, k)
			i++
			if i >= r.maxSize/2 {
				break
			}
		}
	}
	r.m[ip] = domain
}

// Lookup 은 IP에 대응하는 도메인을 반환한다.
func (r *ReverseDNS) Lookup(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.m[ip]
}

// LookupMany 는 여러 IP를 한 번에 조회한다.
func (r *ReverseDNS) LookupMany(ips []string) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]string, len(ips))
	for _, ip := range ips {
		if domain, ok := r.m[ip]; ok {
			result[ip] = domain
		}
	}
	return result
}

// IsPrivate 는 사설 IP인지 확인한다.
func IsPrivate(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback()
}
