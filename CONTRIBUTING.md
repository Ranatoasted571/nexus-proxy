# Contributing to NEXUS

Thanks for your interest in making NEXUS better! 🎉

## Dev setup

Requirements: **Go 1.22+** and **Node 20+** (only needed to rebuild the dashboard).

```bash
git clone https://github.com/lynuxis2026-pixel/nexus-proxy.git
cd nexus-proxy
make setup        # installs web deps + Go tooling

# Run everything in dev mode (Go proxy + Vite dev server)
make dev
```

## Building

```bash
make build        # builds the Svelte dashboard, embeds it, compiles the binary
./bin/nexus start
```

Plain `go build ./cmd/nexus` also works — the embedded dashboard lives in
`internal/dashboard/dist/` (committed), so a build never requires Node.
`make build` regenerates that directory from `web/`.

## Tests

```bash
go test ./...            # unit tests
go test ./... -race      # with the race detector
go vet ./...
```

Please add tests for new behavior. The core logic packages
(`router`, `storage`, `config`, `providers`, `proxy`) all have unit tests.

## Project layout

| Path | What |
|------|------|
| `cmd/nexus` | CLI entrypoint (cobra) |
| `internal/proxy` | HTTP proxy, request handler, format conversion, streaming |
| `internal/router` | task classifier + routing strategies |
| `internal/providers` | provider implementations (Anthropic, DeepSeek, Groq, Gemini, Ollama) |
| `internal/storage` | SQLite logging + stats |
| `internal/dashboard` | dashboard server, SSE broker, embedded UI |
| `web/` | Svelte dashboard source |

## Areas that need help

- More provider integrations (Mistral, Together AI, OpenRouter, Cohere)
- Token-by-token streaming for OpenAI-compatible providers
- Smarter complexity classification
- Dashboard components & charts
- Windows testing

## Pull requests

1. Fork & branch from `main`.
2. Keep changes focused; add tests.
3. Run `go test ./...` and `go vet ./...` before pushing.
4. Open a PR with a clear description.

Conventional commit prefixes (`feat:`, `fix:`, `docs:`, `chore:`) are
appreciated — the changelog is generated from them.
