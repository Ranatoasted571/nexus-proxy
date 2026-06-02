# NEXUS + agent harnesses (ECC, and others)

NEXUS pairs cleanly with **agent-harness toolkits** like
[ECC](https://github.com/affaan-m/ECC) — they live on **different layers**, so
you stack them instead of choosing:

```
┌─────────────────────────────────────────────┐
│  Claude Code / Cursor / Codex / …            │
│  ▶ ECC — skills, subagents, hooks, rules, MCP │   shapes WHAT the agent does
└───────────────────────┬───────────────────────┘
                        │ LLM calls (Anthropic / OpenAI API)
┌───────────────────────▼───────────────────────┐
│  ▶ NEXUS — routing · cache · privacy firewall  │   controls WHERE calls go,
│            cascade · adaptive · cost · bench    │   what they cost, what leaks
└───────────────────────┬───────────────────────┘
                        ▼ cheapest capable provider
```

ECC makes the agent **smarter**; NEXUS makes its traffic **cheaper, private and
measured**. ECC's high-volume agentic workload (many subagents/skills = many LLM
calls) is exactly where NEXUS's cascade, adaptive routing and caching save the
most.

## 1. It works today — no integration required

Install ECC into your harness as usual, then point the harness at NEXUS:

```bash
nexus start --cascade --redact --adaptive --inspect   # in one terminal
# then, in your project:
nexus code                                            # launches Claude Code through NEXUS
# …or set it manually:
export ANTHROPIC_BASE_URL=http://localhost:3000
export ANTHROPIC_API_KEY=nexus-local
```

Every LLM call ECC's skills/subagents make now flows through NEXUS — routed,
cached, cost-tracked, and with secrets/PII masked before they leave your machine.

## 2. Per-skill / per-subagent routing

ECC knows which skill or subagent is running; NEXUS honors two request headers so
a harness can steer routing per call:

| Header | Effect |
|--------|--------|
| `X-Nexus-Tier: premium` | pin this call to the premium tier (e.g. architecture, security) |
| `X-Nexus-Tier: free`    | pin to a free tier (e.g. lint, format, rename, summarize) |
| `X-Nexus-Provider: groq`| pin to one specific provider |

An ECC hook can set these headers per skill — e.g. route `architect` and
`security-audit` skills to `premium`, and `format`/`commit-msg` skills to `free`.
The same effect is available declaratively in `~/.nexus/config.toml`:

```toml
[[rules]]
when_prompt_contains = "security"
use_tier = "premium"

[[rules]]
when_model_contains = "haiku"
use_provider = "groq"
```

## 3. Let the agent see its own cost & savings (MCP)

NEXUS ships an MCP server. Add it to ECC's `mcp-configs/` (or any MCP client) so
the agent can answer "how much did I save today?" mid-session:

```json
{
  "mcpServers": {
    "nexus": { "command": "nexus", "args": ["mcp"] }
  }
}
```

Exposed tools: `nexus_stats`, `nexus_savings`, `nexus_recent`, `nexus_providers`,
`nexus_cost_breakdown`.

## 4. Defense in depth on privacy

ECC's security scanning + NEXUS's **privacy firewall** stack: NEXUS masks API
keys, secrets and PII *before any request leaves the machine* and restores them
in the response. Run `nexus report` for the proof:

```
🔒 Privacy   318 secrets/PII masked before leaving your machine · 0 leaked
```

## TL;DR

- ECC = the agent's brain (in the harness). NEXUS = the agent's nervous system
  (under the harness). Different layers, zero overlap.
- Combine with one env var today; refine with per-skill headers, an MCP config,
  and `config.toml` rules.
- NEXUS stays a single local Go binary — no merging, no new dependencies.
