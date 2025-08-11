FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
WORKDIR /app/services/client-linux-login
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/client-linux-login

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/client-linux-login /client-linux-login
ENV COLLECTOR_ADDR=log-collector:9000
ENTRYPOINT ["/client-linux-login"]


