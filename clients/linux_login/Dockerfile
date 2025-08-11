FROM golang:1.22
WORKDIR /app
COPY . .
WORKDIR /app/clients/linux_login
RUN go build -o linux_login
CMD ["./linux_login"]


