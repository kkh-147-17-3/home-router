package monitor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ConnEntry represents a single conntrack connection.
type ConnEntry struct {
	Protocol  string `json:"protocol"`
	State     string `json:"state"`
	SrcIP     string `json:"srcIp"`
	DstIP     string `json:"dstIp"`
	DstDomain string `json:"dstDomain,omitempty"`
	SrcPort   int    `json:"srcPort"`
	DstPort   int    `json:"dstPort"`
	BytesSent int64  `json:"bytesSent"`
	BytesRecv int64  `json:"bytesRecv"`
}

// HostTraffic aggregates traffic per internal host.
type HostTraffic struct {
	IP          string         `json:"ip"`
	Hostname    string         `json:"hostname,omitempty"`
	BytesSent   int64          `json:"bytesSent"`
	BytesRecv   int64          `json:"bytesRecv"`
	Connections int            `json:"connections"`
	TopDests    []EndpointStat `json:"topDestinations"`
}

// EndpointStat aggregates traffic to a single destination.
type EndpointStat struct {
	IP          string `json:"ip"`
	Domain      string `json:"domain,omitempty"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	BytesSent   int64  `json:"bytesSent"`
	BytesRecv   int64  `json:"bytesRecv"`
	Connections int    `json:"connections"`
}

// TrafficSummary is the top-level traffic overview.
type TrafficSummary struct {
	TotalSent       int64          `json:"totalSent"`
	TotalRecv       int64          `json:"totalRecv"`
	TotalConns      int            `json:"totalConnections"`
	Hosts           []HostTraffic  `json:"hosts"`
	TopDestinations []EndpointStat `json:"topDestinations"`
}

var stateRe = regexp.MustCompile(`\b(ESTABLISHED|TIME_WAIT|CLOSE_WAIT|SYN_SENT|SYN_RECV|FIN_WAIT|CLOSE|LAST_ACK)\b`)

// EnableConntrackAcct enables byte/packet counters in conntrack.
// Should be called once at startup (requires root).
func EnableConntrackAcct() {
	os.WriteFile("/proc/sys/net/netfilter/nf_conntrack_acct", []byte("1"), 0644)
}

// ReadConntrack reads conntrack entries from proc or falls back to conntrack CLI.
func ReadConntrack(lanSubnet string) ([]ConnEntry, error) {
	_, lanNet, err := net.ParseCIDR(lanSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid LAN subnet: %w", err)
	}

	lines, err := readConntrackLines()
	if err != nil {
		return nil, err
	}

	var entries []ConnEntry
	for _, line := range lines {
		if entry, ok := parseConnLine(line, lanNet); ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// readConntrackLines tries /proc/net/nf_conntrack first, then conntrack -L.
func readConntrackLines() ([]string, error) {
	// Try proc file first
	data, err := os.ReadFile("/proc/net/nf_conntrack")
	if err == nil {
		return strings.Split(string(data), "\n"), nil
	}

	// Fallback: conntrack -L
	out, err := exec.Command("conntrack", "-L", "-o", "extended").Output()
	if err != nil {
		// Last resort: read from /proc/net/ip_conntrack (older kernels)
		data, err2 := os.ReadFile("/proc/net/ip_conntrack")
		if err2 == nil {
			return strings.Split(string(data), "\n"), nil
		}
		return nil, fmt.Errorf("conntrack unavailable: proc=%v, cli=%v", err, err2)
	}
	return strings.Split(string(out), "\n"), nil
}

func parseConnLine(line string, lanNet *net.IPNet) (ConnEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ConnEntry{}, false
	}

	fields := strings.Fields(line)
	if len(fields) < 6 {
		return ConnEntry{}, false
	}

	// Detect protocol: find tcp/udp/icmp in first few fields
	var proto string
	for _, f := range fields[:min(5, len(fields))] {
		switch f {
		case "tcp", "udp", "icmp":
			proto = f
		}
	}
	if proto == "" {
		return ConnEntry{}, false
	}

	state := ""
	if m := stateRe.FindString(line); m != "" {
		state = m
	}

	// Extract key=value pairs; conntrack has two direction halves
	kv := make(map[string][]string)
	for _, f := range fields {
		if idx := strings.Index(f, "="); idx > 0 {
			key := f[:idx]
			val := f[idx+1:]
			kv[key] = append(kv[key], val)
		}
	}

	srcIPs := kv["src"]
	dstIPs := kv["dst"]
	sports := kv["sport"]
	dports := kv["dport"]
	bytesVals := kv["bytes"]
	packetsVals := kv["packets"]

	if len(srcIPs) < 2 || len(dstIPs) < 2 {
		return ConnEntry{}, false
	}

	// Original direction: srcIPs[0] → dstIPs[0]
	srcIP := srcIPs[0]
	dstIP := dstIPs[0]

	// Only include connections originating from LAN
	if !lanNet.Contains(net.ParseIP(srcIP)) {
		return ConnEntry{}, false
	}

	// Skip LAN-to-LAN traffic
	if lanNet.Contains(net.ParseIP(dstIP)) {
		return ConnEntry{}, false
	}

	var srcPort, dstPort int
	if len(sports) > 0 {
		srcPort, _ = strconv.Atoi(sports[0])
	}
	if len(dports) > 0 {
		dstPort, _ = strconv.Atoi(dports[0])
	}

	var bytesSent, bytesRecv int64
	if len(bytesVals) >= 1 {
		bytesSent, _ = strconv.ParseInt(bytesVals[0], 10, 64)
	}
	if len(bytesVals) >= 2 {
		bytesRecv, _ = strconv.ParseInt(bytesVals[1], 10, 64)
	}

	// If no byte counters (acct disabled), estimate from packets
	if bytesSent == 0 && bytesRecv == 0 && len(packetsVals) >= 2 {
		pktSent, _ := strconv.ParseInt(packetsVals[0], 10, 64)
		pktRecv, _ := strconv.ParseInt(packetsVals[1], 10, 64)
		// Use packet counts as fallback indicator
		bytesSent = pktSent
		bytesRecv = pktRecv
	}

	return ConnEntry{
		Protocol:  proto,
		State:     state,
		SrcIP:     srcIP,
		DstIP:     dstIP,
		SrcPort:   srcPort,
		DstPort:   dstPort,
		BytesSent: bytesSent,
		BytesRecv: bytesRecv,
	}, true
}

// BuildTrafficSummary aggregates conntrack entries into a summary.
// hostnames maps IP → hostname (from DHCP leases).
func BuildTrafficSummary(entries []ConnEntry, hostnames map[string]string) TrafficSummary {
	hostMap := make(map[string]*HostTraffic)
	hostDestMap := make(map[string]map[string]*EndpointStat)
	globalDestMap := make(map[string]*EndpointStat)

	var totalSent, totalRecv int64

	for _, e := range entries {
		totalSent += e.BytesSent
		totalRecv += e.BytesRecv

		ht, ok := hostMap[e.SrcIP]
		if !ok {
			ht = &HostTraffic{IP: e.SrcIP, Hostname: hostnames[e.SrcIP]}
			hostMap[e.SrcIP] = ht
			hostDestMap[e.SrcIP] = make(map[string]*EndpointStat)
		}
		ht.BytesSent += e.BytesSent
		ht.BytesRecv += e.BytesRecv
		ht.Connections++

		destKey := fmt.Sprintf("%s:%d:%s", e.DstIP, e.DstPort, e.Protocol)

		ds, ok := hostDestMap[e.SrcIP][destKey]
		if !ok {
			ds = &EndpointStat{IP: e.DstIP, Port: e.DstPort, Protocol: e.Protocol}
			hostDestMap[e.SrcIP][destKey] = ds
		}
		ds.BytesSent += e.BytesSent
		ds.BytesRecv += e.BytesRecv
		ds.Connections++

		gds, ok := globalDestMap[destKey]
		if !ok {
			gds = &EndpointStat{IP: e.DstIP, Port: e.DstPort, Protocol: e.Protocol}
			globalDestMap[destKey] = gds
		}
		gds.BytesSent += e.BytesSent
		gds.BytesRecv += e.BytesRecv
		gds.Connections++
	}

	hosts := make([]HostTraffic, 0, len(hostMap))
	for ip, ht := range hostMap {
		ht.TopDests = sortEndpoints(hostDestMap[ip], 10)
		hosts = append(hosts, *ht)
	}
	sort.Slice(hosts, func(i, j int) bool {
		return (hosts[i].BytesSent + hosts[i].BytesRecv) > (hosts[j].BytesSent + hosts[j].BytesRecv)
	})

	topDests := sortEndpoints(globalDestMap, 20)

	return TrafficSummary{
		TotalSent:       totalSent,
		TotalRecv:       totalRecv,
		TotalConns:      len(entries),
		Hosts:           hosts,
		TopDestinations: topDests,
	}
}

func sortEndpoints(m map[string]*EndpointStat, n int) []EndpointStat {
	result := make([]EndpointStat, 0, len(m))
	for _, ds := range m {
		result = append(result, *ds)
	}
	sort.Slice(result, func(i, j int) bool {
		return (result[i].BytesSent + result[i].BytesRecv) > (result[j].BytesSent + result[j].BytesRecv)
	})
	if len(result) > n {
		result = result[:n]
	}
	return result
}
