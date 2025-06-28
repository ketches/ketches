IMAGE_NAME ?= registry.cn-hangzhou.aliyuncs.com/ketches/ketches-ui
IMAGE_TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: build all

build:
	docker buildx create --use --name mybuilder 2>/dev/null || docker buildx use mybuilder
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME):$(IMAGE_TAG) --push . -f Dockerfile

all: build