FROM golang:1.22
WORKDIR /app
COPY . .
WORKDIR /app/services/log-collector
RUN go build -o log-collector
EXPOSE 9000 8080
ENV LISTEN_ADDR=:9000
ENV SERVER_INGEST=http://log-server:8000/ingest
CMD ["./log-collector"]


