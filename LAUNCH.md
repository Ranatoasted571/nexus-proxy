# 🚀 NEXUS Launch Playbook

Everything you need to reach the world. Copy-paste ready. Post in **your** voice —
keep it honest, answer every comment fast, and lead with the problem, not the hype.

Repo: https://github.com/lynuxis2026-pixel/nexus-proxy

---

## 0. Pre-launch checklist (do this first)

- [ ] Repo is **public** ✅ (done)
- [ ] `v0.2.0` release with binaries ✅ (done)
- [ ] Record a **dashboard GIF** (`http://localhost:2222`) → `docs/dashboard.gif`, set it as the README hero. *This is the single biggest conversion lever.*
- [ ] Upload a **social preview image** (Repo → Settings → Social preview, 1280×640 PNG — export `docs/hero.svg`). Makes shared links look pro on X/Slack/Discord.
- [ ] Add a repo **description + website** (Settings) — already topiced.
- [ ] Do a real `curl | sh` install on a clean machine to confirm the one-liner works.
- [ ] Pin a "Show HN is live → [link]" note in the README the morning of launch.

**Timing:** Hacker News is best **Tue–Thu, ~8–10am US Eastern**. Post to one channel,
then seed the others over the day. Be at your desk for the first 2 hours to reply.

---

## 1. Hacker News (Show HN)

**Title** (≤80 chars, no emoji on HN):
```
Show HN: NEXUS – route Claude Code, Cursor and aider to the cheapest LLM
```

**URL:** `https://github.com/lynuxis2026-pixel/nexus-proxy`

**First comment** (post immediately after submitting):
```
I kept watching my Claude/OpenAI credits vanish on tasks that a free model could
have handled, so I built NEXUS: a single-binary local proxy that sits between your
AI coding tool and the providers, classifies each request, and routes it to the
cheapest model that can actually handle it. Simple stuff stays free (Groq, Gemini,
Cerebras…), real work goes to DeepSeek/Mistral/etc., and architecture/security
prompts still go to Claude or GPT.

Two things I wanted that didn't exist together:
- It speaks BOTH the Anthropic API (/v1/messages) and the OpenAI API
  (/v1/chat/completions), so the same proxy works with Claude Code, Cursor, aider,
  Continue, Cline — one env var.
- A live dashboard that shows every request, which model handled it, latency and
  cost in real time, plus a running "vs. Claude" savings number.

It's one Go binary (no Python, no Docker), pure-Go SQLite, 24 providers built in
plus any OpenAI-compatible endpoint, a normalized response cache, automatic
failover on rate-limits, and an optional daily budget cap. MIT licensed.

Honest caveats: routing heuristics are simple (token count + keywords + the model
Claude Code picked) and I'd love smarter classification ideas. Bedrock/Vertex are
implemented but I've only tested them against the documented APIs + a SigV4 vector,
not a paid account. Feedback very welcome.
```

**HN do's:** reply to everything, concede valid criticism, never argue. Don't ask for upvotes anywhere (instant flag).

---

## 2. Reddit

### r/LocalLLaMA  (most likely to love this)
**Title:**
```
I built a single-binary proxy that routes Claude Code / Cursor / aider to the cheapest model (free tiers first), with a live cost dashboard — open source
```
**Body:**
```
NEXUS is a local proxy for AI coding tools. It classifies each request and sends
the simple ones to free providers (Groq, Gemini, Cerebras, SambaNova, NVIDIA NIM,
or your local Ollama) and only escalates the hard stuff to paid models.

- One Go binary, no Docker, pure-Go SQLite.
- Speaks the Anthropic AND OpenAI APIs, so it works with Claude Code, Cursor,
  aider, Continue, Cline, or any OpenAI SDK app — one env var.
- 24 providers built in + any OpenAI-compatible endpoint (vLLM, LM Studio, LiteLLM…).
- Live dashboard with per-request cost, smart cache, auto-failover, budget cap.

MIT, install is one curl command. Repo + binaries: <link>

Would love feedback on the routing heuristics and which providers to add next.
```

### r/ClaudeAI
**Title:**
```
NEXUS: keep using Claude Code, but auto-route the cheap requests to free models (open-source proxy + live cost dashboard)
```
**Body:** (lead with the Claude Code angle)
```
Claude Code is my daily driver but the API bill adds up. NEXUS is a tiny local
proxy — point Claude Code at it with one env var (ANTHROPIC_BASE_URL) and it routes
simple requests to free/cheap models while keeping Claude for architecture, debugging
and anything you mark urgent. There's a live dashboard showing exactly what went
where and what you saved. Single binary, MIT. <link>
```

### r/selfhosted
**Title:**
```
NEXUS – self-hosted LLM gateway for coding tools: one binary, 24 providers, live dashboard, cost cap
```

**Reddit etiquette:** be present in comments, no cross-post spam, follow each sub's self-promo rules (some require a flair or a ratio of contributions).

---

## 3. X / Twitter (thread)

```
1/ I got tired of watching Claude Code credits disappear on trivial tasks.

So I built NEXUS: a single-binary proxy that routes Claude Code, Cursor & aider to
the cheapest model that can handle each request — with a live cost dashboard.

Open source, MIT 👇 github.com/lynuxis2026-pixel/nexus-proxy

2/ It classifies every request: a quick question goes to a free model (Groq,
Gemini, Cerebras…), a refactor goes to DeepSeek, and real architecture work still
goes to Claude or GPT. You keep the quality where it matters.

3/ One proxy, every tool. It speaks BOTH the Anthropic and OpenAI APIs, so Claude
Code, Cursor, aider, Continue and Cline all work through it with a single env var.

4/ 24 providers built in + any OpenAI-compatible endpoint. Smart cache (identical
requests are instant & free), auto-failover when a free tier rate-limits, and a
daily budget cap.

5/ One Go binary. No Python, no Docker. `curl | sh` and you're running, with a
dashboard at localhost:2222 that you'll actually want to screenshot.

6/ It's free and MIT licensed. Stars + feedback make my week:
github.com/lynuxis2026-pixel/nexus-proxy
```

Attach the dashboard GIF/screenshot to tweet 1 or 5 — visual tweets travel far.
The dashboard's **Share** button auto-fills a savings tweet for users too.

---

## 4. Product Hunt

**Name:** NEXUS
**Tagline (≤60 chars):**
```
Route every AI coding tool to the cheapest capable LLM
```
**Description:**
```
NEXUS is an open-source, single-binary proxy + live dashboard for AI coding tools.
Point Claude Code, Cursor, aider or any OpenAI/Anthropic app at it and it routes
each request to the cheapest model that can handle it — free tiers first, Claude
for the hard stuff. 24 providers built in, smart cache, auto-failover, budget cap,
and a real-time cost dashboard. No Python, no Docker. MIT.
```
**First comment:** same problem-first story as the HN comment, shortened.

---

## 5. LinkedIn / Dev.to / blog (optional, longer form)

Angle: *"Why I built an LLM router for coding agents"* — the cost problem, the
classify-then-route idea, the single-binary/dashboard design choices, and what you
learned. Link the repo. Cross-post to Dev.to and Hashnode for SEO.

---

## 6. After you post

- Reply to **every** comment in the first few hours — engagement drives ranking.
- Turn the best questions into README FAQ entries.
- Watch GitHub stars/issues; thank early contributors; label `good first issue`.
- A week later: post a "what I learned launching" follow-up with the numbers.

Good luck — go get seen. 🌍
