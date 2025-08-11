package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"time"
)

type Payload struct {
	Timestamp string `json:"timestamp"`
	Hostname  string `json:"hostname"`
	Source    string `json:"event.source.type"`
	Category  string `json:"event.category"`
	Message   string `json:"message"`
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := getEnv("COLLECTOR_ADDR", "log-collector:9000")
	hostname := getEnv("HOSTNAME", "aiops9242")
	d := net.Dialer{Timeout: 5 * time.Second}
	for {
		conn, err := d.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("connected to %s", addr)
		writer := bufio.NewWriter(conn)
		enc := json.NewEncoder(writer)
		for {
			username := []string{"root", "motadata", "alice", "bob"}[rand.IntN(4)]
			msg := "<86> " + hostname + " systemd: session closed for user " + username
			p := Payload{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Hostname:  hostname,
				Source:    "linux",
				Category:  "logout.audit",
				Message:   msg,
			}
			if err := enc.Encode(p); err != nil {
				log.Printf("encode error: %v", err)
				break
			}
			if err := writer.Flush(); err != nil {
				log.Printf("flush error: %v", err)
				break
			}
			time.Sleep(time.Duration(1000+rand.IntN(1000)) * time.Millisecond)
		}
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}
