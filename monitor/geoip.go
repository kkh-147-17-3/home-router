package monitor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type GeoInfo struct {
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

// GeoIPCache 는 IP별 국가 정보를 캐싱하는 GeoIP 조회기이다.
type GeoIPCache struct {
	cache  map[string]GeoInfo
	mu     sync.RWMutex
	client *http.Client
}

func NewGeoIPCache() *GeoIPCache {
	return &GeoIPCache{
		cache: make(map[string]GeoInfo),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Lookup 은 IP의 국가 정보를 반환한다. 캐시에 있으면 즉시 반환, 없으면 API 조회.
func (g *GeoIPCache) Lookup(ip string) GeoInfo {
	// 사설 IP는 조회하지 않음
	if isPrivateIP(ip) {
		return GeoInfo{Country: "Private", CountryCode: "XX"}
	}

	g.mu.RLock()
	info, ok := g.cache[ip]
	g.mu.RUnlock()
	if ok {
		return info
	}

	info = g.fetchFromAPI(ip)

	g.mu.Lock()
	g.cache[ip] = info
	g.mu.Unlock()

	return info
}

// LookupAsync 는 비동기로 GeoIP를 조회하고 캐시에 저장한다.
// 조회 결과를 기다리지 않는다.
func (g *GeoIPCache) LookupAsync(ip string) {
	if isPrivateIP(ip) {
		return
	}

	g.mu.RLock()
	_, ok := g.cache[ip]
	g.mu.RUnlock()
	if ok {
		return
	}

	go func() {
		info := g.fetchFromAPI(ip)
		g.mu.Lock()
		g.cache[ip] = info
		g.mu.Unlock()
	}()
}

type ipAPIResponse struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
}

func (g *GeoIPCache) fetchFromAPI(ip string) GeoInfo {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode", ip)
	resp, err := g.client.Get(url)
	if err != nil {
		return GeoInfo{Country: "Unknown", CountryCode: "??"}
	}
	defer resp.Body.Close()

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Status != "success" {
		return GeoInfo{Country: "Unknown", CountryCode: "??"}
	}

	return GeoInfo{
		Country:     result.Country,
		CountryCode: result.CountryCode,
	}
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
