FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
WORKDIR /app/services/log-server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/log-server

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/log-server /log-server
EXPOSE 8000
ENV LISTEN_ADDR=:8000
ENV STORE=memory
ENTRYPOINT ["/log-server"]


