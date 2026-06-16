PROJECT = ct-monitor
VERSION = $(shell cat VERSION)
LDFLAGS=-ldflags "-w -s -X main.version=${VERSION}"

AQUA_VERSION = 2.60.1
GINKGO_VERSION = $(shell cat go.mod | grep "github.com/onsi/ginkgo/v2" | awk '{print $$2}' | tr -d 'v')

WORKDIR = /tmp/$(PROJECT)/work
BINDIR = /tmp/$(PROJECT)/bin
GINKGO = $(BINDIR)/ginkgo

PATH := ${HOME}/.local/share/aquaproj-aqua/bin:${BINDIR}:$(PATH)

export PATH

all: build

.PHONY: clean
clean:
	@if [ -f $(PROJECT) ]; then rm $(PROJECT); fi

.PHONY: lint
lint: init-aqua
	pre-commit install
	pre-commit run --all-files

.PHONY: test
test: build-testfilter
	go test --tags=test -coverprofile cover.out -count=1 -race -p 4 -v ./...

.PHONY: build-testfilter
build-testfilter: $(WORKDIR)
	env CGO_ENABLED=0 go build --tags=testfilter $(LDFLAGS) -o $(WORKDIR)/testfilter ./filter/t/main.go

.PHONY: container-structure-test
container-structure-test: init-aqua
	yq '.builds[0].goarch[]' .goreleaser.yml | xargs -n1 -I {} container-structure-test test --image ghcr.io/hsn723/$(PROJECT):$(shell git describe --tags --abbrev=0)-next-{} --platform linux/{} --config cst.yaml

.PHONY: setup-kind
setup-kind: $(BINDIR) init-aqua
	GOBIN=$(BINDIR) go install github.com/onsi/ginkgo/v2/ginkgo@v$(GINKGO_VERSION)

.PHONY: start-kind
start-kind:
	kind create cluster --name=$(PROJECT)-kindtest

.PHONY: stop-kind
stop-kind:
	kind delete cluster --name=$(PROJECT)-kindtest

.PHONY: run-kindtest
run-kindtest: build
	$(GINKGO) --tags=e2e --race -v --fail-fast ./...

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

.PHONY: init-aqua
init-aqua:
	@go install github.com/aquaproj/aqua/v2/cmd/aqua@v$(AQUA_VERSION)
	@aqua i -l

.PHONY: update-aqua
update-aqua:
	aqua update
	aqua update-checksum
