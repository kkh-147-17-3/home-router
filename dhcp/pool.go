package dhcp

import (
	"bytes"
	"encoding/json"
	"home-router/internal/config"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type IPLease struct {
	Address   net.IP
	ExpiredAt time.Time
}

// leaseRecord 는 JSON 영속화용 구조체
type leaseRecord struct {
	MAC       string    `json:"mac"`
	IP        string    `json:"ip"`
	ExpiredAt time.Time `json:"expired_at"`
}

type Pool struct {
	RangeStart   net.IP
	RangeEnd     net.IP
	Leases       map[string]IPLease
	IPToMAC      map[string]string
	DeclinedIPs  map[string]bool
	StaticLeases map[string]net.IP // MAC → 고정 IP
	leaseFile    string
	mu           sync.RWMutex
}

func NewPool(rangeStart, rangeEnd net.IP, leaseFile string, staticLeases []config.StaticLeaseEntry) *Pool {
	p := &Pool{
		RangeStart:   rangeStart.To4(),
		RangeEnd:     rangeEnd.To4(),
		Leases:       make(map[string]IPLease),
		IPToMAC:      make(map[string]string),
		DeclinedIPs:  make(map[string]bool),
		StaticLeases: make(map[string]net.IP),
		leaseFile:    leaseFile,
	}

	// Static lease 등록
	for _, s := range staticLeases {
		ip := net.ParseIP(s.IP).To4()
		if ip == nil {
			log.Printf("[DHCP Pool] 고정 임대 IP 파싱 실패: %s (%s)", s.IP, s.Name)
			continue
		}
		p.StaticLeases[s.MacAddress] = ip
		p.IPToMAC[ip.String()] = s.MacAddress
		log.Printf("[DHCP Pool] 고정 임대 등록: %s → %s (%s)", s.MacAddress, ip, s.Name)
	}

	// 임대 파일에서 복원
	if leaseFile != "" {
		p.loadLeases()
	}

	return p
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
	p.saveLeases()
}

func (p *Pool) cleanExpiredLeases() {
	now := time.Now()
	for mac, lease := range p.Leases {
		if lease.ExpiredAt.Before(now) {
			log.Printf("[DHCP Pool] 만료 임대 정리: MAC=%s, IP=%s", mac, lease.Address)
			delete(p.Leases, mac)
			delete(p.IPToMAC, lease.Address.String())
		}
	}
}

func (p *Pool) handleClientRequest(mac string, cfg *config.Config) net.IP {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 만료된 임대 일괄 정리
	p.cleanExpiredLeases()

	// 1. 고정 임대 확인
	if staticIP, ok := p.StaticLeases[mac]; ok {
		// 고정 임대는 만료 없이 최대 lease time으로 설정
		expiredAt := time.Now().Add(time.Duration(cfg.Dhcp.Server.LeaseTime) * time.Second)
		p.Leases[mac] = IPLease{staticIP, expiredAt}
		p.IPToMAC[staticIP.String()] = mac
		log.Printf("[DHCP Pool] 고정 임대 반환: MAC=%s, IP=%s", mac, staticIP)
		p.saveLeases()
		return staticIP
	}

	// 2. 기존 임대 확인
	lease, ok := p.Leases[mac]
	if ok {
		log.Printf("[DHCP Pool] 기존 임대 반환: MAC=%s, IP=%s (만료: %s)", mac, lease.Address, lease.ExpiredAt.Format(time.RFC3339))
		return lease.Address
	}

	// 3. 새 IP 동적 할당
	for curr := p.RangeStart; curr != nil && bytes.Compare(curr, p.RangeEnd) <= 0; curr = GetNextIP(curr) {
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
		p.saveLeases()
		return allocatedIP
	}
	log.Printf("[DHCP Pool] IP 풀 소진: MAC=%s 에 할당할 IP 없음", mac)
	return nil
}

// loadLeases 는 JSON 파일에서 임대 정보를 복원합니다.
func (p *Pool) loadLeases() {
	data, err := os.ReadFile(p.leaseFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[DHCP Pool] 임대 파일 없음, 빈 상태로 시작: %s", p.leaseFile)
			return
		}
		log.Printf("[DHCP Pool] 임대 파일 읽기 실패: %v", err)
		return
	}

	var records []leaseRecord
	if err := json.Unmarshal(data, &records); err != nil {
		log.Printf("[DHCP Pool] 임대 파일 파싱 실패: %v", err)
		return
	}

	now := time.Now()
	restored := 0
	for _, r := range records {
		// 만료된 임대는 건너뛰기
		if r.ExpiredAt.Before(now) {
			continue
		}
		ip := net.ParseIP(r.IP).To4()
		if ip == nil {
			continue
		}
		// 고정 임대로 이미 등록된 IP는 건너뛰기
		if _, isStatic := p.StaticLeases[r.MAC]; isStatic {
			continue
		}
		p.Leases[r.MAC] = IPLease{Address: ip, ExpiredAt: r.ExpiredAt}
		p.IPToMAC[ip.String()] = r.MAC
		restored++
	}

	log.Printf("[DHCP Pool] 임대 복원 완료: %d개 (파일: %s)", restored, p.leaseFile)
}

// saveLeases 는 현재 임대 정보를 JSON 파일에 저장합니다.
// 호출 시 이미 mu.Lock이 잡혀있어야 합니다.
func (p *Pool) saveLeases() {
	if p.leaseFile == "" {
		return
	}

	records := make([]leaseRecord, 0, len(p.Leases))
	for mac, lease := range p.Leases {
		// 고정 임대는 파일에 저장하지 않음 (config에서 관리)
		if _, isStatic := p.StaticLeases[mac]; isStatic {
			continue
		}
		records = append(records, leaseRecord{
			MAC:       mac,
			IP:        lease.Address.String(),
			ExpiredAt: lease.ExpiredAt,
		})
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		log.Printf("[DHCP Pool] 임대 파일 직렬화 실패: %v", err)
		return
	}

	// 디렉토리가 없으면 생성
	dir := filepath.Dir(p.leaseFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[DHCP Pool] 임대 디렉토리 생성 실패: %v", err)
		return
	}

	if err := os.WriteFile(p.leaseFile, data, 0644); err != nil {
		log.Printf("[DHCP Pool] 임대 파일 저장 실패: %v", err)
	}
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
			if i == 0 {
				return nil
			}
			next[i-1]++
			next[i] = 0
		}
	}

	return next
}
