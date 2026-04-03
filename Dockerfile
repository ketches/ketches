# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG TAG=
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s \
    -X github.com/ketches/ketches/internal/app.Version=${VERSION} \
    -X github.com/ketches/ketches/internal/app.Commit=${COMMIT} \
    -X github.com/ketches/ketches/internal/app.BuildTime=${BUILD_TIME} \
    -X github.com/ketches/ketches/internal/app.Tag=${TAG}" \
    -o ketches \
    ./cmd/api

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S ketches && \
    adduser -S -D -h /app -G ketches ketches

WORKDIR /app

COPY --from=builder --chown=ketches:ketches /app/ketches .

USER ketches

EXPOSE 8080

ENTRYPOINT ["./ketches"]
