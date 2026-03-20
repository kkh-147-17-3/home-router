package dns

import (
	"sync"
	"time"
)

type QueryEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	ClientIP     string    `json:"clientIp"`
	Domain       string    `json:"domain"`
	QueryType    string    `json:"queryType"`
	Blocked      bool      `json:"blocked"`
	Cached       bool      `json:"cached"`
	ResponseTime float64   `json:"responseTimeMs"`
}

type QueryLog struct {
	entries     []QueryEntry
	pos         int
	count       int
	maxSize     int
	mu          sync.Mutex
	subscribers map[chan QueryEntry]struct{}
	subMu       sync.RWMutex
}

func NewQueryLog(maxSize int) *QueryLog {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &QueryLog{
		entries:     make([]QueryEntry, maxSize),
		maxSize:     maxSize,
		subscribers: make(map[chan QueryEntry]struct{}),
	}
}

func (q *QueryLog) Add(entry QueryEntry) {
	q.mu.Lock()
	q.entries[q.pos] = entry
	q.pos = (q.pos + 1) % q.maxSize
	if q.count < q.maxSize {
		q.count++
	}
	q.mu.Unlock()

	// 구독자에게 브로드캐스트
	q.subMu.RLock()
	for ch := range q.subscribers {
		select {
		case ch <- entry:
		default: // 버퍼 가득 차면 skip (느린 클라이언트 보호)
		}
	}
	q.subMu.RUnlock()
}

// Subscribe 는 새 쿼리 엔트리를 수신하는 채널을 생성합니다.
func (q *QueryLog) Subscribe() chan QueryEntry {
	ch := make(chan QueryEntry, 64)
	q.subMu.Lock()
	q.subscribers[ch] = struct{}{}
	q.subMu.Unlock()
	return ch
}

// Unsubscribe 는 구독을 해제합니다.
func (q *QueryLog) Unsubscribe(ch chan QueryEntry) {
	q.subMu.Lock()
	delete(q.subscribers, ch)
	q.subMu.Unlock()
	close(ch)
}

// Recent 는 가장 최근 n개의 쿼리를 최신순으로 반환합니다.
func (q *QueryLog) Recent(n int) []QueryEntry {
	q.mu.Lock()
	defer q.mu.Unlock()

	if n > q.count {
		n = q.count
	}
	if n == 0 {
		return nil
	}

	result := make([]QueryEntry, n)
	for i := 0; i < n; i++ {
		idx := (q.pos - 1 - i + q.maxSize) % q.maxSize
		result[i] = q.entries[idx]
	}

	return result
}

func (q *QueryLog) Stats() QueryLogStats {
	q.mu.Lock()
	defer q.mu.Unlock()

	var totalQueries, blockedQueries, cachedQueries int
	topBlocked := make(map[string]int)
	topClients := make(map[string]int)
	hourly := make(map[int]int) // hour(0-23) → count

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)

	for i := 0; i < q.count; i++ {
		idx := (q.pos - 1 - i + q.maxSize) % q.maxSize
		entry := q.entries[idx]

		// 24시간 이내만 통계에 포함
		if entry.Timestamp.Before(dayAgo) {
			continue
		}

		totalQueries++
		if entry.Blocked {
			blockedQueries++
			topBlocked[entry.Domain]++
		}
		if entry.Cached {
			cachedQueries++
		}
		topClients[entry.ClientIP]++
		hourly[entry.Timestamp.Hour()]++
	}

	var blockPercentage float64
	if totalQueries > 0 {
		blockPercentage = float64(blockedQueries) / float64(totalQueries) * 100
	}

	return QueryLogStats{
		TotalQueries:    totalQueries,
		BlockedQueries:  blockedQueries,
		CachedQueries:   cachedQueries,
		BlockPercentage: blockPercentage,
		TopBlocked:      topN(topBlocked, 10),
		TopClients:      topN(topClients, 10),
		Hourly:          hourly,
	}
}

type QueryLogStats struct {
	TotalQueries    int         `json:"totalQueries"`
	BlockedQueries  int         `json:"blockedQueries"`
	CachedQueries   int         `json:"cachedQueries"`
	BlockPercentage float64     `json:"blockPercentage"`
	TopBlocked      []TopEntry  `json:"topBlocked"`
	TopClients      []TopEntry  `json:"topClients"`
	Hourly          map[int]int `json:"hourly"`
}

type TopEntry struct {
	Name  string `json:"name"`
	Label string `json:"label,omitempty"`
	Count int    `json:"count"`
}

func topN(m map[string]int, n int) []TopEntry {
	result := make([]TopEntry, 0, len(m))
	for name, count := range m {
		result = append(result, TopEntry{Name: name, Count: count})
	}

	// 간단한 정렬 (n이 작으므로 selection sort)
	for i := 0; i < len(result) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(result); j++ {
			if result[j].Count > result[maxIdx].Count {
				maxIdx = j
			}
		}
		result[i], result[maxIdx] = result[maxIdx], result[i]
	}

	if len(result) > n {
		result = result[:n]
	}
	return result
}
