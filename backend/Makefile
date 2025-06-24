APP := ketches-api
IMAGE_NAME := registry.cn-hangzhou.aliyuncs.com/ketches/$(APP)
IMAGE_TAG := latest

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o _output/bin/$(APP)_linux_amd64 cmd/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o _output/bin/$(APP)_linux_arm64 cmd/main.go

	docker buildx build --platform linux/amd64,linux/arm64 -t ${IMAGE_NAME}:${IMAGE_TAG} --push . -f ./hack/docker/Dockerfile

.PHONY: run
run: swag
	go run cmd/api/main.go
.PHONY: test
test:
	go test ./...

.PHONY: swag
swag:
	swag init -g cmd/api/main.go -o ./openapi