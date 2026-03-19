package dns

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

type cacheEntry struct {
	msg       *dns.Msg
	expiresAt time.Time
}

type Cache struct {
	entries map[string]cacheEntry
	maxSize int
	mu      sync.RWMutex

	hits   atomic.Uint64
	misses atomic.Uint64
}

func NewCache(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &Cache{
		entries: make(map[string]cacheEntry),
		maxSize: maxSize,
	}
}

func (c *Cache) Get(name string, qtype uint16) *dns.Msg {
	key := cacheKey(name, qtype)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return nil
	}

	c.hits.Add(1)
	return entry.msg.Copy()
}

func (c *Cache) Put(name string, qtype uint16, msg *dns.Msg) {
	if msg == nil {
		return
	}

	// 응답의 최소 TTL을 캐시 TTL로 사용
	ttl := minTTL(msg)
	if ttl == 0 {
		return
	}

	key := cacheKey(name, qtype)

	c.mu.Lock()
	defer c.mu.Unlock()

	// 캐시 용량 초과 시 만료된 엔트리 정리
	if len(c.entries) >= c.maxSize {
		c.evictExpired()
	}

	// 그래도 가득 차면 가장 오래된 10% 제거
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = cacheEntry{
		msg:       msg.Copy(),
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
}

func (c *Cache) Stats() CacheStats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	var hitRatio float64
	if total > 0 {
		hitRatio = float64(hits) / float64(total) * 100
	}

	c.mu.RLock()
	size := len(c.entries)
	c.mu.RUnlock()

	return CacheStats{
		Size:     size,
		MaxSize:  c.maxSize,
		Hits:     hits,
		Misses:   misses,
		HitRatio: hitRatio,
	}
}

type CacheStats struct {
	Size     int     `json:"size"`
	MaxSize  int     `json:"max_size"`
	Hits     uint64  `json:"hits"`
	Misses   uint64  `json:"misses"`
	HitRatio float64 `json:"hit_ratio"`
}

func cacheKey(name string, qtype uint16) string {
	return name + "|" + dns.TypeToString[qtype]
}

func minTTL(msg *dns.Msg) uint32 {
	var ttl uint32 = 3600 // 기본값 1시간

	for _, rr := range msg.Answer {
		if rr.Header().Ttl < ttl {
			ttl = rr.Header().Ttl
		}
	}
	for _, rr := range msg.Ns {
		if rr.Header().Ttl < ttl {
			ttl = rr.Header().Ttl
		}
	}

	return ttl
}

func (c *Cache) evictExpired() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) evictOldest() {
	target := len(c.entries) / 10
	if target == 0 {
		target = 1
	}

	removed := 0
	for key := range c.entries {
		delete(c.entries, key)
		removed++
		if removed >= target {
			break
		}
	}
}
