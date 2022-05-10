FROM golang:1.18 AS builder

MAINTAINER "pescoding@outlook.com"

WORKDIR /app
COPY . .
ENV GOPROXY https://goproxy.io
RUN go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /ketches ./cmd/main.go 

FROM alpine:3.10
ENV TZ=Asia/Shanghai
COPY --from=builder /ketches /ketches
ENTRYPOINT ["/ketches"]