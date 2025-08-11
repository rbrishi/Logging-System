package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"motadata/internal/model"
	"motadata/internal/storage"
)

type Server struct {
	store storage.LogStore
}

func NewServer(store storage.LogStore) *Server {
	return &Server{store: store}
}

func (s *Server) ingestHandler(w http.ResponseWriter, r *http.Request) {
	var entry model.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if err := s.store.Ingest(entry); err != nil {
		http.Error(w, "failed to ingest", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) logsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := storage.QueryFilter{}
	filter.Service = q.Get("service")
	filter.Level = q.Get("level")
	filter.Username = q.Get("username")
	if v := q.Get("is.blacklisted"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		filter.IsBlacklisted = &b
	}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			filter.Limit = n
		}
	}
	filter.SortBy = q.Get("sort")

	res, err := s.store.Query(filter)
	if err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	m := s.store.Metrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func main() {
	// Select store backend
	storeType := getEnv("STORE", "memory") // memory | file
	var store storage.LogStore
	if storeType == "file" {
		path := getEnv("STORE_PATH", "/data/logs.jsonl")
		fs, err := storage.NewFileBackedStore(path)
		if err != nil {
			log.Fatalf("failed to init file store: %v", err)
		}
		store = fs
	} else {
		store = storage.NewInMemoryStore()
	}
	srv := NewServer(store)

	r := mux.NewRouter()
	r.HandleFunc("/ingest", srv.ingestHandler).Methods(http.MethodPost)
	r.HandleFunc("/logs", srv.logsHandler).Methods(http.MethodGet)
	r.HandleFunc("/metrics", srv.metricsHandler).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }).Methods(http.MethodGet)

	addr := getEnv("LISTEN_ADDR", ":8000")
	log.Printf("log-server listening on %s", addr)
	srvHTTP := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := srvHTTP.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
