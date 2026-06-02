# 📱 Build NEXUS from your phone

You can keep developing NEXUS with Claude Code from anywhere — including your
phone — using **GitHub Codespaces**, a cloud dev environment that runs in a
browser. This repo ships a `.devcontainer/` so a Codespace boots with Go 1.22,
Node 20, all dependencies installed, and **Claude Code pre-installed**.

## One-time setup

1. On your phone, open the repo: **github.com/lynuxis2026-pixel/nexus-proxy**
   (the GitHub mobile app or any mobile browser works).
2. Tap **`< > Code` → Codespaces → Create codespace on main**.
3. Wait ~1–2 min for it to build (it runs `go mod download`, installs the web
   deps, and `npm i -g @anthropic-ai/claude-code`).

## Each session

Open your codespace (it resumes), then in the terminal:

```bash
# authenticate Claude Code once per codespace
claude            # follow the login prompt

# keep building — Claude Code has full repo context via CLAUDE.md + MEMORY.md
claude "add a Mistral-large model option and a test for it"

# verify your change
go test ./...
go build ./cmd/nexus && ./bin/nexus version
```

To see the app live, run `./bin/nexus start` — Codespaces auto-forwards ports
**3000** (proxy) and **2222** (dashboard); tap the forwarded **2222** URL to open
the dashboard right on your phone.

## Tips

- **Commit from the phone:** `git add -A && git commit -m "..." && git push` —
  the CI builds + tests on every push, and a `v*` tag cuts a release.
- The **GitHub mobile app** lets you review the diff, merge PRs, and watch CI runs.
- Prefer a native terminal? Codespaces also works from the **VS Code** or
  **GitHub** mobile-friendly web editors.
- If Claude Code on web/mobile is available to your account, point it at this
  repo — the devcontainer gives it the same ready-to-build environment.

That's it: full NEXUS development, from your pocket. 🚀
