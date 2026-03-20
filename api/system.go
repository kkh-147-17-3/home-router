package api

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (s *Server) handleSystemUptime(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)
	writeJSON(w, map[string]interface{}{
		"uptime_seconds": int(uptime.Seconds()),
		"start_time":     s.startTime.Format(time.RFC3339),
		"uptime":         uptime.String(),
	})
}

func (s *Server) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	// 민감 정보 제거 후 YAML로 반환
	cfgCopy := *s.cfg
	cfgCopy.Web.PasswordHash = "***"

	data, err := yaml.Marshal(cfgCopy)
	if err != nil {
		http.Error(w, `{"error":"failed to marshal config"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"config": string(data)})
}

var allowedUnits = map[string]bool{
	"home-router":       true,
	"systemd-networkd":  true,
	"systemd-resolved":  true,
	"systemd-timesyncd": true,
}

type logEntry struct {
	Timestamp string `json:"timestamp"`
	Priority  string `json:"priority"`
	Unit      string `json:"unit"`
	Message   string `json:"message"`
	PID       int    `json:"pid,omitempty"`
}

func (s *Server) handleSystemLogs(w http.ResponseWriter, r *http.Request) {
	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = "home-router"
	}
	if !allowedUnits[unit] {
		http.Error(w, `{"error":"unit not allowed"}`, http.StatusBadRequest)
		return
	}

	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "100"
	}
	if n, err := strconv.Atoi(lines); err != nil || n < 1 || n > 1000 {
		lines = "100"
	}

	args := []string{"-u", unit, "-n", lines, "-o", "json", "--no-pager"}
	if since := r.URL.Query().Get("since"); since != "" {
		args = append(args, "--since", since)
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		args = append(args, "-p", priority)
	}
	if grep := r.URL.Query().Get("grep"); grep != "" {
		args = append(args, "--grep", grep)
	}

	cmd := exec.CommandContext(r.Context(), "journalctl", args...)
	output, err := cmd.Output()
	if err != nil {
		// journalctl은 결과가 없어도 exit code 1을 반환할 수 있음
		if len(output) == 0 {
			writeJSON(w, []logEntry{})
			return
		}
	}

	entries := parseJournalOutput(string(output), unit)
	writeJSON(w, entries)
}

func parseJournalOutput(output, defaultUnit string) []logEntry {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	entries := make([]logEntry, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		entry := logEntry{
			Unit: defaultUnit,
		}

		if ts, ok := raw["__REALTIME_TIMESTAMP"].(string); ok {
			if usec, err := strconv.ParseInt(ts, 10, 64); err == nil {
				entry.Timestamp = time.UnixMicro(usec).Format(time.RFC3339Nano)
			}
		}

		if msg, ok := raw["MESSAGE"].(string); ok {
			entry.Message = msg
		}

		if p, ok := raw["PRIORITY"].(string); ok {
			entry.Priority = priorityName(p)
		}

		if pid, ok := raw["_PID"].(string); ok {
			entry.PID, _ = strconv.Atoi(pid)
		}

		entries = append(entries, entry)
	}

	return entries
}

func priorityName(p string) string {
	switch p {
	case "0":
		return "emerg"
	case "1":
		return "alert"
	case "2":
		return "crit"
	case "3":
		return "err"
	case "4":
		return "warning"
	case "5":
		return "notice"
	case "6":
		return "info"
	case "7":
		return "debug"
	default:
		return p
	}
}
