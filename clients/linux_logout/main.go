package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"time"
)

type Payload struct {
    Timestamp string `json:"timestamp"`
    Hostname  string `json:"hostname"`
    Source    string `json:"event.source.type"`
    Category  string `json:"event.category"`
    Message   string `json:"message"`
}

func randomHostname() string {
    hosts := []string{"aiops9242", "node-01", "db-02", "api-03", "worker-07"}
    return hosts[rand.IntN(len(hosts))]
}

func main() {
    addr := "log-collector:9000" // hardcoded per assignment
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
            hostname := randomHostname()
            username := []string{"root", "motadata", "alice", "bob"}[rand.IntN(4)]
            msg := fmt.Sprintf("<86> %s systemd: session closed for user %s", hostname, username)
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


