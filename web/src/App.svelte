<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Chart } from 'chart.js/auto'
  import {
    connectSSE, disconnectSSE, fetchInitial, fetchTimeseries, fetchSavings,
    connected, stats, recentRequests, providers, providerBreakdown, timeseries, savings,
  } from './stores/requests'

  const REPO = 'https://github.com/lynuxis2026-pixel/nexus-proxy'
  $: shareText = `I've saved $${$savings.saved_usd.toFixed(2)} on Claude Code with NEXUS 🚀 (${Math.round($savings.percent_saved)}% cheaper than Claude) — open-source smart LLM proxy`
  $: shareUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(shareText)}&url=${encodeURIComponent(REPO)}`

  const PALETTE = ['#7c3aed', '#06b6d4', '#10b981', '#f59e0b', '#ef4444', '#3b82f6', '#ec4899', '#84cc16', '#a855f7', '#14b8a6']

  // ── Request inspector + replay ──
  let inspecting: any = null
  let inspectLoading = false
  let replayProvider = ''
  let replayResult: any = null
  let replayLoading = false

  function pretty(s: string): string {
    try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
  }
  async function openInspector(id: number) {
    inspecting = null; replayResult = null; inspectLoading = true
    try {
      const r = await fetch(`/api/requests/${id}`)
      inspecting = await r.json()
    } catch { inspecting = { error: 'failed to load' } }
    inspectLoading = false
  }
  function closeInspector() { inspecting = null; replayResult = null }
  async function runReplay() {
    if (!inspecting?.id) return
    replayLoading = true; replayResult = null
    try {
      const r = await fetch('/api/replay', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: inspecting.id, provider: replayProvider }),
      })
      replayResult = await r.json()
    } catch (e) { replayResult = { error: String(e) } }
    replayLoading = false
  }

  // ── Team leaderboard ──
  let leaderboard: any[] = []
  async function loadLeaderboard() {
    try {
      const r = await fetch('/api/leaderboard')
      const j = await r.json()
      leaderboard = j.leaderboard || []
    } catch { /* ignore */ }
  }
  $: namedBoard = leaderboard.filter((e) => e.user && e.user !== 'unattributed')

  let costCanvas: HTMLCanvasElement
  let provCanvas: HTMLCanvasElement
  let costChart: Chart | undefined
  let provChart: Chart | undefined
  let poll: ReturnType<typeof setInterval>

  const gridColor = '#1a2035'
  const tickColor = '#64748b'

  onMount(() => {
    fetchInitial()
    connectSSE()

    costChart = new Chart(costCanvas, {
      type: 'line',
      data: { labels: [], datasets: [{
        label: 'Cost (USD)', data: [], borderColor: '#7c3aed',
        backgroundColor: 'rgba(124,58,237,0.15)', fill: true, tension: 0.35, pointRadius: 2,
      }] },
      options: {
        responsive: true, maintainAspectRatio: false,
        plugins: { legend: { labels: { color: tickColor, font: { size: 10 } } } },
        scales: {
          x: { ticks: { color: tickColor, maxTicksLimit: 8, font: { size: 9 } }, grid: { color: gridColor } },
          y: { ticks: { color: tickColor, font: { size: 9 } }, grid: { color: gridColor }, beginAtZero: true },
        },
      },
    })

    provChart = new Chart(provCanvas, {
      type: 'doughnut',
      data: { labels: [], datasets: [{ data: [], backgroundColor: PALETTE, borderColor: '#0a0e1a', borderWidth: 2 }] },
      options: {
        responsive: true, maintainAspectRatio: false, cutout: '62%',
        plugins: { legend: { position: 'right', labels: { color: tickColor, font: { size: 10 }, boxWidth: 12 } } },
      },
    })

    loadLeaderboard()
    poll = setInterval(() => { fetchTimeseries(); fetchSavings(); loadLeaderboard() }, 10000)
  })

  onDestroy(() => {
    disconnectSSE()
    clearInterval(poll)
    costChart?.destroy()
    provChart?.destroy()
  })

  // Live updates.
  $: if (costChart && $timeseries) {
    costChart.data.labels = $timeseries.map(b => b.bucket)
    costChart.data.datasets[0].data = $timeseries.map(b => b.cost_usd)
    costChart.update('none')
  }
  $: if (provChart && $providerBreakdown) {
    const labels = Object.keys($providerBreakdown)
    provChart.data.labels = labels
    provChart.data.datasets[0].data = labels.map(k => $providerBreakdown[k].count)
    provChart.update('none')
  }
</script>

<header>
  <h1>NE<span>X</span>US</h1>
  <div class="status" class:live={$connected}>
    <span class="dot"></span>{$connected ? 'live' : 'connecting'}
  </div>
</header>

<main>
  {#if $savings.saved_usd > 0}
    <div class="savings">
      <div class="savings-msg">
        💸 You've saved <b>${$savings.saved_usd.toFixed(2)}</b> this month —
        <b>{Math.round($savings.percent_saved)}%</b> cheaper than Claude
        {#if $savings.cache_saved_usd > 0}<span class="cache-note">(incl. <b>${$savings.cache_saved_usd.toFixed(2)}</b> from caching)</span>{/if}
      </div>
      <div class="savings-actions">
        <a class="btn" href={shareUrl} target="_blank" rel="noopener">Share on X</a>
        <a class="btn ghost" href="/api/savings/card.svg" target="_blank" rel="noopener">Download card</a>
      </div>
    </div>
  {/if}

  <div class="stats">
    <div class="card"><span class="num accent">{$stats.total_requests}</span><span class="label">requests today</span></div>
    <div class="card"><span class="num green">${$stats.total_cost_usd.toFixed(4)}</span><span class="label">cost today</span></div>
    <div class="card"><span class="num cyan">${($stats.cache_saved_usd || 0).toFixed(4)}</span><span class="label">cache saved today</span></div>
    <div class="card"><span class="num">${$stats.forecast_usd.toFixed(2)}</span><span class="label">forecast / month</span></div>
    <div class="card"><span class="num">{Math.round($stats.avg_latency_ms)}ms</span><span class="label">avg latency</span></div>
  </div>

  {#if $providers.length}
    <div class="providers">
      {#each $providers as p (p.name)}
        <div class="chip" class:up={p.healthy} class:down={!p.healthy}>
          <span class="dot"></span>{p.name}<span class="tier">{p.tier}</span>
        </div>
      {/each}
    </div>
  {/if}

  <div class="charts">
    <div class="panel">
      <div class="panel-title">Cost — last 24h</div>
      <div class="canvas-wrap"><canvas bind:this={costCanvas}></canvas></div>
    </div>
    <div class="panel">
      <div class="panel-title">Requests by provider</div>
      <div class="canvas-wrap"><canvas bind:this={provCanvas}></canvas></div>
    </div>
  </div>

  {#if namedBoard.length}
    <div class="feed-title">Team leaderboard · saved this month</div>
    <div class="board">
      {#each namedBoard as e, i (e.user)}
        <div class="brow">
          <span class="rank">#{i + 1}</span>
          <span class="buser">{e.user}</span>
          <span class="breq">{e.requests} req</span>
          <span class="bsaved">${e.saved_usd.toFixed(2)}</span>
        </div>
      {/each}
    </div>
  {/if}

  <div class="feed-title">Live request feed</div>
  <div class="feed">
    {#if $recentRequests.length === 0}
      <div class="empty">Waiting for requests… point Claude Code at this proxy.</div>
    {/if}
    {#each $recentRequests as req (req.id)}
      <div class="row clickable" role="button" tabindex="0" title="Inspect / replay"
           on:click={() => openInspector(req.id)} on:keydown={(e) => e.key === 'Enter' && openInspector(req.id)}>
        <span class="provider">{req.provider}</span>
        <span class="model">{req.model_asked}</span>
        <span class="cx {req.complexity}">{req.complexity}</span>
        <span class="tokens">{req.input_tokens + req.output_tokens}t</span>
        <span class="cost">${req.cost_usd.toFixed(5)}</span>
        <span class="latency">{req.latency_ms}ms</span>
        <span class="code" class:err={req.status >= 400}>{req.status}</span>
      </div>
    {/each}
  </div>
</main>

{#if inspecting}
  <div class="modal-bg" role="presentation" on:click={closeInspector}>
    <div class="modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
      <div class="modal-head">
        <div>
          <b>Request #{inspecting.id}</b>
          <span class="muted">· {inspecting.provider} · {inspecting.model_used} · ${(inspecting.cost_usd || 0).toFixed(5)} · {inspecting.latency_ms}ms</span>
        </div>
        <button class="x" on:click={closeInspector}>✕</button>
      </div>

      {#if !inspecting.inspected}
        <div class="hint">No prompt/response captured. Start NEXUS with <code>--inspect</code> to enable the inspector + replay.</div>
      {:else}
        <div class="io-grid">
          <div>
            <div class="io-label">PROMPT</div>
            <pre>{pretty(inspecting.prompt)}</pre>
          </div>
          <div>
            <div class="io-label">RESPONSE</div>
            <pre>{pretty(inspecting.response)}</pre>
          </div>
        </div>

        <div class="replay">
          <span class="io-label">REPLAY vs</span>
          <select bind:value={replayProvider}>
            <option value="">auto-route</option>
            {#each $providers as p (p.name)}<option value={p.name}>{p.name}</option>{/each}
          </select>
          <button class="btn-replay" on:click={runReplay} disabled={replayLoading}>
            {replayLoading ? 'Replaying…' : 'Replay'}
          </button>
          {#if replayResult}
            {#if replayResult.error}
              <span class="err">{replayResult.error}</span>
            {:else}
              <span class="muted">→ {replayResult.provider} · {replayResult.model} · {(replayResult.input_tokens || 0) + (replayResult.output_tokens || 0)}t · {replayResult.latency_ms}ms</span>
            {/if}
          {/if}
        </div>
        {#if replayResult && !replayResult.error}
          <div class="io-label">REPLAY OUTPUT</div>
          <pre class="replay-out">{replayResult.output}</pre>
        {/if}
      {/if}
    </div>
  </div>
{/if}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    background: #050816; color: #e2e8f0;
    font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    font-size: 13px;
  }
  header { display: flex; align-items: center; justify-content: space-between; max-width: 1100px; margin: 0 auto; padding: 28px 28px 0; }
  h1 { font-size: 34px; font-weight: 800; letter-spacing: -0.04em; color: #7c3aed; }
  h1 span { color: #06b6d4; }
  .status { font-size: 11px; letter-spacing: 0.15em; color: #ef4444; display: flex; align-items: center; gap: 8px; text-transform: uppercase; }
  .status.live { color: #10b981; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 8px currentColor; }
  .status.live .dot { animation: pulse 1.6s infinite; }
  @keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.35; } }

  main { max-width: 1100px; margin: 0 auto; padding: 20px 28px 40px; }

  .savings {
    display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap;
    background: linear-gradient(90deg, rgba(124,58,237,0.16), rgba(6,182,212,0.12));
    border: 1px solid #2a2150; border-radius: 10px; padding: 14px 18px; margin-bottom: 16px;
  }
  .savings-msg { font-size: 15px; color: #e2e8f0; }
  .savings-msg b { color: #10b981; }
  .savings-actions { display: flex; gap: 8px; }
  .btn {
    display: inline-block; padding: 7px 14px; border-radius: 6px; font-size: 13px; font-weight: 600;
    text-decoration: none; background: #7c3aed; color: #fff; border: 1px solid #7c3aed;
  }
  .btn:hover { background: #6d28d9; }
  .btn.ghost { background: transparent; color: #94a3b8; border-color: #1a2035; }
  .btn.ghost:hover { color: #e2e8f0; border-color: #2a2150; }

  .stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 14px; margin-bottom: 14px; }
  .card { display: flex; flex-direction: column; gap: 4px; background: #0a0e1a; border: 1px solid #1a2035; border-radius: 8px; padding: 18px 20px; }
  .num { font-size: 26px; font-weight: 700; font-family: 'Geist Mono', monospace; }
  .num.accent { color: #7c3aed; } .num.green { color: #10b981; } .num.cyan { color: #06b6d4; }
  .cache-note { color: #06b6d4; font-size: 13px; }

  .clickable { cursor: pointer; }
  .clickable:hover { background: #11182f; }
  .modal-bg { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; padding: 24px; z-index: 50; }
  .modal { background: #0a0e1a; border: 1px solid #1a2035; border-radius: 12px; width: 100%; max-width: 920px; max-height: 86vh; overflow: auto; padding: 18px 20px; }
  .modal-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 14px; }
  .modal-head .muted { color: #64748b; font-size: 12px; }
  .x { background: #1a2035; border: 0; color: #e2e8f0; border-radius: 6px; width: 28px; height: 28px; cursor: pointer; }
  .io-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
  .io-label { font-size: 10px; color: #64748b; letter-spacing: 0.14em; margin: 8px 0 4px; }
  .modal pre { background: #050816; border: 1px solid #1a2035; border-radius: 8px; padding: 10px; font-family: 'Geist Mono','SF Mono',ui-monospace,monospace; font-size: 11px; color: #94a3b8; white-space: pre-wrap; word-break: break-word; max-height: 320px; overflow: auto; }
  .replay { display: flex; align-items: center; gap: 8px; margin-top: 14px; flex-wrap: wrap; }
  .replay select { background: #0a0e1a; color: #e2e8f0; border: 1px solid #1a2035; border-radius: 6px; padding: 6px 8px; }
  .btn-replay { background: #7c3aed; color: #fff; border: 0; border-radius: 6px; padding: 6px 14px; font-weight: 700; cursor: pointer; }
  .btn-replay:disabled { opacity: 0.5; cursor: default; }
  .replay-out { margin-top: 4px; }
  .hint { color: #94a3b8; background: #11182f; border: 1px solid #1a2035; border-radius: 8px; padding: 12px; }
  .hint code { color: #06b6d4; }

  .board { display: flex; flex-direction: column; gap: 5px; margin-bottom: 28px; }
  .brow { display: grid; grid-template-columns: 40px 1fr 90px 90px; align-items: center; gap: 10px;
    background: #0a0e1a; border: 1px solid #1a2035; border-radius: 8px; padding: 9px 14px; }
  .rank { color: #7c3aed; font-weight: 700; font-family: 'Geist Mono', monospace; }
  .buser { color: #e2e8f0; font-weight: 600; }
  .breq { color: #64748b; text-align: right; font-family: 'Geist Mono', monospace; font-size: 12px; }
  .bsaved { color: #10b981; text-align: right; font-weight: 700; font-family: 'Geist Mono', monospace; }
  .label { font-size: 10px; color: #64748b; text-transform: uppercase; letter-spacing: 0.12em; }

  .providers { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 18px; }
  .chip { display: flex; align-items: center; gap: 7px; background: #0a0e1a; border: 1px solid #1a2035; border-radius: 999px; padding: 6px 13px; font-size: 12px; }
  .chip .dot { width: 7px; height: 7px; }
  .chip.up .dot { color: #10b981; } .chip.down .dot { color: #ef4444; }
  .chip .tier { color: #64748b; font-size: 10px; text-transform: uppercase; letter-spacing: 0.1em; }

  .charts { display: grid; grid-template-columns: 1.4fr 1fr; gap: 14px; margin-bottom: 22px; }
  .panel { background: #0a0e1a; border: 1px solid #1a2035; border-radius: 8px; padding: 16px 18px; }
  .panel-title { font-size: 10px; color: #64748b; text-transform: uppercase; letter-spacing: 0.12em; margin-bottom: 12px; }
  .canvas-wrap { height: 200px; position: relative; }

  .feed-title { font-size: 11px; color: #64748b; text-transform: uppercase; letter-spacing: 0.12em; margin-bottom: 10px; }
  .feed { display: flex; flex-direction: column; gap: 5px; }
  .empty { color: #64748b; padding: 40px; text-align: center; }

  .row { display: grid; grid-template-columns: 90px 160px 90px 1fr 90px 70px 48px; gap: 14px; align-items: center; font-family: 'Geist Mono', monospace; background: #0a0e1a; border: 1px solid #1a2035; border-radius: 6px; padding: 9px 15px; animation: slideIn 0.2s ease; }
  @keyframes slideIn { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: none; } }
  .provider { color: #06b6d4; font-weight: 600; }
  .model { color: #64748b; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tokens { color: #94a3b8; } .cost { color: #10b981; } .latency { color: #64748b; }
  .code { text-align: right; color: #64748b; } .code.err { color: #ef4444; }
  .cx { font-size: 10px; padding: 2px 8px; border-radius: 4px; text-transform: uppercase; letter-spacing: 0.08em; text-align: center; }
  .cx.simple { background: rgba(16,185,129,0.12); color: #10b981; }
  .cx.standard { background: rgba(6,182,212,0.12); color: #06b6d4; }
  .cx.complex { background: rgba(124,58,237,0.14); color: #7c3aed; }
  .cx.critical { background: rgba(239,68,68,0.12); color: #ef4444; }
</style>
