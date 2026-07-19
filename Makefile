.PHONY: help build install run test test-integration doctor clean release release-snapshot fmt vet

BINARY ?= dribbblemcp
PREFIX ?= $(HOME)/.local/bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE) \
	-X main.builtBy=make

help: ## Show targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build binary for this platform
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/dribbblemcp
	@echo "built bin/$(BINARY) ($(VERSION))"

install: build ## Install to $(PREFIX)
	mkdir -p $(PREFIX)
	install -m 755 bin/$(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed $(PREFIX)/$(BINARY)"
	@echo "Configure: grok mcp add dribbble -- $(PREFIX)/$(BINARY)"

run: build ## Run MCP server (stdio)
	./bin/$(BINARY)

test: ## Unit tests
	go test -race -count=1 ./internal/images/...

test-integration: ## Live Dribbble tests (Chrome + network)
	go test ./internal/dribbble/ -tags=integration -count=1 -timeout 3m

doctor: build test ## Build + unit tests + version
	./bin/$(BINARY) --version

fmt: ## gofmt
	gofmt -s -w .

vet: ## go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -rf bin/ dist/

release-snapshot: ## GoReleaser snapshot (local, no publish)
	goreleaser release --snapshot --clean

release: ## GoReleaser publish (requires tag + GITHUB_TOKEN)
	goreleaser release --clean
