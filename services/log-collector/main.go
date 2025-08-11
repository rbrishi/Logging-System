package main

import (
	"bufio"
	"bytes"
	"encoding/json"
    "fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"motadata/internal/model"
)

// Incoming payload from clients
type ClientLog struct {
	Timestamp string `json:"timestamp,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Source    string `json:"event.source.type,omitempty"`
	Category  string `json:"event.category,omitempty"`
	Message   string `json:"message"`
}

var (
	blacklistUsers = map[string]struct{}{"root": {}, "admin": {}}
	blacklistIPs   = map[string]struct{}{"10.0.0.13": {}, "192.168.1.66": {}}
)

// Regexes for simple syslog-like message parsing
var (
	// e.g. <86> hostname sudo: pam_unix(sudo:session): session opened for user root(uid=0)
	reSyslog = regexp.MustCompile(`^<(\d+)>\s+(\S+)\s+([^:]+):\s+(.*)$`)
	reUser   = regexp.MustCompile(`user\s+([A-Za-z0-9_-]+)`)
)

func parseSeverity(codeStr string) string {
	code := 6 // default info-ish
	if codeStr != "" {
		// ignore error; already defaulted
		if n, err := strconv.Atoi(codeStr); err == nil {
			code = n
		}
	}
	switch {
	case code <= 3:
		return "ERROR"
	case code <= 5:
		return "WARN"
	default:
		return "INFO"
	}
}

func enrichLog(entry *model.LogEntry) {
	if _, ok := blacklistUsers[strings.ToLower(entry.Username)]; ok {
		entry.IsBlacklisted = true
		return
	}
	// naive IP search in raw message
	for ip := range blacklistIPs {
		if strings.Contains(entry.RawMessage, ip) {
			entry.IsBlacklisted = true
			return
		}
	}
}

func parseLog(cl ClientLog) model.LogEntry {
	ts := time.Now().UTC()
	if cl.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, cl.Timestamp); err == nil {
			ts = t
		}
	}
	entry := model.LogEntry{
		Timestamp:       ts,
		EventCategory:   cl.Category,
		EventSourceType: cl.Source,
		Hostname:        cl.Hostname,
		RawMessage:      cl.Message,
		Service:         strings.ToLower(cl.Source) + "_" + strings.ReplaceAll(strings.ToLower(cl.Category), ".", "_"),
	}
	// Parse severity, hostname, and username from message where possible
	if m := reSyslog.FindStringSubmatch(cl.Message); len(m) == 5 {
		entry.Severity = parseSeverity(m[1])
		if entry.Hostname == "" {
			entry.Hostname = m[2]
		}
		// Try to get username
		if u := reUser.FindStringSubmatch(cl.Message); len(u) == 2 {
			entry.Username = u[1]
		}
	}
	enrichLog(&entry)
	return entry
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func forwardLog(entry model.LogEntry, endpoint string) error {
	b, _ := json.Marshal(entry)
	req, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ingest returned status %d", resp.StatusCode)
	}
	return nil
}

// Metrics for collector
type collectorMetrics struct {
	mu    sync.RWMutex
	total int
	byCat map[string]int
	bySev map[string]int
}

func newCollectorMetrics() *collectorMetrics {
	return &collectorMetrics{byCat: make(map[string]int), bySev: make(map[string]int)}
}

func (m *collectorMetrics) inc(category, severity string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total++
	if category != "" {
		m.byCat[strings.ToLower(category)]++
	}
	if severity != "" {
		m.bySev[strings.ToUpper(severity)]++
	}
}

func (m *collectorMetrics) snapshot() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	byCat := make(map[string]int, len(m.byCat))
	bySev := make(map[string]int, len(m.bySev))
	for k, v := range m.byCat {
		byCat[k] = v
	}
	for k, v := range m.bySev {
		bySev[k] = v
	}
	return map[string]any{"total": m.total, "byCategory": byCat, "bySeverity": bySev}
}

func startMetricsServer(addr string, m *collectorMetrics) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.snapshot())
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("metrics server error: %v", err)
		}
	}()
}

// listenTCP accepts connections and emits ClientLog messages to the channel
func listenTCP(addr string, out chan<- ClientLog) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("log-collector listening on %s", addr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("accept error: %v", err)
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				for {
					line, err := reader.ReadBytes('\n')
					if len(line) > 0 {
						var cl ClientLog
						if err := json.Unmarshal(bytes.TrimSpace(line), &cl); err == nil {
							out <- cl
						} else {
							log.Printf("invalid client payload: %v", err)
						}
					}
					if err != nil {
						if err != io.EOF {
							log.Printf("conn read error: %v", err)
						}
						return
					}
				}
			}(conn)
		}
	}()
	return nil
}

func main() {
	// Config via env
	listenAddr := getEnv("LISTEN_ADDR", ":9000")
	serverIngest := getEnv("SERVER_INGEST", "http://log-server:8000/ingest")
	// Start metrics server
	m := newCollectorMetrics()
	startMetricsServer(":8080", m)

	// Channel and worker pool
	ch := make(chan ClientLog, 1024)
	if err := listenTCP(listenAddr, ch); err != nil {
		log.Fatalf("listen error: %v", err)
	}
	workers := 4
	if wStr := os.Getenv("WORKERS"); wStr != "" {
		if n, err := strconv.Atoi(wStr); err == nil && n > 0 {
			workers = n
		}
	}
	log.Printf("forwarding to %s with %d workers", serverIngest, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for cl := range ch {
				entry := parseLog(cl)
				// update metrics before forwarding
				m.inc(entry.EventCategory, entry.Severity)
				if err := forwardLog(entry, serverIngest); err != nil {
					log.Printf("forward error: %v", err)
				}
			}
		}()
	}
	wg.Wait()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
