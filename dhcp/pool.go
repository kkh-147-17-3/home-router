package dhcp

import (
	"bytes"
	"home-router/internal/config"
	"log"
	"net"
	"sync"
	"time"
)

type IPLease struct {
	Address   net.IP
	ExpiredAt time.Time
}

type Pool struct {
	RangeStart  net.IP
	RangeEnd    net.IP
	Leases      map[string]IPLease
	IPToMAC     map[string]string
	DeclinedIPs map[string]bool
	mu          sync.RWMutex
}

func NewPool(rangeStart, rangeEnd net.IP) *Pool {
	return &Pool{
		RangeStart:  rangeStart.To4(),
		RangeEnd:    rangeEnd.To4(),
		Leases:      make(map[string]IPLease),
		IPToMAC:     make(map[string]string),
		DeclinedIPs: make(map[string]bool),
	}
}

func (p *Pool) handleDecline(mac string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	lease, ok := p.Leases[mac]
	if !ok {
		return
	}
	ipStr := lease.Address.String()
	log.Printf("[DHCP Pool] DECLINE 처리: MAC=%s, IP=%s → 충돌 IP로 등록", mac, ipStr)
	delete(p.Leases, mac)
	delete(p.IPToMAC, ipStr)
	p.DeclinedIPs[ipStr] = true
}

func (p *Pool) handleClientRequest(mac string, cfg *config.Config) net.IP {
	p.mu.Lock()
	defer p.mu.Unlock()

	lease, ok := p.Leases[mac]
	if ok {
		if lease.ExpiredAt.After(time.Now()) {
			log.Printf("[DHCP Pool] 기존 임대 반환: MAC=%s, IP=%s (만료: %s)", mac, lease.Address, lease.ExpiredAt.Format(time.RFC3339))
			return lease.Address
		} else {
			log.Printf("[DHCP Pool] 임대 만료 제거: MAC=%s, IP=%s", mac, lease.Address)
			delete(p.Leases, mac)
			delete(p.IPToMAC, lease.Address.String())
		}
	}

	for curr := p.RangeStart; bytes.Compare(curr, p.RangeEnd) <= 0; curr = GetNextIP(curr) {
		if _, ok := p.IPToMAC[curr.String()]; ok {
			continue
		}
		if p.DeclinedIPs[curr.String()] {
			continue
		}
		expiredAt := time.Now().Add(time.Duration(cfg.Dhcp.Server.LeaseTime) * time.Second)
		allocatedIP := make(net.IP, len(curr))
		copy(allocatedIP, curr)
		p.Leases[mac] = IPLease{
			allocatedIP,
			expiredAt,
		}
		p.IPToMAC[curr.String()] = mac
		log.Printf("[DHCP Pool] 새 IP 할당: MAC=%s, IP=%s (만료: %s) [현재 총 %d개 임대]", mac, allocatedIP, expiredAt.Format(time.RFC3339), len(p.Leases))
		return allocatedIP
	}
	log.Printf("[DHCP Pool] IP 풀 소진: MAC=%s 에 할당할 IP 없음", mac)
	return nil
}

func GetNextIP(ip net.IP) net.IP {
	src := ip.To4()
	next := make(net.IP, 4)
	copy(next, src)

	newVal := int16(next[3]) + 1
	if newVal >= 255 {
		next[2]++
		next[3] = 0
	} else {
		next[3] = byte(newVal)
	}
	for i := 2; i >= 0; i-- {
		if next[i] >= 255 {
			next[i-1]++
			next[i] = 0
		}
	}

	return next
}
