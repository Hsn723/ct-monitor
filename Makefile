VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"

KIND_VERSION = 0.11.1
KUBERNETES_VERSION = 1.21.2

all: build

.PHONY: clean
clean:
	if [ -f ct-monitor ]; then rm ct-monitor; fi

.PHONY: lint
lint:
	if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

.PHONY: test
test:
	go test -race -v $$(go list ./... | grep -v test)

.PHONY: setup-kind
setup-kind:
	curl -sSLf -O https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	sudo install kubectl /usr/local/bin/kubectl
	go install sigs.k8s.io/kind@v$(KIND_VERSION)

.PHONY: start-kind
start-kind:
	kind create cluster --name=ct-monitor-kindtest

.PHONY: stop-kind
stop-kind:
	kind delete cluster --name=ct-monitor-kindtest

.PHONY: kindtest
kindtest: clean stop-kind start-kind build
	go test -race -v ./test

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean
	env CGO_ENABLED=0 go build $(LDFLAGS) .
