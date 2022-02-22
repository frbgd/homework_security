FROM golang:1.17-alpine

RUN apk add openssl

WORKDIR /app

COPY . .

RUN ./init.sh

RUN go build -o /app/proxy main.go

cmd ["/app/proxy"]

EXPOSE 8080
