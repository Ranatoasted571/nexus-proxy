# Draft proposal to affaan-m/ECC

> This is a **draft** for an issue (or PR) on https://github.com/affaan-m/ECC.
> It has not been submitted. Posting it is a manual decision.

---

**Title:** Optional cost + privacy layer for ECC workloads (NEXUS, local single binary)

**Body:**

Hi — ECC is a great harness-native operator system. I maintain
[NEXUS](https://github.com/lynuxis2026-pixel/nexus-proxy), an open-source single
Go binary that sits *under* a harness (Claude Code, Cursor, …) as a local proxy.
The two operate on different layers and compose cleanly:

- **ECC** shapes *what* the agent does (skills, subagents, hooks, rules, MCP).
- **NEXUS** controls *where* the resulting LLM calls go and what they cost/leak
  (intelligent + cheap-first-with-verification routing, adaptive learning,
  semantic cache, **privacy firewall that masks secrets/PII before egress**, and
  per-provider benchmarking on your own traffic).

Because ECC fans work out across many subagents/skills, the LLM-call volume is
exactly where routing + caching save the most — and where "secrets never leave
the machine" matters most.

**It already works** with zero integration (`ANTHROPIC_BASE_URL` → NEXUS). I'd
like to offer a small, opt-in integration so ECC users can get this for free:

1. **An MCP config** (`mcp-configs/nexus.json`) so the agent can read its own
   spend/savings mid-session:
   ```json
   { "mcpServers": { "nexus": { "command": "nexus", "args": ["mcp"] } } }
   ```
   Tools: `nexus_stats`, `nexus_savings`, `nexus_recent`, `nexus_cost_breakdown`.

2. **Per-skill routing hooks** — NEXUS honors `X-Nexus-Tier: premium|free` and
   `X-Nexus-Provider: <name>` headers, so an ECC hook can route e.g. `architect`
   / `security-audit` skills to premium and `format` / `commit-msg` skills to a
   free tier. Happy to contribute an example hook.

3. **A short docs section** ("Cut cost & keep data local with NEXUS") pointing at
   the stack-them guide.

No dependency or lock-in: NEXUS is a single local binary, MIT, no cloud, no token
markup. If this is welcome, I'm glad to open a focused PR adding just the
`mcp-configs/nexus.json` + a docs paragraph (nothing invasive). If you'd rather
not, no worries at all — wanted to flag the synergy.

Stack-them guide: <link to docs/agent-harnesses.md once published>

Thanks for ECC!
