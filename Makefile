# ──────────────────────────────────────────────────────────────────────────────
# Project metadata
# ──────────────────────────────────────────────────────────────────────────────
BINARY_NAME  := ketches
MODULE       := github.com/ketches/ketches
VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME   := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS      := -ldflags "-w -s"

# ──────────────────────────────────────────────────────────────────────────────
# Docker / registry
# ──────────────────────────────────────────────────────────────────────────────
REGISTRY     := ghcr.io
ORG          := ketches
API_IMAGE    := $(REGISTRY)/$(ORG)/ketches-api
UI_IMAGE     := $(REGISTRY)/$(ORG)/ketches-ui
PLATFORMS    := linux/amd64,linux/arm64

# ──────────────────────────────────────────────────────────────────────────────
# Phony targets
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: all build build-ui build-all \
        test test-coverage \
        lint lint-ui \
        run dev-ui \
        docker-build docker-build-api docker-build-ui \
        docker-push docker-push-api docker-push-ui \
        docker-buildx \
        up down logs \
        clean help

all: build

# ──────────────────────────────────────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────────────────────────────────────
build: ## Build the backend binary (requires GCC for CGO/SQLite)
	@mkdir -p bin
	CGO_ENABLED=1 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/api

build-ui: ## Build the frontend (requires Node.js)
	cd ui && npm run build

build-all: build build-ui ## Build both backend and frontend

# ──────────────────────────────────────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────────────────────────────────────
test: ## Run Go tests with race detector
	go test -race -coverprofile=coverage.out ./...

test-coverage: test ## Open test coverage report in browser
	go tool cover -html=coverage.out

# ──────────────────────────────────────────────────────────────────────────────
# Lint
# ──────────────────────────────────────────────────────────────────────────────
lint: ## Lint Go source code (requires golangci-lint)
	golangci-lint run ./...

lint-ui: ## Lint frontend source code
	cd ui && npm run lint

# ──────────────────────────────────────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────────────────────────────────────
run: ## Run the backend API locally (reads .env)
	go run ./cmd/api

dev-ui: ## Start the frontend Vite dev server
	cd ui && npm run dev

# ──────────────────────────────────────────────────────────────────────────────
# Docker — single-arch (fast local builds)
# ──────────────────────────────────────────────────────────────────────────────
docker-build: docker-build-api docker-build-ui ## Build all Docker images

docker-build-api: ## Build the backend Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(API_IMAGE):$(VERSION) \
		-t $(API_IMAGE):latest \
		.

docker-build-ui: ## Build the frontend Docker image
	docker build \
		-t $(UI_IMAGE):$(VERSION) \
		-t $(UI_IMAGE):latest \
		./ui

docker-push: docker-push-api docker-push-ui ## Push all Docker images to the registry

docker-push-api: ## Push the backend Docker image
	docker push $(API_IMAGE):$(VERSION)
	docker push $(API_IMAGE):latest

docker-push-ui: ## Push the frontend Docker image
	docker push $(UI_IMAGE):$(VERSION)
	docker push $(UI_IMAGE):latest

# ──────────────────────────────────────────────────────────────────────────────
# Docker — multi-arch (CI / release)
# ──────────────────────────────────────────────────────────────────────────────
docker-buildx: ## Build and push multi-arch images via buildx (requires a buildx builder)
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(API_IMAGE):$(VERSION) \
		-t $(API_IMAGE):latest \
		--push \
		.
	docker buildx build \
		--platform $(PLATFORMS) \
		-t $(UI_IMAGE):$(VERSION) \
		-t $(UI_IMAGE):latest \
		--push \
		./ui

# ──────────────────────────────────────────────────────────────────────────────
# Docker Compose
# ──────────────────────────────────────────────────────────────────────────────
up: ## Start all services (builds images if needed)
	docker compose up -d --build

down: ## Stop and remove all services
	docker compose down

logs: ## Tail logs from all services
	docker compose logs -f

# ──────────────────────────────────────────────────────────────────────────────
# Cleanup
# ──────────────────────────────────────────────────────────────────────────────
clean: ## Remove build artifacts
	rm -rf bin/ coverage.out ui/dist/

# ──────────────────────────────────────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────────────────────────────────────
help: ## Print this help message
	@printf "\033[1mKetches\033[0m — Kubernetes Application Management Platform\n\n"
	@printf "\033[33mUsage:\033[0m\n  make <target>\n\n"
	@printf "\033[33mTargets:\033[0m\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
