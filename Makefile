BUILD_DIR=/tmp/ct-monitor/artifacts
VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X github.com/Hsn723/ct-monitor/cmd.CurrentVersion=${VERSION}"
OS ?= linux
ARCH ?= amd64
ifeq ($(OS), windows)
EXT = .exe
endif

KIND_VERSION = 0.9.0
KUBERNETES_VERSION = 1.18.2

all: build

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

.PHONY: setup
setup:
	mkdir -p ${BUILD_DIR}
	pip3 install pre-commit
	pre-commit install

.PHONY: lint
lint: clean setup
	pre-commit run --all-files

.PHONY: test
test: clean
	go test -race -v $$(go list ./... | grep -v test)

.PHONY: setup-kind
setup-kind:
	curl -sSLf -O https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	sudo install kubectl /usr/local/bin/kubectl
	cd /tmp; env GOFLAGS= GO111MODULE=on go get sigs.k8s.io/kind@v$(KIND_VERSION)

.PHONY: start-kind
start-kind:
	kind create cluster --name=ct-monitor-kindtest

.PHONY: stop-kind
stop-kind:
	kind delete cluster --name=ct-monitor-kindtest

.PHONY: kindtest
kindtest: clean stop-kind start-kind
	go test -race -v ./test

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean setup
	env GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/ct-monitor-$(OS)-$(ARCH)$(EXT) .
