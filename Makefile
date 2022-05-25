PROJECT = ct-monitor
VERSION = $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"

KIND_VERSION = 0.11.1
KUBERNETES_VERSION = 1.22.1
CST_VERSION = 1.10.0

WORKDIR = /tmp/$(PROJECT)/work
BINDIR = /tmp/$(PROJECT)/bin
CONTAINER_STRUCTURE_TEST = $(BINDIR)/container-structure-test
GINKGO = $(BINDIR)/ginkgo
KIND = $(BINDIR)/kind
KUBECTL = $(BINDIR)/kubectl

PATH := $(PATH):$(BINDIR)

export PATH

all: build

.PHONY: clean
clean:
	@if [ -f $(PROJECT) ]; then rm $(PROJECT); fi

.PHONY: lint
lint:
	@if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

.PHONY: test
test: build-testfilter
	go test --tags=test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: build-testfilter
build-testfilter: $(WORKDIR)
	env CGO_ENABLED=0 go build --tags=testfilter $(LDFLAGS) -o $(WORKDIR)/testfilter ./filter/t/main.go

.PHONY: $(CONTAINER_STRUCTURE_TEST)
$(CONTAINER_STRUCTURE_TEST): $(BINDIR)
	curl -sSLf -o $(CONTAINER_STRUCTURE_TEST) -O https://storage.googleapis.com/container-structure-test/v$(CST_VERSION)/container-structure-test-linux-amd64 && chmod +x $(CONTAINER_STRUCTURE_TEST)

.PHONY: container-structure-test
container-structure-test: $(CONTAINER_STRUCTURE_TEST)
	printf "amd64\narm64" | xargs -n1 -I {} $(CONTAINER_STRUCTURE_TEST) test --image ghcr.io/hsn723/$(PROJECT):$(shell git describe --tags --abbrev=0)-next-{} --config cst.yaml

.PHONY: setup-kind
setup-kind: $(BINDIR) $(KUBECTL)
	GOBIN=$(BINDIR) go install github.com/onsi/ginkgo/v2/ginkgo@latest
	GOBIN=$(BINDIR) go install sigs.k8s.io/kind@v$(KIND_VERSION)

.PHONY: $(KUBECTL)
$(KUBECTL): $(BINDIR)
	curl -sSLf -o $(KUBECTL) -O https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl

.PHONY: start-kind
start-kind:
	$(KIND) create cluster --name=$(PROJECT)-kindtest

.PHONY: stop-kind
stop-kind:
	$(KIND) delete cluster --name=$(PROJECT)-kindtest

.PHONY: run-kindtest
run-kindtest: build
	$(GINKGO) --tags=e2e -race -v ./... -ginkgo.progress -ginkgo.v -ginkgo.fail-fast

.PHONY: kindtest
kindtest: clean stop-kind start-kind run-kindtest

.PHONY: verify
verify:
	go mod download
	go mod verify

.PHONY: build
build: clean
	env CGO_ENABLED=0 go build $(LDFLAGS) .

$(BINDIR):
	mkdir -p $(BINDIR)

$(WORKDIR):
	mkdir -p $(WORKDIR)
