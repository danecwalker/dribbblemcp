.PHONY: build install run doctor clean test

BINARY ?= dribbblemcp
PREFIX ?= $(HOME)/.local/bin

build:
	go build -o bin/$(BINARY) ./cmd/dribbblemcp

install: build
	mkdir -p $(PREFIX)
	cp bin/$(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed to $(PREFIX)/$(BINARY)"
	@echo "Uses system Chrome/Chromium via chromedp (set CHROME_PATH to override)."

run: build
	./bin/$(BINARY)

# Quick live smoke test (requires network + Chrome)
doctor: build
	@echo "Running unit + integration smoke tests…"
	go test ./internal/images/ -count=1
	go test ./internal/dribbble/ -tags=integration -count=1 -timeout 3m

clean:
	rm -rf bin/

test:
	go test ./internal/images/ -count=1

test-integration:
	go test ./internal/dribbble/ -tags=integration -count=1 -timeout 3m
