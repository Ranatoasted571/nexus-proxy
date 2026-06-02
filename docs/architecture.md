# NEXUS Architecture

NEXUS is one Go binary that runs entirely on your machine: a proxy server, a
dashboard server, and the embedded web UI, sharing a local SQLite database.

```
        Claude Code / Cursor / aider / any OpenAI or Anthropic SDK
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│                            NEXUS  (single binary)                      │
│                                                                        │
│  Proxy :3000                                   Dashboard :2222         │
│  ├─ POST /v1/messages          (Anthropic)     ├─ GET /  (Svelte SPA)  │
│  └─ POST /v1/chat/completions  (OpenAI)        ├─ GET /api/*  (stats,  │
│                                                │     requests, savings,│
│   request pipeline (per call):                 │     report, leaderbd) │
│   1. 🔒 privacy firewall  — mask secrets/PII   └─ GET /events  (SSE)   │
│   2. ⚡ cache lookup      — exact + semantic                            │
│   3. 🧭 classify          — simple/standard/complex/critical            │
│   4. 🧩 rules / headers   — X-Nexus-Tier / X-Nexus-Provider / config    │
│   5. 🛡️ cost guardrail    — downgrade pricey single requests            │
│   6. 🪜 cascade or chain  — cheap-first + verify + escalate / failover  │
│   7. 🗝️ key rotation      — round-robin a provider's key pool (429→next)│
│   8. 🔁 transform         — Anthropic ↔ OpenAI (both directions)        │
│   9. ↩ restore           — un-redact secrets in the response           │
│  10. 🧠 record outcome    — adaptive routing learns per task type       │
│  11. 💾 log + cost        — cache-aware (off-peak) cost → SQLite        │
│  12. 📡 SSE push          — live dashboard update                       │
└──────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼   cheapest capable provider
   Anthropic · OpenAI · Groq · Gemini · DeepSeek · Mistral · Cohere · xAI
   Together · Fireworks · OpenRouter · Cerebras · SambaNova · NVIDIA · …
   Azure · Bedrock (SigV4) · Vertex · Ollama (local) · any OpenAI-compatible
```

## Packages

| Package | Responsibility |
|---------|----------------|
| `cmd/nexus` | Cobra CLI: `start`, `code`, `add`, `doctor`, `top`, `bench`, `report`, `mcp`, `status`, `logs`, `cost`, `models`, `config`, `version` |
| `internal/proxy` | HTTP servers + the request pipeline: `handler`, `gateway` (OpenAI inbound), `transformer`, `stream`, `openai`, `cascade`, `firewall`, `semantic`, `cache`, `rules`, `usage` |
| `internal/router` | Task classifier, routing strategies (auto/cheapest/fastest/manual), cheap-first `CascadeChain`, and adaptive learned ordering |
| `internal/providers` | The `Provider` interface + 30 built-ins, a config-driven generic provider, enterprise providers (Azure/Vertex/Bedrock + SigV4), pricing (cache-aware + off-peak) |
| `internal/storage` | Pure-Go SQLite: request log, aggregated stats, savings, leaderboard, request detail |
| `internal/dashboard` | SSE broker, JSON API, embedded Svelte build (`go:embed`) |
| `internal/config` | TOML config, `env:` key resolution, env auto-discovery |

## Privacy model

- NEXUS runs locally. The only data that leaves your machine is the LLM call to
  the provider **you** configured.
- With the privacy firewall on (`--redact`), detected secrets/API keys/PII are
  replaced with reversible placeholders **before** the request leaves, and the
  originals are restored in the response (even across SSE chunks) by a
  carry-buffered writer. The redaction count is recorded for `nexus report`.
- No telemetry. Keys live in `~/.nexus/config.toml` (or `env:` refs). Route to
  Ollama for fully offline operation.

## Key design decisions

- **Single binary.** Proxy + dashboard + embedded UI in one Go binary — no
  Python, Docker, Postgres, Redis or cloud. Lowers the trust + ops surface.
- **Pure-Go SQLite** (`modernc.org/sqlite`) — cross-compiles with no C compiler.
- **Dual API.** Speaks both `/v1/messages` (Anthropic) and `/v1/chat/completions`
  (OpenAI), so every AI coding tool routes through one proxy with one env var.
- **SSE over WebSockets** for the one-directional live dashboard stream.
- **Cheap-first + verify** rather than predict-then-commit: try the cheapest
  capable model, verify its output, escalate only on failure.
- **Learn locally.** Adaptive routing learns from real verification outcomes on
  your own traffic — no labels, no cloud, no generic crowd preferences.

## Ports

| Port | Service |
|------|---------|
| 3000 | Proxy — Claude Code / tools point here |
| 2222 | Dashboard — open in a browser, or `nexus top` in a terminal |
