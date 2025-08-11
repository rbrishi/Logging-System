package main

import (
	"testing"
	"time"
)

func TestToLogEntryExtraction(t *testing.T) {
    cl := ClientLog{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Hostname:  "aiops9242",
        Source:    "linux",
        Category:  "login.audit",
        Message:   "<86> aiops9242 sudo: pam_unix(sudo:session): session opened for user root(uid=0)",
    }
    le := parseLog(cl)
    if le.Username == "" || le.Severity == "" || le.Hostname == "" {
        t.Fatalf("expected fields to be extracted, got: %+v", le)
    }
}

func TestEnrichLogBlacklistedUser(t *testing.T) {
    cl := ClientLog{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Hostname:  "aiops9242",
        Source:    "linux",
        Category:  "login.audit",
        Message:   "<86> aiops9242 sudo: pam_unix(sudo:session): session opened for user root(uid=0)",
    }
    le := parseLog(cl)
    if !le.IsBlacklisted {
        t.Fatalf("expected root to be blacklisted, got: %+v", le)
    }
}


