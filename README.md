## Centralized Logging System (Go + Docker)

### Architecture

- `clients/linux_login`, `clients/linux_logout`: Emit Linux login/logout logs every 1–2s over TCP to `log-collector:9000`.
- `services/log-collector`: TCP listener on `:9000`, parses/enriches logs, forwards to `log-server` `POST /ingest`, exposes `/metrics` on `:8080`.
- `services/log-server`: Receives logs on `:8000` (`/ingest`), serves `GET /logs`, `GET /metrics`, `GET /healthz`.

### Setup (Docker)

1) Prerequisites
- Docker Desktop 4.x with Compose V2

2) Build and start

```
docker compose up --build -d
```

3) Verify health
- Collector health: `http://localhost:8080/healthz` (HTTP 200)
- Server health: `http://localhost:8000/healthz` (HTTP 200)

4) Endpoints and Ports
- Collector
  - Metrics: `GET http://localhost:8080/metrics`
  - TCP listener: `localhost:9000`
- Server
  - `POST http://localhost:8000/ingest`
  - `GET http://localhost:8000/logs`
  - `GET http://localhost:8000/metrics`
  - `GET http://localhost:8000/healthz`

5) Persistence
- `log-server` runs with `STORE=file` and writes JSONL to `/data/logs.jsonl` (persisted via Docker volume `logdata`).

### API usage (curl)

Ingest directly into server (normally done by collector):

```
curl -s -X POST http://localhost:8000/ingest \
  -H 'Content-Type: application/json' \
  -d '{
    "timestamp": "2025-07-29T12:35:24Z",
    "event.category": "login.audit",
    "event.source.type": "linux",
    "username": "alice",
    "hostname": "aiops9242",
    "severity": "INFO",
    "service": "linux_login_audit",
    "raw.message": "<86> aiops9242 sudo: pam_unix(sudo:session): session opened for user alice(uid=0)",
    "is.blacklisted": false
  }'
```

Query logs with filters and pagination/sorting:

```
curl -s 'http://localhost:8000/logs?service=linux_login_audit'
curl -s 'http://localhost:8000/logs?level=error'
curl -s 'http://localhost:8000/logs?service=linux_login_audit&level=warn'
curl -s 'http://localhost:8000/logs?username=root&is.blacklisted=true'
curl -s 'http://localhost:8000/logs?limit=10&sort=timestamp'
```

Send a sample client log to the collector over TCP (collector parses/enriches and forwards):

```
printf '{"timestamp":"%s","hostname":"aiops9242","event.source.type":"linux","event.category":"login.audit","message":"<86> aiops9242 sudo: pam_unix(sudo:session): session opened for user root(uid=0) by motadata(uid=1000)"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" | nc localhost 9000
```

Metrics:

```
curl -s http://localhost:8080/metrics | jq
curl -s http://localhost:8000/metrics | jq
```

### Postman

- Create a new collection with requests:
  - POST `http://localhost:8000/ingest` (raw JSON body as above)
  - GET `http://localhost:8000/logs?service=linux_login_audit`
  - GET `http://localhost:8080/metrics`
  - GET `http://localhost:8000/metrics`

### Components

- client-linux-login: generates Linux login audit logs every 1–2 seconds.
- client-linux-logout: generates Linux logout audit logs every 1–2 seconds.
- log-collector: TCP on `:9000`, forwards to server `:8000`, exposes `/metrics` on `:8080`.
- log-server: central store with `/ingest`, `/logs`, `/metrics`, `/healthz` on `:8000`.

### Local Dev

```
go test ./...
go run ./services/log-server
go run ./services/log-collector
go run ./clients/linux_login
go run ./clients/linux_logout
```
