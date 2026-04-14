# ──────────────────────────────────────────────────────────────────────────────
# Project metadata
# ──────────────────────────────────────────────────────────────────────────────
BINARY_NAME  := ketches
MODULE       := github.com/ketches/ketches
VERSION      := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT       := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TAG          := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "")
BUILD_TIME   := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS      := -ldflags "-w -s \
	-X github.com/ketches/ketches/internal/app.Version=$(VERSION) \
	-X github.com/ketches/ketches/internal/app.Commit=$(COMMIT) \
	-X github.com/ketches/ketches/internal/app.BuildTime=$(BUILD_TIME) \
	-X github.com/ketches/ketches/internal/app.Tag=$(TAG)"

# ──────────────────────────────────────────────────────────────────────────────
# Docker / registry
# ──────────────────────────────────────────────────────────────────────────────
GHCR_REGISTRY      ?= ghcr.io
DOCKERHUB_REGISTRY ?= docker.io
ALIYUN_REGISTRY    ?= registry.cn-hangzhou.aliyuncs.com
ORG                ?= ketches
API_IMAGE    := ketches-api
UI_IMAGE     := ketches-ui
PLATFORMS    ?= linux/amd64,linux/arm64
BUILDX_BUILDER ?= ketches-builder

# ──────────────────────────────────────────────────────────────────────────────
# Phony targets
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: all build build-ui build-all \
        test test-coverage \
        lint lint-ui \
	run dev-ui openapi \
        docker-build docker-build-api docker-build-ui \
        docker-push docker-push-api docker-push-ui \
        docker-buildx docker-buildx-ensure-builder \
        up down logs \
        clean help

all: build

# ──────────────────────────────────────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────────────────────────────────────
build: ## Build the backend binary
	@mkdir -p bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/api

build-ui: ## Build the frontend (requires Node.js)
	cd ui && pnpm run build

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
	cd ui && pnpm run lint

# ──────────────────────────────────────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────────────────────────────────────
run: ## Run the backend API locally (reads .env)
	go run ./cmd/api

dev-ui: ## Start the frontend Vite dev server
	cd ui && pnpm run dev

openapi: ## Generate openapi/openapi.json and openapi/openapi.yaml
	go run ./cmd/openapi
	cd ui && pnpm run generate:api-types

# ──────────────────────────────────────────────────────────────────────────────
# Docker — single-arch (fast local builds)
# ──────────────────────────────────────────────────────────────────────────────
docker-build: docker-build-api docker-build-ui ## Build all Docker images

docker-build-api: ## Build the backend Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		.

docker-build-ui: ## Build the frontend Docker image
	docker build \
		-t $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		./ui

docker-push: docker-push-api docker-push-ui ## Push all Docker images to the registry

docker-push-api: ## Push the backend Docker image
	docker push $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION)
	docker push $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):latest
	docker push $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION)
	docker push $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):latest
	docker push $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION)
	docker push $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):latest

docker-push-ui: ## Push the frontend Docker image
	docker push $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION)
	docker push $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):latest
	docker push $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION)
	docker push $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):latest
	docker push $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION)
	docker push $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):latest

# ──────────────────────────────────────────────────────────────────────────────
# Docker — multi-arch (CI / release)
# ──────────────────────────────────────────────────────────────────────────────
docker-buildx: docker-buildx-api docker-buildx-ui ## Build and push multi-arch images via buildx

docker-buildx-ensure-builder: ## Ensure the multi-arch buildx builder is healthy
	@if docker buildx inspect $(BUILDX_BUILDER) >/dev/null 2>&1; then \
		docker buildx use $(BUILDX_BUILDER) >/dev/null; \
	else \
		echo "Creating buildx builder $(BUILDX_BUILDER)..."; \
		docker buildx create --name $(BUILDX_BUILDER) --use >/dev/null; \
	fi
	@if ! docker buildx inspect $(BUILDX_BUILDER) --bootstrap >/dev/null 2>&1; then \
		echo "Recreating stale buildx builder $(BUILDX_BUILDER)..."; \
		docker buildx rm -f $(BUILDX_BUILDER) >/dev/null 2>&1 || true; \
		docker buildx create --name $(BUILDX_BUILDER) --use >/dev/null; \
		docker buildx inspect $(BUILDX_BUILDER) --bootstrap >/dev/null; \
	fi

docker-buildx-api: docker-buildx-ensure-builder ## Build and push multi-arch images via buildx (requires a buildx builder)
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):$(VERSION) \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(API_IMAGE):latest \
		--push \
		.
docker-buildx-ui: docker-buildx-ensure-builder ## Build and push multi-arch frontend image via buildx
	docker buildx build \
		--platform $(PLATFORMS) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(GHCR_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(DOCKERHUB_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):$(VERSION) \
		-t $(ALIYUN_REGISTRY)/$(ORG)/$(UI_IMAGE):latest \
		--push \
		./ui

# ──────────────────────────────────────────────────────────────────────────────
# Docker Compose
# ──────────────────────────────────────────────────────────────────────────────
up: ## Start all services (builds images if needed)
	cd deploy/docker && docker compose up -d --build

down: ## Stop and remove all services
	cd deploy/docker && docker compose down

logs: ## Tail logs from all services
	cd deploy/docker && docker compose logs -f

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
