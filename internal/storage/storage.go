package storage

import (
    "errors"
    "sort"
    "strings"
    "sync"
    "time"

    "motadata/internal/model"
)

// LogStore defines abstract storage for logs so we can plug different backends.
type LogStore interface {
    Ingest(entry model.LogEntry) error
    Query(filter QueryFilter) ([]model.LogEntry, error)
    Metrics() Metrics
}

// QueryFilter supports basic filtering and sorting.
type QueryFilter struct {
    Service      string
    Level        string
    Username     string
    IsBlacklisted *bool
    Limit        int
    SortBy       string // e.g. "timestamp"
}

type Metrics struct {
    Total int
    ByCategory map[string]int
    BySeverity map[string]int
}

// InMemoryStore provides a threadsafe in-memory log store.
type InMemoryStore struct {
    mu       sync.RWMutex
    logs     []model.LogEntry
    total    int
    byCat    map[string]int
    bySev    map[string]int
}

func NewInMemoryStore() *InMemoryStore {
    return &InMemoryStore{
        logs:  make([]model.LogEntry, 0, 1024),
        byCat: make(map[string]int),
        bySev: make(map[string]int),
    }
}

func (s *InMemoryStore) Ingest(entry model.LogEntry) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.logs = append(s.logs, entry)
    s.total++
    if entry.EventCategory != "" {
        s.byCat[strings.ToLower(entry.EventCategory)]++
    }
    if entry.Severity != "" {
        s.bySev[strings.ToUpper(entry.Severity)]++
    }
    return nil
}

func (s *InMemoryStore) Query(filter QueryFilter) ([]model.LogEntry, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    results := make([]model.LogEntry, 0)
    for _, e := range s.logs {
        if filter.Service != "" && !strings.EqualFold(e.Service, filter.Service) {
            continue
        }
        if filter.Level != "" && !strings.EqualFold(e.Severity, filter.Level) {
            continue
        }
        if filter.Username != "" && !strings.EqualFold(e.Username, filter.Username) {
            continue
        }
        if filter.IsBlacklisted != nil && e.IsBlacklisted != *filter.IsBlacklisted {
            continue
        }
        results = append(results, e)
    }

    if filter.SortBy != "" {
        switch strings.ToLower(filter.SortBy) {
        case "timestamp":
            sort.Slice(results, func(i, j int) bool {
                return results[i].Timestamp.Before(results[j].Timestamp)
            })
        default:
            // ignore unknown sort
        }
    }
    if filter.Limit > 0 && len(results) > filter.Limit {
        results = results[:filter.Limit]
    }
    return results, nil
}

func (s *InMemoryStore) Metrics() Metrics {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // Copy maps to avoid external mutation
    mCat := make(map[string]int, len(s.byCat))
    mSev := make(map[string]int, len(s.bySev))
    for k, v := range s.byCat {
        mCat[k] = v
    }
    for k, v := range s.bySev {
        mSev[k] = v
    }
    return Metrics{Total: s.total, ByCategory: mCat, BySeverity: mSev}
}

// Utility for parsing severity code like "<86>" into a symbolic level.
func SeverityFromCode(code int) string {
    // Map basic ranges to levels; simple heuristic
    switch {
    case code <= 3:
        return "ERROR"
    case code <= 5:
        return "WARN"
    default:
        return "INFO"
    }
}

// ParseTimestamp gracefully parses a RFC3339 time or returns now.
func ParseTimestamp(ts string) time.Time {
    if ts == "" {
        return time.Now().UTC()
    }
    t, err := time.Parse(time.RFC3339, ts)
    if err != nil {
        return time.Now().UTC()
    }
    return t
}

var ErrNotImplemented = errors.New("not implemented")


