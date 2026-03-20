package dhcp

import (
	"bytes"
	"encoding/json"
	"home-router/internal/config"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mdlayher/arp"
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

// leaseFile 전체 구조 (임대 + 충돌 IP)
type leaseFileData struct {
	Leases      []leaseRecord `json:"leases"`
	DeclinedIPs []string      `json:"declined_ips,omitempty"`
}

type Pool struct {
	RangeStart   net.IP
	RangeEnd     net.IP
	Leases       map[string]IPLease
	IPToMAC      map[string]string
	DeclinedIPs  map[string]bool
	StaticLeases map[string]net.IP // MAC → 고정 IP
	leaseFile    string
	lanIface     string
	mu           sync.RWMutex
}

func NewPool(rangeStart, rangeEnd net.IP, leaseFile string, staticLeases []config.StaticLeaseEntry, lanIface string) *Pool {
	p := &Pool{
		RangeStart:   rangeStart.To4(),
		RangeEnd:     rangeEnd.To4(),
		Leases:       make(map[string]IPLease),
		IPToMAC:      make(map[string]string),
		DeclinedIPs:  make(map[string]bool),
		StaticLeases: make(map[string]net.IP),
		leaseFile:    leaseFile,
		lanIface:     lanIface,
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

func (p *Pool) handleRelease(mac string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	lease, ok := p.Leases[mac]
	if !ok {
		return
	}
	ipStr := lease.Address.String()
	log.Printf("[DHCP Pool] RELEASE 처리: MAC=%s, IP=%s → 임대 해제", mac, ipStr)
	delete(p.Leases, mac)
	delete(p.IPToMAC, ipStr)
	p.saveLeases()
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

	// 2. 기존 임대 확인 → 만료시간 갱신
	lease, ok := p.Leases[mac]
	if ok {
		lease.ExpiredAt = time.Now().Add(time.Duration(cfg.Dhcp.Server.LeaseTime) * time.Second)
		p.Leases[mac] = lease
		log.Printf("[DHCP Pool] 기존 임대 반환: MAC=%s, IP=%s (만료: %s)", mac, lease.Address, lease.ExpiredAt.Format(time.RFC3339))
		p.saveLeases()
		return lease.Address
	}

	// 3. 새 IP 동적 할당 (ARP 프로브로 실제 사용 여부 확인)
	arpClient := p.newARPClient()
	if arpClient != nil {
		defer arpClient.Close()
	}
	for curr := p.RangeStart; curr != nil && bytes.Compare(curr, p.RangeEnd) <= 0; curr = GetNextIP(curr) {
		if _, ok := p.IPToMAC[curr.String()]; ok {
			continue
		}
		if p.DeclinedIPs[curr.String()] {
			continue
		}
		if arpClient != nil && p.isIPInUse(arpClient, curr) {
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

	// 새 형식(leaseFileData) 시도
	var fileData leaseFileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		// 구 형식([]leaseRecord) 호환
		var records []leaseRecord
		if err2 := json.Unmarshal(data, &records); err2 != nil {
			log.Printf("[DHCP Pool] 임대 파일 파싱 실패: %v", err)
			return
		}
		fileData.Leases = records
	}

	now := time.Now()
	restored := 0
	for _, r := range fileData.Leases {
		if r.ExpiredAt.Before(now) {
			continue
		}
		ip := net.ParseIP(r.IP).To4()
		if ip == nil {
			continue
		}
		if _, isStatic := p.StaticLeases[r.MAC]; isStatic {
			continue
		}
		p.Leases[r.MAC] = IPLease{Address: ip, ExpiredAt: r.ExpiredAt}
		p.IPToMAC[ip.String()] = r.MAC
		restored++
	}

	for _, ipStr := range fileData.DeclinedIPs {
		p.DeclinedIPs[ipStr] = true
	}

	log.Printf("[DHCP Pool] 임대 복원 완료: %d개, 충돌 IP: %d개 (파일: %s)", restored, len(fileData.DeclinedIPs), p.leaseFile)
}

// saveLeases 는 현재 임대 정보를 JSON 파일에 저장합니다.
// 호출 시 이미 mu.Lock이 잡혀있어야 합니다.
func (p *Pool) saveLeases() {
	if p.leaseFile == "" {
		return
	}

	records := make([]leaseRecord, 0, len(p.Leases))
	for mac, lease := range p.Leases {
		if _, isStatic := p.StaticLeases[mac]; isStatic {
			continue
		}
		records = append(records, leaseRecord{
			MAC:       mac,
			IP:        lease.Address.String(),
			ExpiredAt: lease.ExpiredAt,
		})
	}

	declinedIPs := make([]string, 0, len(p.DeclinedIPs))
	for ip := range p.DeclinedIPs {
		declinedIPs = append(declinedIPs, ip)
	}

	fileData := leaseFileData{
		Leases:      records,
		DeclinedIPs: declinedIPs,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		log.Printf("[DHCP Pool] 임대 파일 직렬화 실패: %v", err)
		return
	}

	dir := filepath.Dir(p.leaseFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[DHCP Pool] 임대 디렉토리 생성 실패: %v", err)
		return
	}

	if err := os.WriteFile(p.leaseFile, data, 0644); err != nil {
		log.Printf("[DHCP Pool] 임대 파일 저장 실패: %v", err)
	}
}

func (p *Pool) newARPClient() *arp.Client {
	if p.lanIface == "" {
		return nil
	}
	iface, err := net.InterfaceByName(p.lanIface)
	if err != nil {
		log.Printf("[DHCP Pool] ARP 클라이언트 생성 실패 (인터페이스): %v", err)
		return nil
	}
	client, err := arp.Dial(iface)
	if err != nil {
		log.Printf("[DHCP Pool] ARP 클라이언트 생성 실패: %v", err)
		return nil
	}
	return client
}

func (p *Pool) isIPInUse(client *arp.Client, ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	_ = client.SetDeadline(time.Now().Add(500 * time.Millisecond))
	hwAddr, err := client.Resolve(addr)
	if err != nil {
		return false
	}
	log.Printf("[DHCP Pool] ARP 프로브: IP=%s 사용 중 (MAC=%s), 건너뜀", ip, hwAddr)
	return true
}

// LeaseInfo 는 API 응답용 임대 정보 구조체
type LeaseInfo struct {
	MAC       string    `json:"mac"`
	IP        string    `json:"ip"`
	Hostname  string    `json:"hostname,omitempty"`
	ExpiredAt time.Time `json:"expired_at"`
	Static    bool      `json:"static"`
}

// PoolInfoResponse 는 풀 설정 정보 응답 구조체
type PoolInfoResponse struct {
	RangeStart    string `json:"range_start"`
	RangeEnd      string `json:"range_end"`
	TotalLeases   int    `json:"total_leases"`
	StaticLeases  int    `json:"static_leases"`
	DeclinedIPs   int    `json:"declined_ips"`
}

// ActiveLeases 는 전체 임대 스냅샷을 반환합니다.
func (p *Pool) ActiveLeases() []LeaseInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]LeaseInfo, 0, len(p.Leases))
	for mac, lease := range p.Leases {
		_, isStatic := p.StaticLeases[mac]
		result = append(result, LeaseInfo{
			MAC:       mac,
			IP:        lease.Address.String(),
			ExpiredAt: lease.ExpiredAt,
			Static:    isStatic,
		})
	}
	return result
}

// PoolInfo 는 풀 범위, 카운트 정보를 반환합니다.
func (p *Pool) PoolInfo() PoolInfoResponse {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolInfoResponse{
		RangeStart:   p.RangeStart.String(),
		RangeEnd:     p.RangeEnd.String(),
		TotalLeases:  len(p.Leases),
		StaticLeases: len(p.StaticLeases),
		DeclinedIPs:  len(p.DeclinedIPs),
	}
}

// StaticLeaseList 는 정적 임대 목록을 반환합니다.
func (p *Pool) StaticLeaseList() []LeaseInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]LeaseInfo, 0, len(p.StaticLeases))
	for mac, ip := range p.StaticLeases {
		result = append(result, LeaseInfo{
			MAC:    mac,
			IP:     ip.String(),
			Static: true,
		})
	}
	return result
}

// AddStaticLease 는 정적 임대를 추가합니다.
func (p *Pool) AddStaticLease(name, mac string, ip net.IP) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.StaticLeases[mac] = ip.To4()
	p.IPToMAC[ip.String()] = mac
	log.Printf("[DHCP Pool] 정적 임대 추가: %s → %s (%s)", mac, ip, name)
}

// RemoveStaticLease 는 정적 임대를 삭제합니다.
func (p *Pool) RemoveStaticLease(mac string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ip, ok := p.StaticLeases[mac]; ok {
		delete(p.IPToMAC, ip.String())
		delete(p.StaticLeases, mac)
		delete(p.Leases, mac)
		log.Printf("[DHCP Pool] 정적 임대 삭제: %s", mac)
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
