# NEXUS Custom Claude Code Commands

## /sprint1
Start Sprint 1: Core proxy werkend

Implementeer in deze volgorde:
1. Run `go mod tidy` om dependencies te installeren
2. Implementeer `internal/proxy/server.go` volledig
3. Implementeer `internal/proxy/handler.go` - fix alle TODOs
4. Implementeer `internal/proxy/stream.go` - volledige streaming support
5. Wire alles samen in `cmd/nexus/main.go` - runStart functie
6. Test met: `go run ./cmd/nexus start`
7. Test Claude Code: `export ANTHROPIC_BASE_URL=http://localhost:3000 && claude --version`

Succes = Claude Code werkt via de proxy als simpele doorstuurder naar Anthropic.

## /sprint2
Start Sprint 2: Router + Storage

Implementeer in deze volgorde:
1. Implementeer SQLite opslag in `internal/storage/`
2. Implementeer classifier in `internal/router/classifier.go`
3. Implementeer router logica in `internal/router/router.go`
4. Koppel providers aan config (`~/.nexus/config.toml`)
5. Wire router in `internal/proxy/handler.go`
6. Implementeer `nexus add` command volledig
7. Test met twee providers geconfigureerd

Succes = requests gaan naar juiste provider op basis van complexiteit.

## /sprint3
Start Sprint 3: Dashboard

Implementeer in deze volgorde:
1. `cd web && npm install`
2. Bouw Svelte components in `web/src/components/`
3. Implementeer SSE consumer in dashboard
4. Implementeer `internal/dashboard/embed.go` met go:embed
5. Wire SSE broker in proxy handler (push events na elke request)
6. Run `make build` - test single binary met embedded dashboard
7. Open http://localhost:2222

Succes = live dashboard toont requests in real-time in single binary.

## /sprint4
Start Sprint 4: Polish & Release

1. Schrijf tests in `*_test.go` files
2. Verfijn README met echte screenshots/GIFs
3. Maak `install.sh` script
4. Test cross-compile met `make build-all`
5. Setup GoReleaser
6. Push naar GitHub
7. Post op Hacker News: "Show HN: NEXUS — smart proxy for Claude Code"

## /status
Geef huidige project status:
- Welke bestanden zijn compleet?
- Welke TODOs staan er open?
- Wat is het volgende te doen?
- Zijn er blokkerende issues?

## /test-proxy
Test de proxy snel:
```bash
# Start proxy
go run ./cmd/nexus start &

# Test health
curl http://localhost:3000/health

# Test met echte request (vereist ANTHROPIC_API_KEY)
curl -X POST http://localhost:3000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -d '{"model":"claude-haiku-4-5","max_tokens":100,"messages":[{"role":"user","content":"Say hello"}]}'
```
