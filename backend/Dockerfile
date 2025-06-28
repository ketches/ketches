# More base image detail: https://github.com/ketches/golang-sqlite-builder-image/blob/master/Dockerfile
FROM registry.cn-hangzhou.aliyuncs.com/ketches/golang-sqlite-builer-image:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
COPY openapi/ ./openapi/

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags='-s -w -extldflags "-static"' -o ./bin/ketches-api cmd/api/main.go

# Deploy the application binary into a lean image
FROM alpine:latest

WORKDIR /ketches

COPY --from=builder /app/bin/ketches-api ./ketches-api

ENTRYPOINT ["./ketches-api"]