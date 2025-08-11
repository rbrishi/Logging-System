package model

import "time"


type LogEntry struct {
	Timestamp       time.Time `json:"timestamp"`
	EventCategory   string    `json:"event.category"`
	EventSourceType string    `json:"event.source.type"`
	Username        string    `json:"username,omitempty"`
	Hostname        string    `json:"hostname,omitempty"`
	Severity        string    `json:"severity,omitempty"`
	Service         string    `json:"service,omitempty"`
	RawMessage      string    `json:"raw.message"`
	IsBlacklisted   bool      `json:"is.blacklisted"`
}
