BUILD_DIR=./artifacts
WORK_DIR=./bin
VERSION := $(shell cat VERSION)
LDFLAGS=-ldflags "-X github.com/Hsn723/ct-monitor/cmd.CurrentVersion=${VERSION}"
OS ?= linux
ARCH ?= amd64
ifeq ($(OS), windows)
EXT = .exe
endif

all: build

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR} ${WORK_DIR}

.PHONY: setup
setup:
	mkdir -p ${BUILD_DIR} ${WORK_DIR}

.PHONY: lint
lint: clean setup
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b ${WORK_DIR} v1.31.0
	${WORK_DIR}/golangci-lint run

.PHONY: test
test: clean
	go test -race -v ./...

.PHONY: build
build: clean setup
	env GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/ct-monitor-$(OS)-$(ARCH)$(EXT) .
