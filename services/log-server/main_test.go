package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gorilla/mux"
    "motadata/internal/model"
    "motadata/internal/storage"
)

func setupTestServer() (*Server, *mux.Router) {
    s := NewServer(storage.NewInMemoryStore())
    r := mux.NewRouter()
    r.HandleFunc("/ingest", s.ingestHandler).Methods(http.MethodPost)
    r.HandleFunc("/logs", s.logsHandler).Methods(http.MethodGet)
    r.HandleFunc("/metrics", s.metricsHandler).Methods(http.MethodGet)
    return s, r
}

func TestIngestAndQueryRoutes(t *testing.T) {
    _, r := setupTestServer()
    entry := model.LogEntry{
        Timestamp:       time.Now().UTC(),
        EventCategory:   "login.audit",
        EventSourceType: "linux",
        Username:        "alice",
        Hostname:        "h1",
        Severity:        "INFO",
        Service:         "linux_login",
        RawMessage:      "<86> h1 sudo: session opened for user alice",
        IsBlacklisted:   false,
    }
    body, _ := json.Marshal(entry)
    req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(body))
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusAccepted {
        t.Fatalf("expected 202, got %d", w.Code)
    }

    // Query it back
    req2 := httptest.NewRequest(http.MethodGet, "/logs?service=linux_login", nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w2.Code)
    }
}


