package monitor

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"

	nflog "github.com/florianl/go-nflog/v2"
)

// nflogGroup is the netfilter log group used for deep inspection.
const nflogGroup = 100

// CaptureTarget identifies one port-forwarded internal service to inspect.
type CaptureTarget struct {
	InternalIP   string
	InternalPort int
	Protocol     string // "tcp" (only tcp is inspected)
}

// Capture mirrors the first packets of inbound connections to port-forwarded
// services into userspace via NFLOG, extracts the application-layer host/URL
// (TLS SNI or HTTP Host), and records them in the access log.
type Capture struct {
	wanIface  string
	targets   []CaptureTarget
	accessLog *AccessLog
	geoCache  *GeoIPCache

	flows  map[string]time.Time // dedupe key -> last seen; one entry per inbound flow
	flowMu sync.Mutex
}

// NewCapture installs NFLOG rules for the given targets and starts reading.
func NewCapture(ctx context.Context, wanIface string, forwards []CaptureTarget, accessLog *AccessLog, geoCache *GeoIPCache) *Capture {
	c := &Capture{
		wanIface:  wanIface,
		accessLog: accessLog,
		geoCache:  geoCache,
		flows:     make(map[string]time.Time),
	}
	for _, f := range forwards {
		if f.Protocol == "tcp" {
			c.targets = append(c.targets, f)
		}
	}

	if len(c.targets) == 0 {
		log.Println("[Capture] 검사할 TCP 포트포워딩이 없어 비활성화")
		return c
	}

	c.addRules()
	go c.run(ctx)
	return c
}

// ruleSpec builds the iptables argument list for one target (without the -A/-D verb).
// We hook the mangle FORWARD chain: it runs post-DNAT (so the destination is the
// internal host) and before the filter FORWARD ACCEPT, and NFLOG is non-terminating
// so the packet still proceeds normally. connbytes limits us to the first packets
// of each connection (handshake + first data segment) to keep overhead negligible.
func (c *Capture) ruleSpec(t CaptureTarget) []string {
	return []string{
		"-t", "mangle", "FORWARD",
		"-i", c.wanIface,
		"-p", "tcp",
		"-d", t.InternalIP,
		"--dport", strconv.Itoa(t.InternalPort),
		"-m", "conntrack", "--ctdir", "original",
		"-m", "connbytes", "--connbytes", "0:8",
		"--connbytes-dir", "original", "--connbytes-mode", "packets",
		"-j", "NFLOG",
		"--nflog-group", strconv.Itoa(nflogGroup),
		"--nflog-range", "2048",
	}
}

func (c *Capture) addRules() {
	for _, t := range c.targets {
		spec := c.ruleSpec(t)
		// Remove first (idempotent), then insert at the top of mangle FORWARD.
		exec.Command("iptables", rebuild("-D", spec)...).Run()
		if err := exec.Command("iptables", rebuild("-I", spec)...).Run(); err != nil {
			log.Printf("[Capture] NFLOG 룰 추가 실패 (%s:%d): %v", t.InternalIP, t.InternalPort, err)
		}
	}
}

func (c *Capture) removeRules() {
	for _, t := range c.targets {
		exec.Command("iptables", rebuild("-D", c.ruleSpec(t))...).Run()
	}
}

// rebuild inserts the chain verb after the "-t <table>" prefix.
// spec is ["-t","mangle","FORWARD", ...]; result is ["-t","mangle",verb,"FORWARD",...].
func rebuild(verb string, spec []string) []string {
	out := make([]string, 0, len(spec)+1)
	out = append(out, spec[0], spec[1], verb) // -t mangle <verb>
	out = append(out, spec[2:]...)            // FORWARD ...
	return out
}

func (c *Capture) run(ctx context.Context) {
	defer c.removeRules()

	for {
		if err := c.readOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Capture] NFLOG 종료, 재시작 대기: %v", err)
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return
			}
		} else {
			return
		}
	}
}

func (c *Capture) readOnce(ctx context.Context) error {
	cfg := nflog.Config{
		Group:    nflogGroup,
		Copymode: nflog.CopyPacket,
		Bufsize:  2048,
	}
	nf, err := nflog.Open(&cfg)
	if err != nil {
		return fmt.Errorf("nflog open: %w", err)
	}
	defer nf.Close()

	hook := func(a nflog.Attribute) int {
		if a.Payload == nil {
			return 0
		}
		c.handlePayload(*a.Payload)
		return 0
	}
	errFn := func(e error) int {
		log.Printf("[Capture] NFLOG 읽기 오류: %v", e)
		return 0
	}

	if err := nf.RegisterWithErrorFunc(ctx, hook, errFn); err != nil {
		return fmt.Errorf("nflog register: %w", err)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *Capture) handlePayload(data []byte) {
	pkt, ok := parsePacket(data)
	if !ok || pkt.Proto != "tcp" || len(pkt.Payload) == 0 {
		return
	}

	var host, url, clientIP string
	if info, isHTTP := parseHTTPRequest(pkt.Payload); isHTTP {
		host, url, clientIP = info.Host, info.URL, info.ClientIP
	} else if sni, ok := parseTLSServerName(pkt.Payload); ok {
		host = sni
	} else {
		return // no application-layer info in this packet
	}

	// Dedupe: one access entry per inbound connection.
	flowKey := fmt.Sprintf("%s:%d->%s:%d", pkt.SrcIP, pkt.SrcPort, pkt.DstIP, pkt.DstPort)
	if !c.markFlow(flowKey) {
		return
	}

	// If a proxy (e.g. Cloudflare) forwarded the real client IP in a header,
	// use it as the source and keep the packet's source as the edge/proxy IP.
	sourceIP := pkt.SrcIP
	viaProxy := ""
	reason := "inbound request"
	if clientIP != "" && clientIP != pkt.SrcIP {
		sourceIP = clientIP
		viaProxy = pkt.SrcIP
		reason = "inbound via proxy"
	}

	entry := AccessEntry{
		Timestamp: time.Now(),
		SourceIP:  sourceIP,
		DestIP:    pkt.DstIP,
		DestPort:  pkt.DstPort,
		Protocol:  "tcp",
		Host:      host,
		URL:       url,
		ViaProxy:  viaProxy,
		Action:    "FORWARD",
		Reason:    reason,
	}
	if c.geoCache != nil {
		geo := c.geoCache.Lookup(sourceIP)
		entry.Country = geo.Country
		entry.CountryCode = geo.CountryCode
		entry.Org = geo.Org
	}
	entry.PortName = wellKnownPort(pkt.DstPort, "tcp")

	c.accessLog.Add(entry)
}

// markFlow returns true the first time a flow key is seen within the dedupe
// window, false otherwise. It opportunistically evicts stale entries.
func (c *Capture) markFlow(key string) bool {
	now := time.Now()
	c.flowMu.Lock()
	defer c.flowMu.Unlock()

	if last, ok := c.flows[key]; ok && now.Sub(last) < 60*time.Second {
		c.flows[key] = now
		return false
	}
	c.flows[key] = now

	if len(c.flows) > 4096 {
		for k, t := range c.flows {
			if now.Sub(t) > 60*time.Second {
				delete(c.flows, k)
			}
		}
	}
	return true
}
