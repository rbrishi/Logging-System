FROM golang:1.22
WORKDIR /app
COPY . .
WORKDIR /app/services/log-server
RUN go build -o log-server
EXPOSE 8000
ENV LISTEN_ADDR=:8000
ENV STORE=memory
CMD ["./log-server"]


