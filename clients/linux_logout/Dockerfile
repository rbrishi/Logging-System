FROM golang:1.22
WORKDIR /app
COPY . .
WORKDIR /app/clients/linux_logout
RUN go build -o linux_logout
CMD ["./linux_logout"]


