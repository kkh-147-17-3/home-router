package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

func (s *Server) handleSSEDNSQueryLog(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"SSE not supported"}`, http.StatusInternalServerError)
		return
	}

	if s.queryLog == nil {
		http.Error(w, `{"error":"dns not enabled"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := s.queryLog.Subscribe()
	defer s.queryLog.Unsubscribe(ch)

	for {
		select {
		case entry := <-ch:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleSSESystemLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"SSE not supported"}`, http.StatusInternalServerError)
		return
	}

	unit := r.URL.Query().Get("unit")
	if unit == "" {
		unit = "home-router"
	}
	if !allowedUnits[unit] {
		http.Error(w, `{"error":"unit not allowed"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cmd := exec.CommandContext(r.Context(), "journalctl",
		"-u", unit, "-f", "-o", "json", "--no-pager")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, `{"error":"failed to start journalctl"}`, http.StatusInternalServerError)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, `{"error":"failed to start journalctl"}`, http.StatusInternalServerError)
		return
	}

	go func() {
		<-r.Context().Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		entries := parseJournalOutput(line, unit)
		if len(entries) > 0 {
			data, _ := json.Marshal(entries[0])
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}

	cmd.Wait()
}
