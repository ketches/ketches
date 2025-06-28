IMAGE_NAME_PREFIX ?= registry.cn-hangzhou.aliyuncs.com/ketches/ketches
IMAGE_TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: build all

build:
	docker buildx create --use --name mybuilder 2>/dev/null || docker buildx use mybuilder

	# Backend build
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME_PREFIX)-api:$(IMAGE_TAG) --push ./backend -f backend/Dockerfile

	# Frontend build
	docker buildx build --platform $(PLATFORMS) -t $(IMAGE_NAME_PREFIX)-ui:$(IMAGE_TAG) --push ./frontend -f frontend/Dockerfile

all: build