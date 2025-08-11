package storage

import (
    "encoding/json"
    "os"
    "sync"

    "motadata/internal/model"
)

// FileBackedStore persists logs to a JSONL file while keeping an in-memory index for queries.
type FileBackedStore struct {
    mem   *InMemoryStore
    file  *os.File
    enc   *json.Encoder
    fMu   sync.Mutex
}

func NewFileBackedStore(filePath string) (*FileBackedStore, error) {
    if err := os.MkdirAll(dirOf(filePath), 0o755); err != nil && !os.IsExist(err) {
        return nil, err
    }
    f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil {
        return nil, err
    }
    return &FileBackedStore{
        mem:  NewInMemoryStore(),
        file: f,
        enc:  json.NewEncoder(f),
    }, nil
}

func (s *FileBackedStore) Ingest(entry model.LogEntry) error {
    // Write to file first to ensure durability
    s.fMu.Lock()
    if err := s.enc.Encode(entry); err != nil {
        s.fMu.Unlock()
        return err
    }
    if err := s.file.Sync(); err != nil {
        s.fMu.Unlock()
        return err
    }
    s.fMu.Unlock()
    // Update in-memory metrics and index
    return s.mem.Ingest(entry)
}

func (s *FileBackedStore) Query(filter QueryFilter) ([]model.LogEntry, error) {
    return s.mem.Query(filter)
}

func (s *FileBackedStore) Metrics() Metrics {
    return s.mem.Metrics()
}

func dirOf(path string) string {
    for i := len(path) - 1; i >= 0; i-- {
        if path[i] == '/' {
            return path[:i]
        }
    }
    return "."
}


