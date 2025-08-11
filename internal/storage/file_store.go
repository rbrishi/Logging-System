package storage

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"motadata/internal/model"
)

// FileBackedStore persists logs to a JSONL file while keeping an in-memory index for queries.
type FileBackedStore struct {
	mem  *InMemoryStore
	file *os.File
	enc  *json.Encoder
	fMu  sync.Mutex
}

func NewFileBackedStore(filePath string) (*FileBackedStore, error) {
	if err := os.MkdirAll(dirOf(filePath), 0o755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	// Open the append handle for durability on each ingest
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	s := &FileBackedStore{
		mem:  NewInMemoryStore(),
		file: f,
		enc:  json.NewEncoder(f),
	}
	// Preload existing entries into memory index for queries/metrics
	if err := s.loadExisting(filePath); err != nil {
		// If loading fails for a non-existent file, ignore; otherwise propagate
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return s, nil
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

// loadExisting reads existing JSONL file and populates the in-memory index.
func (s *FileBackedStore) loadExisting(path string) error {
	rf, err := os.Open(path)
	if err != nil {
		return err
	}
	defer rf.Close()
	dec := json.NewDecoder(rf)
	for {
		var e model.LogEntry
		if err := dec.Decode(&e); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		_ = s.mem.Ingest(e)
	}
	return nil
}
