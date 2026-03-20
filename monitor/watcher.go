package monitor

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const logPrefix = "HOME-ROUTER-WAN: "

var (
	srcRe   = regexp.MustCompile(`SRC=(\S+)`)
	dstRe   = regexp.MustCompile(`DST=(\S+)`)
	dptRe   = regexp.MustCompile(`DPT=(\d+)`)
	protoRe = regexp.MustCompile(`PROTO=(\S+)`)
)

// Watcher adds iptables LOG rules and parses kernel log for WAN access attempts.
type Watcher struct {
	wanIface  string
	accessLog *AccessLog
	geoCache  *GeoIPCache
	cancel    context.CancelFunc
}

// NewWatcher creates a Watcher and starts monitoring.
func NewWatcher(ctx context.Context, wanIface string, accessLog *AccessLog) *Watcher {
	ctx, cancel := context.WithCancel(ctx)
	w := &Watcher{
		wanIface:  wanIface,
		accessLog: accessLog,
		geoCache:  NewGeoIPCache(),
		cancel:    cancel,
	}

	w.addLogRules()
	go w.watchLogs(ctx)

	return w
}

func (w *Watcher) addLogRules() {
	// Remove existing rules first (idempotent)
	exec.Command("iptables", "-D", "INPUT", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run()
	exec.Command("iptables", "-D", "FORWARD", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run()

	// Add LOG rules
	if err := exec.Command("iptables", "-I", "INPUT", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run(); err != nil {
		log.Printf("[Monitor] INPUT LOG 룰 추가 실패: %v", err)
	}
	if err := exec.Command("iptables", "-I", "FORWARD", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run(); err != nil {
		log.Printf("[Monitor] FORWARD LOG 룰 추가 실패: %v", err)
	}
}

func (w *Watcher) removeLogRules() {
	exec.Command("iptables", "-D", "INPUT", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run()
	exec.Command("iptables", "-D", "FORWARD", "-i", w.wanIface,
		"-m", "state", "--state", "NEW",
		"-j", "LOG", "--log-prefix", logPrefix).Run()
}

func (w *Watcher) watchLogs(ctx context.Context) {
	defer w.removeLogRules()

	for {
		if err := w.runJournalctl(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Monitor] journalctl 종료, 재시작 대기: %v", err)
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (w *Watcher) runJournalctl(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "journalctl", "-k", "-f",
		"--grep", logPrefix, "--no-pager", "-o", "short-unix")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("journalctl pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("journalctl start: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if entry, ok := parseLine(line); ok {
			geo := w.geoCache.Lookup(entry.SourceIP)
			entry.Country = geo.Country
			entry.CountryCode = geo.CountryCode
			entry.PortName = wellKnownPort(entry.DestPort, entry.Protocol)
			w.accessLog.Add(entry)
		}
	}

	return cmd.Wait()
}

func parseLine(line string) (AccessEntry, bool) {
	if !strings.Contains(line, logPrefix) {
		return AccessEntry{}, false
	}

	srcMatch := srcRe.FindStringSubmatch(line)
	dstMatch := dstRe.FindStringSubmatch(line)
	dptMatch := dptRe.FindStringSubmatch(line)
	protoMatch := protoRe.FindStringSubmatch(line)

	if len(srcMatch) < 2 || len(protoMatch) < 2 {
		return AccessEntry{}, false
	}

	var destIP string
	if len(dstMatch) >= 2 {
		destIP = dstMatch[1]
	}

	var destPort int
	if len(dptMatch) >= 2 {
		destPort, _ = strconv.Atoi(dptMatch[1])
	}

	action := "DROP"
	reason := "WAN inbound"

	return AccessEntry{
		Timestamp: time.Now(),
		SourceIP:  srcMatch[1],
		DestIP:    destIP,
		DestPort:  destPort,
		Protocol:  strings.ToLower(protoMatch[1]),
		Action:    action,
		Reason:    reason,
	}, true
}

// Stop halts the watcher.
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}
