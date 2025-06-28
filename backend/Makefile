APP_NAME ?= ketches-api
IMAGE_NAME ?= registry.cn-hangzhou.aliyuncs.com/ketches/$(APP_NAME)
IMAGE_TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64


.PHONY: build run test swag
build: swag
	docker buildx create --use --name mybuilder 2>/dev/null || docker buildx use mybuilder
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME):$(IMAGE_TAG) --push . -f Dockerfile

run: swag
	go mod tidy
	go run cmd/api/main.go

test:
	go test ./...

swag:
	swag init -g cmd/api/main.go -o ./openapi