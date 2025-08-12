FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
WORKDIR /app/services/log-collector
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/log-collector

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/log-collector /log-collector
EXPOSE 9000 8080
ENV LISTEN_ADDR=:9000
ENV SERVER_INGEST=http://log-server:8000/ingest
ENTRYPOINT ["/log-collector"]


