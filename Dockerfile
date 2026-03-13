# syntax=docker/dockerfile:1

# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# CGO is required for the SQLite driver. On arm64, Go may invoke gcc with
# -fuse-ld=gold during external linking, so ld.gold must be available too.
RUN apk add --no-cache gcc musl-dev binutils binutils-gold

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG TAG=
ARG BUILD_TIME

RUN CGO_ENABLED=1 GOOS=linux go build \
    -p=1 \
    -ldflags="-w -s \
    -X github.com/ketches/ketches/internal/app.Version=${VERSION} \
    -X github.com/ketches/ketches/internal/app.Commit=${COMMIT} \
    -X github.com/ketches/ketches/internal/app.BuildTime=${BUILD_TIME} \
    -X github.com/ketches/ketches/internal/app.Tag=${TAG}" \
    -o ketches \
    ./cmd/api

# ─── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/ketches .

EXPOSE 8080

ENTRYPOINT ["./ketches"]
