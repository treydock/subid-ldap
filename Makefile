export GOPATH ?= $(firstword $(subst :, ,$(shell go env GOPATH)))
GOHOSTOS     ?= $(shell go env GOHOSTOS)
GOHOSTARCH   ?= $(shell go env GOHOSTARCH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint
GOLANGCI_LINT_VERSION := v2.11.4
VERSION ?= $(shell git describe --tags --abbrev=0 || git rev-parse --short HEAD)
GITSHA := $(shell git rev-parse HEAD)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
BUILDUSER := $(shell whoami)@$(shell hostname)
BUILDDATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
export GO111MODULE=auto

all: unused lint style test

build:
	GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) CGO_ENABLED=0 go build -ldflags="\
	-X github.com/prometheus/common/version.Version=$(VERSION) \
	-X github.com/prometheus/common/version.Revision=$(GITSHA) \
	-X github.com/prometheus/common/version.Branch=$(GITBRANCH) \
	-X github.com/prometheus/common/version.BuildUser=$(BUILDUSER) \
	-X github.com/prometheus/common/version.BuildDate=$(BUILDDATE)" \
	-o subid-ldap cmd/subid-ldap/main.go

test:
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go test $(test-flags) ./...

coverage:
	GO111MODULE=on GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go test $(test-flags) -coverpkg=./... -coverprofile=coverage.txt -covermode=atomic ./...

unused:
	@echo ">> running check for unused/missing packages in go.mod"
	GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go mod tidy
	@git diff --exit-code -- go.sum go.mod

lint: $(GOLANGCI_LINT)
	@echo ">> running golangci-lint"
	GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) go list -e -compiled -test=true -export=false -deps=true -find=false -tags= -- ./... > /dev/null
	GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GOLANGCI_LINT) run ./...

style:
	@echo ">> checking code style"
	@fmtRes=$$(gofmt -d $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	if [ -n "$${fmtRes}" ]; then \
		echo "gofmt checking failed!"; echo "$${fmtRes}"; echo; \
		echo "Please ensure you are using $$($(GO) version) for formatting code."; \
		exit 1; \
	fi

format:
	go fmt ./...

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)
