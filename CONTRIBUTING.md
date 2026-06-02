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
| `internal/proxy` | HTTP servers + the request pipeline: handler, gateway (OpenAI inbound), transformer, streaming, cascade, privacy firewall, semantic cache, rules, usage |
| `internal/router` | task classifier, routing strategies, cheap-first cascade chain, adaptive learned ordering |
| `internal/providers` | the `Provider` interface + 30 built-ins, a config-driven generic provider, enterprise providers (Azure/Vertex/Bedrock + SigV4), cache-aware + off-peak pricing |
| `internal/storage` | pure-Go SQLite: request log, stats, savings, leaderboard, detail |
| `internal/dashboard` | dashboard server, SSE broker, JSON API, embedded Svelte build |
| `internal/config` | TOML config, `env:` key resolution, env auto-discovery |
| `web/` | Svelte dashboard source (rebuilt + re-embedded by `make build`) |

See **[docs/architecture.md](docs/architecture.md)** for the full request pipeline.

## Areas that need help

- More provider integrations + keeping model maps / pricing current
- Smarter complexity classification (the classifier is intentionally simple)
- A stronger quality signal for `nexus bench` (today's "agreement" is a local
  word-cosine proxy — an optional local cross-encoder/judge would be great)
- More privacy-firewall detectors (secret shapes, languages)
- Dashboard components & charts; more MCP tools
- Testing on more platforms / more real providers

## Pull requests

1. Fork & branch from `main`.
2. Keep changes focused; add tests.
3. Run `go test ./...` and `go vet ./...` before pushing.
4. Open a PR with a clear description.

Conventional commit prefixes (`feat:`, `fix:`, `docs:`, `chore:`) are
appreciated — the changelog is generated from them.
