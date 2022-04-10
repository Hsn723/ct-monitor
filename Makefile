VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"

KIND_VERSION = 0.11.1
KUBERNETES_VERSION = 1.22.1
CST_VERSION = 1.10.0

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
	go test --tags=test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: setup-container-structure-test
setup-container-structure-test:
	if [ -z "$(shell which container-structure-test)" ]; then \
		curl -LO https://storage.googleapis.com/container-structure-test/v$(CST_VERSION)/container-structure-test-linux-amd64 && mv container-structure-test-linux-amd64 container-structure-test && chmod +x container-structure-test && sudo mv container-structure-test /usr/local/bin/; \
	fi

.PHONY: container-structure-test
container-structure-test: setup-container-structure-test
	printf "amd64\narm64" | xargs -n1 -I {} container-structure-test test --image ghcr.io/hsn723/ct-monitor:$(shell git describe --tags --abbrev=0)-next-{} --config cst.yaml

.PHONY: setup-kind
setup-kind:
	curl -sSLf -o /tmp/kubectl -O https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	sudo install /tmp/kubectl /usr/local/bin/kubectl
	go install sigs.k8s.io/kind@v$(KIND_VERSION)

.PHONY: start-kind
start-kind:
	kind create cluster --name=ct-monitor-kindtest

.PHONY: stop-kind
stop-kind:
	kind delete cluster --name=ct-monitor-kindtest

.PHONY: run-kindtest
run-kindtest: build
	go test --tags=e2e -count=1 -coverprofile e2e.out -race -p 4 -v ./...

.PHONY: kindtest
kindtest: clean stop-kind start-kind run-kindtest

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean
	env CGO_ENABLED=0 go build $(LDFLAGS) .
