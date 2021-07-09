.PHONY: default check build clean

GO           ?= go
GOFMT        ?= $(GO)fmt
APP          := interpreter

VERSION ?= $(shell git describe --tag --always --dirty)
PROGVER ?= $(shell git describe --tags `git rev-list --tags --max-count=1` | tail -1 | sed 's/v\(.*\)/\1/')
BUILDTIME = $(shell date -u '+%c')
GITCOMMIT = $(shell git rev-parse --short HEAD)
GOBUILDFLAGS = -a -ldflags "-w -s -X 'main.BuildTime=$(BUILDTIME)' -X main.GitCommit=$(GITCOMMIT) -X main.Version=$(VERSION)" -o $(APP)

default: build

check:
	golangci-lint run -n | tee errors.txt

build:
	CGO_ENABLED=0 $(GO) build $(GOBUILDFLAGS)

clean:
	-rm -r .ignore/ $(APP) errors.txt report.json coverage.txt

