FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod .

COPY main.go .

RUN go build -o proxy

cmd ["/app/proxy"]

EXPOSE 8080
