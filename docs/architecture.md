# NEXUS Architecture

## Overview

```
┌─────────────────────────────────────────────────────────┐
│                     NEXUS Binary                         │
│                                                         │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │  Proxy Server│    │  Dashboard   │                   │
│  │  :3000       │    │  Server :2222│                   │
│  └──────┬───────┘    └──────┬───────┘                   │
│         │                   │                           │
│  ┌──────▼───────────────────▼───────┐                   │
│  │           Core Services          │                   │
│  │  ┌──────────┐  ┌──────────────┐  │                   │
│  │  │  Router  │  │  SSE Broker  │  │                   │
│  │  │ Classify │  │  Broadcast   │  │                   │
│  │  └──────────┘  └──────────────┘  │                   │
│  │  ┌──────────────────────────┐    │                   │
│  │  │      SQLite Storage      │    │                   │
│  │  └──────────────────────────┘    │                   │
│  └──────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│           LLM Providers                │
│  Anthropic  DeepSeek  Groq  Gemini     │
│  Ollama (local)                        │
└────────────────────────────────────────┘
```

## Request Flow

```
1. Claude Code sends POST /v1/messages to NEXUS
2. Handler reads and parses the request
3. Classifier determines task complexity (simple/standard/complex/critical)
4. Router selects best provider based on complexity + strategy + health
5. If provider is OpenAI-compatible: Transformer converts Anthropic → OpenAI format
6. Request forwarded to provider
7. Response (or stream) forwarded back to Claude Code
   - If OpenAI format: Transformer converts back to Anthropic format
8. Request logged to SQLite
9. SSE event pushed to all connected dashboard clients
```

## Key Design Decisions

### Single Binary
Everything — proxy, dashboard, embedded web UI — ships in one binary.
No Python runtime, no Docker, no Node.js required.

### Pure Go SQLite (modernc)
Uses `modernc.org/sqlite` instead of the CGO-based `mattn/go-sqlite3`.
This enables cross-compilation without a C compiler.

### SSE over WebSockets
Server-Sent Events are simpler (HTTP/1.1), browser-native, and work
through proxies and firewalls. For a one-directional stream (server → client),
SSE is the right tool.

### Anthropic-native API
NEXUS accepts the real Anthropic API format (not OpenAI-compatible).
This means Claude Code needs zero changes — it just works.
When routing to OpenAI-compatible providers, NEXUS transforms the format
internally and transparently.

## Port Assignment

| Port | Service |
|------|---------|
| 3000 | Proxy (Claude Code points here) |
| 2222 | Dashboard (humans open this) |
