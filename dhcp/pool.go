package dhcp

import (
	"bytes"
	"home-router/internal/config"
	"net"
	"sync"
	"time"
)

type IPLease struct {
	Address   net.IP
	ExpiredAt time.Time
}

type Pool struct {
	RangeStart net.IP
	RangeEnd   net.IP
	Leases     map[string]IPLease
	IPToMAC    map[string]string
	mu         sync.RWMutex // ← map 전체 락
}

func NewPool(rangeStart, rangeEnd net.IP) *Pool {
	leases := make(map[string]IPLease)
	ipToMAC := make(map[string]string)

	return &Pool{
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		Leases:     leases,
		IPToMAC:    ipToMAC,
		mu:         sync.RWMutex{},
	}
}

func (p *Pool) handleClientRequest(mac string, cfg *config.Config) net.IP {
	p.mu.Lock()
	defer p.mu.Unlock()

	lease, ok := p.Leases[mac]
	if ok {
		if lease.ExpiredAt.After(time.Now()) {
			return lease.Address
		} else {
			delete(p.Leases, mac)
			delete(p.IPToMAC, lease.Address.String())
		}
	}

	for curr := p.RangeStart; bytes.Compare(curr, p.RangeEnd) <= 0; curr = GetNextIP(curr) {
		if _, ok := p.IPToMAC[curr.String()]; ok {
			continue
		}
		p.Leases[mac] = IPLease{
			curr,
			time.Now().Add(time.Duration(cfg.Dhcp.Server.LeaseTime) * time.Second),
		}
		p.IPToMAC[curr.String()] = mac
		return curr
	}
	return nil
}

func GetNextIP(ip net.IP) net.IP {
	next := ip.To4()

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
