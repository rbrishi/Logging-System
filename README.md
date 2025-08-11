## Centralized Logging System (Go + Docker)

### Architecture

- `clients/linux_login`, `clients/linux_logout`: Emit Linux login/logout logs every 1–2s over TCP to `log-collector:9000`.
- `services/log-collector`: TCP listener on `:9000`, parses/enriches logs, forwards to `log-server` `POST /ingest`, exposes `/metrics` on `:8080`.
- `services/log-server`: Receives logs on `:8000` (`/ingest`), serves `GET /logs`, `GET /metrics`, `GET /healthz`.

### Run with Docker

1. Build and start all services:

```
docker compose up --build
```

2. Endpoints and Ports:

- Collector: `http://localhost:8080/metrics` (metrics), TCP `localhost:9000`
- Server: `http://localhost:8000`
  - `POST /ingest`
  - `GET /logs` query params: `service`, `level`, `username`, `is.blacklisted`, `limit`, `sort`
  - `GET /metrics`
  - `GET /healthz`

3. Example queries:

- `GET /logs?service=linux_login_audit`
- `GET /logs?level=error`
- `GET /logs?service=linux_login_audit&level=warn`
- `GET /logs?username=root&is.blacklisted=true`
- `GET /logs?limit=10&sort=timestamp`

Send a sample log to collector via TCP:

```
printf '{"timestamp":"%s","hostname":"aiops9242","event.source.type":"linux","event.category":"login.audit","message":"<86> aiops9242 sudo: pam_unix(sudo:session): session opened for user root(uid=0) by motadata(uid=1000)"}\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" | nc localhost 9000
```

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
