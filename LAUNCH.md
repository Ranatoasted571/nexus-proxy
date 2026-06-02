# 🚀 NEXUS Launch Playbook

Copy-paste ready. Post in **your** voice — honest, problem-first, reply fast.

Repo: https://github.com/lynuxis2026-pixel/nexus-proxy

> **Positioning (read this first).** "Route Claude Code to cheaper models" is a
> commodity — a 34k-star repo and OpenRouter already own that headline. Do **not**
> lead with "save money." Lead with the quadrant nobody else occupies:
> **Private · Proven · Self-learning · Local.** Cost is the *benefit*, not the pitch.
>
> The one-liner that survives every comparison:
> *"OpenRouter routes to the cheapest provider. NotDiamond predicts from the crowd.
> NEXUS learns which provider wins **your** tasks, verifies the cheap answer before
> trusting it, redacts secrets before they leave your machine, and proves it by
> benchmarking every provider on **your own** traffic — one local binary, no markup."*

---

## 0. Pre-launch checklist

- [ ] Repo **public** ✅ · `v0.4.0` release with binaries ✅
- [ ] Hero **GIF** set in README ✅ (the single biggest conversion lever)
- [ ] **Social preview** image (Settings → Social preview, 1280×640 — export `docs/hero.svg`)
- [ ] Clean-machine `curl | sh` smoke test of the one-liner
- [ ] Have a real **`nexus report`** screenshot ready ("$ saved + N secrets masked · 0 leaked") and a **`nexus bench`** Report Card screenshot — these are the proof shots
- [ ] Be at your desk the first 2 hours to reply

**Timing:** HN is best **Tue–Thu, ~8–10am US Eastern**. Post one channel, seed the rest over the day.

---

## 1. Hacker News (Show HN)

**Title** (≤80 chars, no emoji):
```
Show HN: NEXUS – cut AI coding cost without your code leaving your machine
```

**URL:** `https://github.com/lynuxis2026-pixel/nexus-proxy`

**First comment** (post immediately):
```
Routing Claude Code to cheaper models is a solved/commodity thing now (a 34k-star
proxy and OpenRouter both do it). The parts that bugged me weren't solved, so I
built NEXUS around them — it's a single local Go binary that sits between your AI
coding tool and the providers:

1. Privacy. It masks API keys, secrets, private keys and PII *before* a request
   ever leaves your machine, and restores them in the response — so you can use a
   cheap third-party model on a codebase you're under NDA about. Cloud routers
   structurally can't offer this; the request leaves their way.

2. Proof, not vibes. `nexus bench` replays your *own* recent requests against every
   provider and prints cost × latency × agreement (how close each model's output is
   to your original), then recommends the cheapest one that stays close. You pick a
   model by measuring on your real traffic, not from a generic leaderboard.

3. It learns *your* routing. A cheap-first cascade tries the cheapest capable model,
   verifies the output (valid tool-call JSON / non-empty), and escalates only on
   failure — and it learns which provider wins which task type from those outcomes.
   Not the crowd's preferences; yours.

Plus the table stakes: speaks both the Anthropic and OpenAI APIs (Claude Code,
Cursor, aider, Cline — one env var), 30 providers + any OpenAI-compatible endpoint,
semantic cache, failover, budget caps, a live local dashboard, and a `nexus report`
that prints "$X saved · N secrets masked before leaving your machine · 0 leaked."

One Go binary, pure-Go SQLite, no cloud, no token markup, MIT. `nexus code` starts
it and launches Claude Code through it in one command.

Honest caveats: the bench "agreement" score is a cheap local word-overlap proxy for
quality, not an LLM judge — directionally useful, not gospel. Adaptive routing
learns within a run (in-memory). The privacy firewall uses high-confidence detectors
(key prefixes, JWTs, private keys, KEY=secret, emails) — it won't catch every secret
shape, so it's defense-in-depth, not a guarantee. Feedback very welcome.
```

**HN do's:** reply to everything, concede valid criticism, never argue, never ask for upvotes.

---

## 2. Reddit

### r/LocalLLaMA
**Title:**
```
I built a local single-binary proxy for Claude Code/Cursor that redacts your secrets before they hit a cloud model, benchmarks providers on YOUR traffic, and learns your routing — open source
```
**Body:**
```
NEXUS is a local proxy for AI coding tools. The cheap-routing part is table stakes;
what I actually wanted:

- Privacy firewall: secrets/API keys/PII are masked before any request leaves the
  machine and restored in the response. Use DeepSeek/Groq on NDA'd code safely.
- `nexus bench`: replays your real captured requests across every provider and
  ranks them by cost × latency × agreement on YOUR work — not a generic benchmark.
- Cheap-first cascade with verification + adaptive routing that learns which
  provider wins your task types.
- One Go binary, pure-Go SQLite, no Docker, no cloud, no token markup. 30 providers
  + any OpenAI-compatible endpoint. Works with Claude Code, Cursor, aider, Cline via
  one env var (speaks both Anthropic + OpenAI APIs).

MIT, one curl to install. Repo + binaries: <link>

Feedback wanted on the routing + the agreement metric.
```

### r/ClaudeAI
**Title:**
```
Use Claude Code, route the cheap requests to free models — but mask your secrets first and prove the cheap model is good enough (open-source, local)
```
**Body:**
```
Claude Code is my daily driver. NEXUS is a tiny local proxy (one env var:
ANTHROPIC_BASE_URL) that routes simple requests to free/cheap models and keeps
Claude for the hard stuff — but two things make it different from the other
"cheaper Claude Code" proxies:

1) it redacts secrets/PII before anything leaves your machine, and
2) `nexus bench` proves which cheap model is actually good enough on YOUR real
   prompts before you trust it.

Live local dashboard + a `nexus report` that shows what you saved AND that nothing
leaked. Single Go binary, MIT. <link>
```

### r/selfhosted
**Title:**
```
NEXUS – self-hosted, single-binary AI-coding proxy: privacy firewall + cost routing + benchmark-on-your-traffic, no cloud
```

**Reddit etiquette:** be present in comments, follow each sub's self-promo rules.

---

## 3. X / Twitter (thread)

```
1/ "Route Claude Code to a cheaper model" is a solved problem now.

The unsolved parts: doing it without leaking your code, and knowing the cheap model
is actually good enough.

So I built NEXUS — a local, open-source binary that does both 👇
github.com/lynuxis2026-pixel/nexus-proxy

2/ 🔒 Privacy firewall. It masks API keys, secrets and PII *before* a request ever
leaves your machine, then restores them in the response. You can point a cheap
third-party model at NDA'd code. A cloud router can't make that promise.

3/ 🧪 Proof, not vibes. `nexus bench` replays YOUR real requests across every
provider and ranks them by cost × latency × agreement on your own work — then tells
you the cheapest one that stays close. Stop guessing which model is "good enough."

4/ 🧠 It learns your routing. Cheap-first: try the cheapest model, verify the
output, escalate only on failure — and learn which provider wins which of YOUR task
types over time. Not the crowd's preference. Yours.

5/ 📦 One Go binary. No Python, no Docker, no cloud, no token markup. `nexus code`
starts it and launches Claude Code through it in one command. Works with Cursor,
aider, Cline too (speaks both Anthropic + OpenAI APIs).

6/ And `nexus report` gives you one shareable line:
"$X saved · N secrets masked before leaving your machine · 0 leaked."

Free, MIT, stars + feedback make my week 🙏
github.com/lynuxis2026-pixel/nexus-proxy
```

Attach the **dashboard GIF** to tweet 1 and a **`nexus report`/`bench` screenshot** to tweet 3 or 6.

---

## 4. Product Hunt

**Name:** NEXUS
**Tagline (≤60 chars):**
```
Private, proven cost-routing for Claude Code — local
```
**Description:**
```
NEXUS is an open-source, single-binary local proxy for AI coding tools. It routes
Claude Code, Cursor or aider to the cheapest capable model — but unlike cloud
routers it masks your secrets/PII before anything leaves your machine, proves which
cheap model is good enough by benchmarking every provider on YOUR real traffic
(nexus bench), and learns your best routing from real outcomes. Live local
dashboard, semantic cache, budget caps, MCP server. No Python, no Docker, no cloud,
no token markup. MIT.
```
**First comment:** the problem-first HN story, shortened.

---

## 5. Longer form (LinkedIn / Dev.to / blog)

Angle: *"Cheaper Claude Code is a commodity. I built the layer above it."* — the
privacy problem (NDA'd code + cloud models), benchmark-on-your-own-traffic, and the
local-first/single-binary design (a nod to why "no cloud, no supply-chain surface"
matters in 2026). Link the repo; cross-post to Dev.to + Hashnode.

---

## 6. After you post

- Reply to **every** comment in the first hours — engagement drives ranking.
- Turn the best questions into README FAQ entries.
- The two screenshots that convert: the **dashboard GIF** and a real **`nexus report`** ("0 leaked").
- A week later: a "what I learned launching" follow-up with real numbers.

Go get seen. 🌍
