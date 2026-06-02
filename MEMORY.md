# NEXUS — Project Memory

## Status
- **Fase**: Sprint 4 compleet — MVP klaar; build/vet/test groen, cross-compile werkt
- **Laatste update**: Juni 2026
- **Sprint**: 4 (polish & release) ✓ — rest: handmatige release-stappen (zie onder)
- **Providers**: 24 ingebouwd + generieke "custom" provider (élke OpenAI-compatibele endpoint).
  Built-in: anthropic, openai, xai, deepseek, mistral, cohere, together, fireworks, openrouter,
  deepinfra, perplexity, novita, hyperbolic, nebius, moonshot, zhipu, ai21, lambda, groq, gemini,
  cerebras, sambanova, nvidia, ollama
- **Toolchain**: Go 1.26.3 (winget) + Node 20 (nodejs op PATH); go.mod blijft op `go 1.22`

## Vóór publicatie (handmatig)
- [ ] Module-pad `github.com/yourusername/nexus` → echte repo (go.mod + alle imports + `yourusername` in README/.goreleaser.yml/install-scripts)
- [ ] `git init` + push naar GitHub; tag `v0.1.0` → CI draait GoReleaser
- [ ] `HOMEBREW_TAP_TOKEN` secret zetten voor de Homebrew-tap
- [ ] Dashboard-GIF opnemen → `docs/dashboard.gif`, hero in README uncommenten
- [ ] (Optioneel) domein `get.nexus.sh` of gebruik de raw.githubusercontent install-URL

---

## Beslissingen Log

### [2026-06] Bedrock/Vertex kosteloos testbaar (base_url override)
- Probleem: Bedrock/Vertex hadden hun endpoint hardgecodeerd → niet te testen zonder
  betaald cloud-account. Opgelost: optionele `base_url` override op beide (default blijft
  de echte AWS/GCP-URL). Maakt routeren via gateway/VPC-endpoint mogelijk ÉN testen gratis.
- Nieuwe integratietests (`handler_test.go`) tegen een in-proces server valideren de
  VOLLEDIGE flow: Bedrock (SigV4 Authorization + X-Amz-Date + /invoke + body→bedrock-2023-05-31,
  model gestript), Vertex (Bearer + :rawPredict + body→vertex-2023-10-16, stream gestript +
  gesynthetiseerde SSE). Beide groen, $0 cloudkosten.
- Caveat uit vorige entry is hiermee opgelost: hele request/auth/transform/response-pad is nu
  getest. (Optioneel echte smoke-test: GCP $300 gratis trial voor Vertex; Bedrock ~centen.)

### [2026-06] Gepubliceerd op GitHub + release v0.1.0
- Repo: github.com/lynuxis2026-pixel/nexus-proxy (PRIVATE, branch main), gepusht via gh.
- Module-pad hernoemd `github.com/yourusername/nexus` → `github.com/lynuxis2026-pixel/nexus-proxy`
  (alle imports + go.mod + README/scripts). `.gitignore` gefixt: `dist/` → `/dist/` zodat de
  embedded UI (`internal/dashboard/dist`) wél mee-commit.
- `.goreleaser.yml`: owner/repo gezet, Homebrew-blok uitgezet (geen tap). CI `release`-job kreeg
  `permissions: contents: write`.
- Tag `v0.1.0` → GitHub Actions: Test (1m24s, `go test -race` groen) + GoReleaser (3m21s) →
  release met 5 platform-archives + checksums.txt.
- **Nog te doen voor publiek/viral**: repo public maken (`gh repo edit --visibility public`),
  dashboard-GIF in README-hero, eventueel Homebrew-tap, en aankondigen (HN/Reddit).

### [2026-06] Mock weg + in-suite integratietests
- Externe mock-server (Temp\nexusmock) + test-home + stale bin/ verwijderd. Broncode bevat
  GEEN mock/TODO/placeholder/stub (alleen node_modules/compiled bundle).
- Live-verificaties vervangen door permanente `httptest`-integratietests (in-proces, geen los
  binary): `internal/proxy/handler_test.go` test de échte HandleMessages-flow:
  OpenAI non-stream, live streaming, 503→200 failover, budget→free, Azure api-key auth, 400 bad-JSON.
  `internal/dashboard/server_test.go` test stats/requests/timeseries/breakdown uit een temp-DB.
- 46 testfuncties, alles groen; `go vet` schoon; 5 platforms cross-compileren (v0.1.0).
- Eerlijk: Bedrock/Vertex round-trip naar de échte cloud is niet te testen zonder betaald
  account; hun request-opbouw is unit-tested en SigV4 is geverifieerd tegen AWS' get-vanilla
  vector. Azure is wél end-to-end getest (endpoint configureerbaar).

### [2026-06] Vijf grote features: budget, health, streaming, enterprise, charts
- **Daily budget cap**: `--budget` / `routing.daily_budget_usd`. In-memory `budgetTracker`
  (init uit DB today, dag-rollover); over budget → chain gefilterd op free/local (`freeLocalOnly`).
- **Achtergrond-health-checks**: `Router` kreeg `sync.RWMutex` + `SetHealthy`; handler draait
  een `healthLoop` (init-pass + 30s ticker, concurrent), stopt via `Handler.Close` op shutdown.
- **Echte OpenAI-streaming**: client-stream → upstream `stream:true` + `stream_options.include_usage`;
  `relayOpenAIStream` parst OpenAI-SSE en zet live om naar Anthropic-events (text + tool_calls).
- **Enterprise providers** (config-driven types):
  - `azure` — OpenAI-formaat, `api-key` header, `base_url`+`api_version`.
  - `vertex` — Anthropic-formaat, bearer GCP-token, URL met project/region, body→anthropic_version vertex.
  - `bedrock` — Anthropic-formaat, **SigV4** (env AWS-creds), body→anthropic_version bedrock.
  - Nieuwe interfaces `Authorizer` (custom auth) + `AnthropicNative` (custom URL/body). SigV4-signer
    in `sigv4.go`, geverifieerd tegen AWS "get-vanilla" testvector. Bedrock/Vertex = buffered relay
    (`relayAnthropicBuffered`) met SSE-synthese bij client-stream.
- **Dashboard-grafieken**: storage `GetHourlySeries` + `GetComplexityBreakdown`; endpoints
  `/api/timeseries` + `/api/breakdown`; Svelte `App.svelte` met chart.js (kosten-lijn + provider-doughnut),
  live via stores. Bundle 222 kB, embedded.
- **Config**: `Provider` kreeg type/model_map/pricing/region/project/api_version + `ResolveKey` (env:VAR).
- **Geverifieerd**: failover bad(503)→good(200) + env-key live; SigV4-vector; OpenAI-stream-conversie;
  azure/vertex/bedrock construct + URL/body/auth (unit); chart-endpoints live (timeseries/breakdown/stats).
  Bedrock/Vertex endpoints zelf niet tegen echte cloud getest → markeren als "valideren met echte creds".

### [2026-06] Provider-catalogus → 24 + custom + flexibiliteit
- **+8 ingebouwde providers**: novita, hyperbolic, nebius, nvidia (free), moonshot (Kimi),
  zhipu (GLM), ai21 (Jamba), lambda. Totaal 24 built-in.
- **Generieke "custom" provider** (`internal/providers/generic.go`): `Spec` + `New(spec)`.
  Type `openai-compatible` met `base_url` → élke OpenAI-compatibele endpoint zonder code
  (vLLM, LM Studio, LiteLLM, Azure-style, …). CLI: `nexus add X <key> --type openai-compatible --base-url <url>`.
- **Config-overschrijfbare model-mapping** (`model_map`) via een `overridden` wrapper rond
  built-ins, en native in Generic. Ook pricing- en tier-override per provider.
- **Env-var keys**: `api_key = "env:VAR"` → `config.ResolveKey` lost op bij gebruik.
- **Automatische failover**: handler schuift door naar de volgende provider bij 429/500/502/
  503/504 (`isRetryableStatus`). 4xx (bv. 401) wordt wél gerelayd.
- **`nexus models`** command toont de Claude→provider mapping per provider.
- **Handler** bouwt providers nu via `providers.New(spec)` (env-resolved key) i.p.v. FromConfig.
- Live geverifieerd: failover bad(503)→good(200) + env-key (`Bearer resolved-secret-123`
  kwam aan bij de custom provider) in één test. Build/vet/test groen.

### [2026-06] Provider-catalogus → 16
- 11 OpenAI-compatibele providers toegevoegd: openai, mistral, together, openrouter,
  cohere, xai, fireworks, perplexity, deepinfra, cerebras, sambanova.
- Elk bestand volgt het bestaande patroon (Name/BaseURL/Tier/MapModel/Pricing/
  ChatCompletionsURL/HealthCheck). Gedeelde `bearerHealthCheck` helper in provider.go
  (GET /models met Bearer); perplexity gebruikt een lichte reachability-check.
- Geregistreerd in `FromConfig`, `DefaultTier`, `DefaultModels`; test dekt alle 16.
- Tier-indeling: premium=anthropic/openai/xai; free=groq/gemini/cerebras/sambanova;
  local=ollama; rest standard. Router/classifier ongewijzigd — pakt ze automatisch op.
- Live geverifieerd: `nexus add` werkt voor alle 11, config laadt 12, `nexus status`
  bereikt elk echt endpoint (401/400 met fake keys = correcte URL + bereikbaar).

### [2026-06] Sprint 4 — polish & release
- **Tests uitgebreid**: `config` (load/save/upsert), `storage` (log/recent/stats/breakdown),
  `providers` (FromConfig/URLs/cost/tier/mapmodel), `proxy` (transformer + SSE + usage).
  Alles groen; coverage core-packages: storage 71%, config 68%, router 63%.
- **Cross-compile geverifieerd**: linux/darwin/windows × amd64/arm64 (CGO_ENABLED=0),
  5 binaries 12–13 MB elk — single binary mét embedded dashboard, geen cgo. ✓
- **Release-artefacten**: `LICENSE` (MIT) + `CONTRIBUTING.md` toegevoegd (werden door
  README/.goreleaser.yml verwacht). `install.sh` (Linux/macOS, OS+arch-detectie, GitHub
  releases) en `install.ps1` (Windows, user-PATH) — beide syntax-gecheckt.
- **README**: Windows-install + build-from-source secties; hero-GIF placeholder.
- **Bewust niet gedaan**: GitHub push, release-tag en HN-post (handmatig, zie checklist boven).
  Module-pad blijft `yourusername` tot de echte repo bekend is.

### [2026-06] Sprint 3 — live dashboard werkend
- **Gedeelde resources**: `main` opent nu de SQLite-DB én de SSE-broker en geeft beide
  door aan zowel de proxy (`proxy.New(cfg, db, broker)`) als de dashboard-server
  (`dashboard.NewServer(port, broker, db)`). Handler bezit de DB niet meer; main sluit 'm.
- **Events**: handler pusht na elke request een `request`- en een `stats`-event naar de
  broker (via een `proxy.EventPublisher` interface — geen import-cycle met dashboard).
  `storage.LogRequest` geeft nu het rij-id terug voor de event-payload.
- **Dashboard-API**: `/api/stats`, `/api/requests`, `/api/providers` lezen uit storage/config
  (geen placeholders meer).
- **go:embed**: `internal/dashboard/embed.go` met `//go:embed all:dist`. De UI wordt uit
  `internal/dashboard/dist/` geserveerd via `http.FileServer`. Makefile `embed` kopieert
  `web/dist/*` hierheen; deze map is de gecommitte embed-payload (zodat `go build` werkt
  zonder Node). Plain `go build` werkt altijd; `make build` regenereert de UI.
- **Svelte-build fixes**: `"type":"module"` in package.json (vite-plugin-svelte is ESM-only),
  `vitePreprocess()` voor `lang="ts"`, en de Vite-entry verplaatst van `web/public/index.html`
  naar `web/index.html` (Vite-conventie). Store kreeg `fetchInitial()` + `providers`-store.
- **Geverifieerd (live, single binary)**: `/events` pushte real-time een `request`-event
  (provider=ollama, status=200) + `stats`-event toen ik mid-stream een request afvuurde;
  embedded Svelte-app + assets geserveerd met juiste MIME-types; API's lezen uit SQLite.

### [2026-06] Sprint 2 — router + storage werkend
- **Nieuw `internal/config`**: TOML laden/opslaan/upsert van `~/.nexus/config.toml`, met
  zero-config defaults. `BurntSushi/toml` weer in go.mod.
- **Providers-bridge**: `ChatCompletionsURL()` toegevoegd aan de interface + alle 5 impls;
  `FromConfig`, `IsOpenAICompatible`, `DefaultTier`, `DefaultModels` helpers.
- **Router**: `RouteChain()` geeft een geordende fallback-keten (healthy providers, per
  tier-voorkeur); `Route()` = head daarvan. Strategieën: auto/cheapest/fastest/manual.
- **Handler wiring**: classify → RouteChain → forward. Native Anthropic = passthrough
  (sync + stream). OpenAI-providers = upstream NIET-streamend, daarna OpenAI→Anthropic
  conversie; als de client streamt synthetiseren we de Anthropic SSE-events (`openai.go`).
  Fallback naar volgende provider ALLEEN bij transport-fout (HTTP-fout wordt gerelayd).
- **Zero-config behouden**: geen providers → rechtstreeks naar Anthropic (Sprint 1-gedrag).
- **Storage fix**: `created_at` als UTC-tekst `2006-01-02 15:04:05` opslaan zodat SQLite
  `date()` werkt (anders telde `cost` 0 requests). modernc scant het terug naar time.Time.
- **CLI**: `add` (persisteert provider), `status` (live health), `logs`, `cost` volledig.
- **Tests**: `internal/router` (routing-by-complexity, fallback, cheapest, classifier) +
  `internal/proxy` (OpenAI→Anthropic transform, SSE-synthese, usage-parsing). Alles groen.
- **Streaming-keuze**: OpenAI-providers gaan upstream non-streaming; échte token-streaming
  voor die providers = latere verbetering. Anthropic streamt wél token-voor-token.
- Live geverifieerd met mock-OpenAI provider (geïsoleerde HOME): simple→ollama 200 (met
  format-conversie), complex→anthropic, SSE-synthese valide, logs+cost correct.

### [2026-06] Sprint 1 — core proxy werkend
- `cmd/nexus/main.go`: `runStart` start nu echt de proxy + (optioneel) dashboard + SSE-broker,
  met zerolog console-logging en graceful shutdown via signal-context (Ctrl+C).
- `internal/proxy/handler.go`: veilige header-forwarding (hop-by-hop + Content-Length/Encoding
  worden gedropt), `extractAPIKey` helper, en fallback naar server-side `ANTHROPIC_API_KEY`
  als de client een lege of `nexus-local` key stuurt.
- `internal/proxy/stream.go` (nieuw): streaming losgetrokken uit de handler, flush per chunk,
  stopt netjes bij client-disconnect.
- `internal/dashboard/server.go`: `Shutdown()` toegevoegd.
- `go mod tidy` heeft ongebruikte deps (viper, BurntSushi/toml, spinner, progressbar, gjson,
  sjson) uit go.mod gehaald — komen terug zodra Sprint 2/3 ze importeert.
- Geverifieerd: `go build ./...` + `go vet ./...` schoon; binary 9.5MB; live test relayde
  een echte Anthropic 401 (provider=anthropic, ~319ms) → forwarding werkt end-to-end.

### [2026-06] Initieel ontwerp
- Gekozen voor Go single binary als core onderscheidende factor
- LiteLLM als inspiratie maar doet te veel/te complex voor developers
- Bestaande claude-code-proxies hebben geen dashboard of zijn te simpel
- Dashboard design: donker, data-dense, mooi genoeg om te screenshotten

### [2026-06] Stack keuzes
- **SQLite modernc** gekozen boven bolt/badger: SQL queries voor stats zijn veel makkelijker
- **SSE** boven WebSockets: browser native, minder overhead, werkt door proxies
- **Cobra** voor CLI: de standaard in Go CLI tools, goed bekende API
- **Svelte** boven React: kleinere bundle (belangrijk voor embedded binary)
- **TOML** boven YAML: minder footguns, leesbaarder voor non-devs

---

## Open vragen

- [ ] Wil je een hosted versie (SaaS) naast de self-hosted binary?
- [ ] Budget alerts via email/webhook of alleen in dashboard?
- [ ] Team support (meerdere gebruikers, gedeelde proxy)?
- [ ] Plugin systeem voor custom providers?
- [ ] MCP server integratie (NEXUS als MCP server)?

---

## Competitor research

| Project | Stars | Gap |
|---|---|---|
| 1rgs/claude-code-proxy | ~2k | Geen UI, alleen model mapping |
| seifghazi/claude-code-proxy | ~500 | Monitoring only, geen routing |
| litellm | ~18k | Te complex, lelijke UI, Python |
| meridian | ~300 | Alleen Claude Max → andere tools |
| ujisati/claude-code-provider-proxy | ~200 | Format conversie, geen intelligence |

**NEXUS gap**: enige die combineert:
1. Single binary (geen deps)
2. Intelligente router (auto complexity)
3. Mooie live dashboard
4. Claude Code-specifiek gebouwd

---

## GitHub viral strategie

1. **README** — animated GIF van dashboard als hero
2. **One-liner install** — `curl -fsSL https://get.nexus.sh | sh`
3. **Hacker News post** — "Show HN: I built a smart proxy for Claude Code that routes to free models"
4. **Reddit** — r/ClaudeAI, r/LocalLLaMA, r/programming
5. **Twitter/X** — demo video van live dashboard
6. **Product Hunt** launch

---

## Providers prijzen (juni 2026)

| Provider | Model | Input/1M | Output/1M |
|---|---|---|---|
| Anthropic | claude-opus-4-5 | $15.00 | $75.00 |
| Anthropic | claude-sonnet-4-6 | $3.00 | $15.00 |
| Anthropic | claude-haiku-4-5 | $0.25 | $1.25 |
| DeepSeek | deepseek-chat | $0.27 | $1.10 |
| Groq | llama-3.3-70b | FREE | FREE |
| Gemini | gemini-2.0-flash | FREE | FREE |
| Ollama | any | $0 | $0 |

---

## Notities van eigenaar
- Bouwen = Claude Code met CLAUDE.md als context
- Begin altijd met Sprint 1 (core proxy)
- Test na elke sprint met echte Claude Code
- Dashboard pas bouwen als proxy 100% stabiel is
