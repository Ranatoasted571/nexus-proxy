# Prepared PR to affaan-m/ECC (NOT submitted)

Everything is staged for a one-command submit. The change is pushed to a branch
on the fork; only `gh pr create` remains.

- **Fork:** https://github.com/lynuxis2026-pixel/ECC
- **Branch:** `add-nexus-mcp-server` (1 file changed, +5 lines)
- **Change:** adds a `"nexus"` entry to `mcp-configs/mcp-servers.json`
- **Related issue:** affaan-m/ECC#2123
- **Create-PR URL:** https://github.com/lynuxis2026-pixel/ECC/pull/new/add-nexus-mcp-server

## To submit (run only on your go)

```bash
gh pr create -R affaan-m/ECC \
  --head lynuxis2026-pixel:add-nexus-mcp-server \
  --title "Add NEXUS to mcp-configs/mcp-servers.json (local cost/privacy proxy)" \
  --body-file docs/ECC-pr-body.md
```

## PR title

Add NEXUS to mcp-configs/mcp-servers.json (local cost/privacy proxy)

## PR body

Follow-up to #2123 — the small, focused change proposed there: one new entry in
`mcp-configs/mcp-servers.json` for **NEXUS**
(https://github.com/lynuxis2026-pixel/nexus-proxy), a local single-binary
cost/privacy proxy that runs under the harness.

Adding it lets an ECC agent introspect its own usage mid-session via these MCP
tools:

- `nexus_stats` — requests, cost, tokens, cache savings
- `nexus_savings` — saved $ and % vs all-Claude
- `nexus_recent` — recent requests (provider / model / cost / latency)
- `nexus_providers` — configured providers + tiers
- `nexus_cost_breakdown` — cost grouped by provider

**Opt-in, zero dependency:** it's a single JSON entry; nothing runs unless a user
has `nexus` installed and enables it. NEXUS is MIT, one Go binary, no cloud, no
token markup. Happy to tweak the wording or add a docs line if you'd prefer.

Stack-them guide: https://github.com/lynuxis2026-pixel/nexus-proxy/blob/main/docs/agent-harnesses.md
