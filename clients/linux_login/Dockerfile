FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/linux_login .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /out/linux_login /linux_login
ENV COLLECTOR_ADDR=log-collector:9000
ENTRYPOINT ["/linux_login"]


