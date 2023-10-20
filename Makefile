# REGISTRY ?= docker.io
REGISTRY ?= registry.cn-hangzhou.aliyuncs.com
CONTROLLER_MANAGER_IMG ?= ${REGISTRY}/ketches/controller-manager:latest
API_SERVER_IMG ?= ${REGISTRY}/ketches/api-server:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.27.1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate ,WebhookConfiguration ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: udpate-codegen
update-codegen: ## Update clientset, listers, informers etc. after API changes.
	hack/update_codegen.sh

.PHONE: update-codegen-verify
update-codegen-verify: ## Verify that codegen is up to date.
	hack/verify_codegen.sh

.PHONY: swag-init
swag-init: ## Generate swagger docm.
	swag init -g ./cmd/api-server/main.go -o ./openapi

.PHONY: mockgen
mockgen: ## Generate mocks.
	hack/mockgen.sh

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build-controller-manager
build-controller-manager: manifests generate fmt vet ## Build controller-manager binary.
	go build -o bin/ketches-controller-manager ./cmd/controller-manager/main.go

.PHONY: build-api-server
build-api-server: manifests generate fmt vet ## Build api-server binary.
	go build -o bin/ketches-api-server ./cmd/api-server/main.go

.PHONY: run-controller-manager
run-controller-manager: manifests generate fmt vet ## Run a controller-manager from your host.
	ENABLE_WEBHOOKS=false go run ./cmd/controller-manager/main.go

.PHONY: run-api-server
run-api-server: manifests generate fmt vet ## Run a api-server from your host.
	go run ./cmd/api-server/main.go

.PHONY: docker-build
docker-build: test docker-build-controller-manager docker-build-api-server ## Build all docker images.

.PHONY: docker-build-controller-manager
docker-build-controller-manager: test ## Build controller-manager docker image.
	$(CONTAINER_TOOL) build -t ${CONTROLLER_MANAGER_IMG} -f build/api-server/Dockerfile .

.PHONY: docker-build-api-server
docker-build-api-server: test ## Build api-server docker image.
	$(CONTAINER_TOOL) build -t ${API_SERVER_IMG} -f build/api-server/Dockerfile .

.PHONY: docker-build-local
docker-build: test docker-build-local-controller-manager docker-build-local-api-server ## Build all docker images locally.

.PHONY: docker-build-local-controller-manager
docker-build-local-controller-manager: test ## Build controller-manager docker image locally.
	hack/build_binaries.sh controller-manager
	$(CONTAINER_TOOL) build -t ${CONTROLLER_MANAGER_IMG} -f build/api-server/Dockerfile.local .

.PHONY: docker-build-local-api-server
docker-build-local-api-server: test ## Build api-server docker image locally.
	hack/build_binaries.sh api-server
	$(CONTAINER_TOOL) build -t ${API_SERVER_IMG} -f build/api-server/Dockerfile.local .

.PHONY: docker-push
docker-push: docker-push-controller-manager docker-push-api-server ## Push all docker images.

.PHONY: docker-push-controller-manager
docker-push-controller-manager: ## Push controller-manager docker image.
	$(CONTAINER_TOOL) push ${CONTROLLER_MANAGER_IMG}

.PHONY: docker-push-api-server
docker-push-api-server: ## Push api-server docker image.
	$(CONTAINER_TOOL) push ${API_SERVER_IMG}

PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx ## Build all docker images using buildx.
docker-buildx: test docker-buildx-controller-manager docker-buildx-api-server

.PHONY: docker-buildx-controller-manager
docker-buildx-controller-manager: test ## Build controller-manager docker image using buildx.
	- $(CONTAINER_TOOL) buildx create --name ketches-builder
	$(CONTAINER_TOOL) buildx use ketches-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${CONTROLLER_MANAGER_IMG} -f build/controller-manager/Dockerfile .
	-# $(CONTAINER_TOOL) buildx rm ketches-builder

.PHONY: docker-buildx-api-server
docker-buildx-api-server: test ## Build api-server docker image using buildx.
	- $(CONTAINER_TOOL) buildx create --name ketches-builder
	$(CONTAINER_TOOL) buildx use ketches-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${API_SERVER_IMG} -f build/api-server/Dockerfile .
	-# $(CONTAINER_TOOL) buildx rm ketches-builder

.PHONY: docker-local-buildx
docker-local-buildx: docker-local-buildx-controller-manager docker-local-buildx-api-server ## Build all docker images locally using buildx.

.PHONY: docker-local-buildx-controller-manager
docker-local-buildx-controller-manager: ## Build controller-manager docker image locally using buildx.
	hack/build_binaries.sh controller-manager
	- $(CONTAINER_TOOL) buildx create --name ketches-builder
	$(CONTAINER_TOOL) buildx use ketches-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${CONTROLLER_MANAGER_IMG} -f build/controller-manager/Dockerfile.local .
	- rm -rf .out/controller-manager
	-# $(CONTAINER_TOOL) buildx rm ketches-builder

.PHONY: docker-local-buildx-api-server
docker-local-buildx-api-server: ## Build api-server docker image locally using buildx.
	hack/build_binaries.sh api-server
	- $(CONTAINER_TOOL) buildx create --name ketches-builder
	$(CONTAINER_TOOL) buildx use ketches-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${API_SERVER_IMG} -f build/api-server/Dockerfile.local .
	- rm -rf .out/api-server
	-# $(CONTAINER_TOOL) buildx rm ketches-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${CONTROLLER_MANAGER_IMG}
	#cd config/api-server && $(KUSTOMIZE) edit set image api-server=${API_SERVER_IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.12.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
