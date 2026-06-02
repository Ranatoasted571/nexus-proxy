# NEXUS Makefile

BINARY_NAME=nexus
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w"

.PHONY: all dev build build-web embed clean test release help

## help: show this help
help:
	@grep -E '^##' Makefile | sed 's/## //'

## dev: start development mode (Go proxy + Vite dev server)
dev:
	@echo "→ Starting NEXUS in development mode..."
	@$(MAKE) -j2 dev-go dev-web

dev-go:
	@go run ./cmd/nexus start --dev

dev-web:
	@cd web && npm run dev

## build-web: build Svelte dashboard
build-web:
	@echo "→ Building dashboard..."
	@cd web && npm ci && npm run build
	@echo "✓ Dashboard built to web/dist/"

## embed: copy the built dashboard into the embed dir (run after build-web)
embed: build-web
	@echo "→ Embedding dashboard into binary..."
	@rm -rf internal/dashboard/dist
	@mkdir -p internal/dashboard/dist
	@cp -r web/dist/* internal/dashboard/dist/
	@echo "✓ Dashboard embedded into internal/dashboard/dist/"

## build: compile nexus binary (includes embed)
build: embed
	@echo "→ Compiling nexus $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/nexus
	@echo "✓ Binary: bin/$(BINARY_NAME)"

## build-all: cross-compile for all platforms
build-all: embed
	@echo "→ Cross-compiling for all platforms..."
	@GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/nexus-linux-amd64 ./cmd/nexus
	@GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o bin/nexus-linux-arm64 ./cmd/nexus
	@GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/nexus-darwin-amd64 ./cmd/nexus
	@GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/nexus-darwin-arm64 ./cmd/nexus
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/nexus-windows-amd64.exe ./cmd/nexus
	@echo "✓ All binaries in bin/"

## test: run tests
test:
	@go test ./... -v -race

## lint: run linter
lint:
	@golangci-lint run ./...

## clean: remove build artifacts
clean:
	@rm -rf bin/ web/dist/ web/node_modules/
	@echo "✓ Cleaned"

## install: install binary to /usr/local/bin
install: build
	@cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Installed to /usr/local/bin/$(BINARY_NAME)"

## release: create GitHub release with GoReleaser
release:
	@goreleaser release --clean

## snapshot: test release without publishing
snapshot:
	@goreleaser release --snapshot --clean

## setup-web: install web dependencies
setup-web:
	@cd web && npm install

## setup: install all dev dependencies
setup: setup-web
	@go mod download
	@which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@which goreleaser || go install github.com/goreleaser/goreleaser/v2@latest
	@echo "✓ Dev setup complete"
