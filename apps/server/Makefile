# croffle-server — Go toolchain entry points.
#
# The repository root is a pnpm workspace; this app is deliberately not part of
# it (no package.json here), so Go tasks live in this Makefile instead of
# package scripts.

BIN     := bin/croffle-server
PKG     := ./cmd/croffle-server
VERSION ?= $(shell git describe --tags --match 'server-v*' --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

# Local runs use a database in the working directory; the default
# (/var/lib/croffle/server.db) is meant for the container image.
RUN_DB  ?= ./server.db

.PHONY: all build run test vet fmt fmt-check check docker clean

all: check build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

run:
	CROFFLE_DB_PATH=$(RUN_DB) go run -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed in:"; echo "$$out"; exit 1; fi

# The same gates CI runs (work item #11).
check: fmt-check vet test

docker:
	@if [ ! -f Dockerfile ]; then \
		echo "Dockerfile arrives in work item server-0.1.0#11"; exit 1; \
	fi
	docker build -t croffle-server:$(VERSION) .

clean:
	rm -rf bin
