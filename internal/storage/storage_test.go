package storage

import (
	"testing"
	"time"

	"motadata/internal/model"
)

func TestIngestAndQuery(t *testing.T) {
    store := NewInMemoryStore()
    now := time.Now().UTC()
    _ = store.Ingest(model.LogEntry{Timestamp: now, EventCategory: "login.audit", Severity: "INFO", Username: "alice", Service: "linux_login", RawMessage: "msg"})
    _ = store.Ingest(model.LogEntry{Timestamp: now.Add(time.Second), EventCategory: "logout.audit", Severity: "ERROR", Username: "bob", Service: "linux_logout", RawMessage: "msg2", IsBlacklisted: true})

    // Query by service
    res, err := store.Query(QueryFilter{Service: "linux_login"})
    if err != nil {
        t.Fatalf("query error: %v", err)
    }
    if len(res) != 1 || res[0].Username != "alice" {
        t.Fatalf("unexpected result: %+v", res)
    }

    // Query by is.blacklisted
    v := true
    res, err = store.Query(QueryFilter{IsBlacklisted: &v})
    if err != nil {
        t.Fatalf("query error: %v", err)
    }
    if len(res) != 1 || res[0].Username != "bob" {
        t.Fatalf("unexpected result: %+v", res)
    }

    m := store.Metrics()
    if m.Total != 2 || m.ByCategory["login.audit"] != 1 || m.BySeverity["ERROR"] != 1 {
        t.Fatalf("unexpected metrics: %+v", m)
    }
}

func TestSortingAndLimit(t *testing.T) {
    store := NewInMemoryStore()
    base := time.Now().UTC()
    for i := 0; i < 5; i++ {
        _ = store.Ingest(model.LogEntry{Timestamp: base.Add(time.Duration(i) * time.Second), EventCategory: "login.audit", Severity: "INFO", Username: "u", Service: "svc", RawMessage: "m"})
    }
    res, err := store.Query(QueryFilter{SortBy: "timestamp", Limit: 3})
    if err != nil { t.Fatalf("query error: %v", err) }
    if len(res) != 3 { t.Fatalf("expected 3 results, got %d", len(res)) }
    if !res[0].Timestamp.Before(res[1].Timestamp) || !res[1].Timestamp.Before(res[2].Timestamp) {
        t.Fatalf("expected ascending timestamps")
    }
}


