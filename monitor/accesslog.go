package monitor

import (
	"fmt"
	"sync"
	"time"
)

type AccessEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	SourceIP    string    `json:"sourceIp"`
	Country     string    `json:"country,omitempty"`
	CountryCode string    `json:"countryCode,omitempty"`
	Org         string    `json:"org,omitempty"`
	DestIP      string    `json:"destIp,omitempty"`
	DestPort    int       `json:"destPort"`
	PortName    string    `json:"portName,omitempty"`
	Protocol    string    `json:"protocol"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
}

type AccessLog struct {
	entries     []AccessEntry
	pos         int
	count       int
	maxSize     int
	mu          sync.Mutex
	subscribers map[chan AccessEntry]struct{}
	subMu       sync.RWMutex
}

func NewAccessLog(maxSize int) *AccessLog {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &AccessLog{
		entries:     make([]AccessEntry, maxSize),
		maxSize:     maxSize,
		subscribers: make(map[chan AccessEntry]struct{}),
	}
}

func (a *AccessLog) Add(entry AccessEntry) {
	a.mu.Lock()
	a.entries[a.pos] = entry
	a.pos = (a.pos + 1) % a.maxSize
	if a.count < a.maxSize {
		a.count++
	}
	a.mu.Unlock()

	a.subMu.RLock()
	for ch := range a.subscribers {
		select {
		case ch <- entry:
		default:
		}
	}
	a.subMu.RUnlock()
}

func (a *AccessLog) Subscribe() chan AccessEntry {
	ch := make(chan AccessEntry, 64)
	a.subMu.Lock()
	a.subscribers[ch] = struct{}{}
	a.subMu.Unlock()
	return ch
}

func (a *AccessLog) Unsubscribe(ch chan AccessEntry) {
	a.subMu.Lock()
	delete(a.subscribers, ch)
	a.subMu.Unlock()
	close(ch)
}

func (a *AccessLog) Recent(n int) []AccessEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	if n > a.count {
		n = a.count
	}
	if n == 0 {
		return nil
	}

	result := make([]AccessEntry, n)
	for i := 0; i < n; i++ {
		idx := (a.pos - 1 - i + a.maxSize) % a.maxSize
		result[i] = a.entries[idx]
	}
	return result
}

// Filter returns recent entries matching optional filters.
func (a *AccessLog) Filter(n int, sourceIP string, destPort int) []AccessEntry {
	entries := a.Recent(n)
	if sourceIP == "" && destPort == 0 {
		return entries
	}
	filtered := make([]AccessEntry, 0, len(entries))
	for _, e := range entries {
		if sourceIP != "" && e.SourceIP != sourceIP {
			continue
		}
		if destPort != 0 && e.DestPort != destPort {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

type TopItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type AccessStats struct {
	TotalEvents    int            `json:"totalEvents"`
	UniqueSourceIP int            `json:"uniqueSourceIps"`
	TopSourceIPs   []TopItem      `json:"topSourceIps"`
	TopPorts       []TopItem      `json:"topPorts"`
	Hourly         map[int]int    `json:"hourly"`
}

func (a *AccessLog) Stats() AccessStats {
	a.mu.Lock()
	defer a.mu.Unlock()

	sourceIPs := make(map[string]int)
	ports := make(map[string]int)
	hourly := make(map[int]int)

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	total := 0

	for i := 0; i < a.count; i++ {
		idx := (a.pos - 1 - i + a.maxSize) % a.maxSize
		entry := a.entries[idx]
		if entry.Timestamp.Before(dayAgo) {
			continue
		}
		total++
		sourceIPs[entry.SourceIP]++
		ports[fmt.Sprintf("%d/%s", entry.DestPort, entry.Protocol)]++
		hourly[entry.Timestamp.Hour()]++
	}

	return AccessStats{
		TotalEvents:    total,
		UniqueSourceIP: len(sourceIPs),
		TopSourceIPs:   topNItems(sourceIPs, 10),
		TopPorts:       topNItems(ports, 10),
		Hourly:         hourly,
	}
}

func topNItems(m map[string]int, n int) []TopItem {
	result := make([]TopItem, 0, len(m))
	for name, count := range m {
		result = append(result, TopItem{Name: name, Count: count})
	}
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
