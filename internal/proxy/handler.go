package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/router"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

// EventPublisher lets the handler push live events to the dashboard over SSE.
// (*dashboard.SSEBroker satisfies this — passed in from main to avoid coupling.)
type EventPublisher interface {
	Publish(eventType string, data interface{})
}

// activeProvider bundles a provider implementation with its API key.
type activeProvider struct {
	impl   providers.Provider
	apiKey string
}

// Handler handles incoming Claude Code requests.
type Handler struct {
	httpClient *http.Client
	router     *router.Router
	providers  map[string]*activeProvider
	db         *storage.DB
	broker     EventPublisher // may be nil
	budget     *budgetTracker
	cache      *responseCache // may be nil (disabled)
	stopHealth chan struct{}
}

// NewHandler builds the provider set + router from config and wires in the
// shared storage and (optional) event broker.
func NewHandler(cfg *Config, db *storage.DB, broker EventPublisher) (*Handler, error) {
	appCfg, err := config.Load(cfg.ConfigPath)
	if err != nil {
		return nil, err
	}

	rt := router.New(router.RoutingStrategy(appCfg.Routing.Strategy))
	active := make(map[string]*activeProvider)
	for _, pc := range appCfg.Providers {
		key := config.ResolveKey(pc.APIKey)
		impl, err := providers.New(providers.Spec{
			Name:        pc.Name,
			Type:        pc.Type,
			APIKey:      key,
			BaseURL:     pc.BaseURL,
			Models:      pc.Models,
			Tier:        pc.Tier,
			ModelMap:    pc.ModelMap,
			InputPer1M:  pc.InputPer1M,
			OutputPer1M: pc.OutputPer1M,
			Region:      pc.Region,
			Project:     pc.Project,
			APIVersion:  pc.APIVersion,
		})
		if err != nil {
			log.Warn().Str("provider", pc.Name).Err(err).Msg("Skipping provider")
			continue
		}
		active[impl.Name()] = &activeProvider{impl: impl, apiKey: key}
		rt.AddProvider(&router.Provider{
			Name:    impl.Name(),
			BaseURL: impl.BaseURL(),
			APIKey:  key,
			Tier:    impl.Tier(),
			Pricing: router.Pricing{InputPer1M: impl.Pricing().InputPer1M, OutputPer1M: impl.Pricing().OutputPer1M},
			Healthy: true, // optimistic; runtime failover handles outages/rate-limits
		})
	}

	budgetLimit := cfg.DailyBudgetUSD
	if budgetLimit <= 0 {
		budgetLimit = appCfg.Routing.DailyBudgetUSD
	}
	var spentToday float64
	if s, err := db.GetStats("today"); err == nil {
		spentToday = s.TotalCostUSD
	}

	h := &Handler{
		httpClient: &http.Client{Timeout: 5 * time.Minute},
		router:     rt,
		providers:  active,
		db:         db,
		broker:     broker,
		budget:     newBudgetTracker(budgetLimit, spentToday),
		stopHealth: make(chan struct{}),
	}
	if !cfg.DisableCache {
		semantic := cfg.SemanticCache || appCfg.Routing.SemanticCache
		threshold := cfg.SemanticThreshold
		if threshold <= 0 {
			threshold = appCfg.Routing.SemanticThreshold
		}
		h.cache = newResponseCache(5*time.Minute, 500, semantic, threshold)
		log.Info().Msg("Response cache enabled (5m TTL) — identical requests served instantly & free")
		if semantic {
			log.Info().Float64("threshold", h.cache.threshold).Msg("Semantic cache enabled — near-identical tool-less requests served from cache")
		}
	}

	if len(active) == 0 {
		log.Info().Msg("No providers configured — zero-config mode (forwarding directly to Anthropic)")
	} else {
		log.Info().Int("providers", len(active)).Str("strategy", appCfg.Routing.Strategy).Msg("Router configured")
		if budgetLimit > 0 {
			log.Info().Float64("daily_budget_usd", budgetLimit).Msg("Daily budget cap enabled — free/local only once exceeded")
		}
		go h.healthLoop() // periodic background health checks
	}
	return h, nil
}

// Close stops background work. The shared DB is owned and closed by the caller.
func (h *Handler) Close() error {
	if h.stopHealth != nil {
		close(h.stopHealth)
		h.stopHealth = nil
	}
	return nil
}

// ProviderCount returns the number of configured providers.
func (h *Handler) ProviderCount() int { return len(h.providers) }

// CacheEnabled reports whether the response cache is active.
func (h *Handler) CacheEnabled() bool { return h.cache != nil }

// healthLoop periodically health-checks every provider and updates the router,
// so unhealthy providers are skipped (and recover automatically).
func (h *Handler) healthLoop() {
	stop := h.stopHealth
	check := func() {
		var wg sync.WaitGroup
		for name, ap := range h.providers {
			wg.Add(1)
			go func(name string, ap *activeProvider) {
				defer wg.Done()
				h.router.SetHealthy(name, ap.impl.HealthCheck() == nil)
			}(name, ap)
		}
		wg.Wait()
	}
	check() // initial pass shortly after startup
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			check()
		}
	}
}

// HandleMessages is the main handler for POST /v1/messages (Claude Code calls this).
func (h *Handler) HandleMessages(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Response cache: serve identical requests instantly (and free).
	if h.cache != nil {
		key := cacheKey("m", body)
		if e, ok := h.cache.get(key); ok {
			h.serveCached(w, e, startTime)
			return
		}
		var vec sparseVec
		hasTools := false
		if h.cache.semantic {
			if text, ht, ok := promptText(body); ok {
				hasTools = ht
				if !ht {
					vec = embed(text)
					if e, ok := h.cache.getSemantic(quickModel(body), vec); ok {
						h.serveCached(w, e, startTime)
						return
					}
				}
			}
		}
		cw := newCachingWriter(w)
		defer func() {
			if cw.cacheable() {
				e := cw.entry()
				e.model = quickModel(body)
				e.vec = vec
				e.hasTools = hasTools
				h.cache.set(key, e)
			}
		}()
		w = cw
	}

	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON in request body")
		return
	}

	// Parse messages as raw maps for the classifier.
	var raw struct {
		Messages []map[string]interface{} `json:"messages"`
	}
	_ = json.Unmarshal(body, &raw)
	hasTools := len(req.Tools) > 0
	complexity := router.ClassifyRequest(req.Model, raw.Messages, hasTools)

	log.Debug().
		Str("model", req.Model).
		Str("complexity", complexity.String()).
		Int("messages", len(req.Messages)).
		Bool("stream", req.Stream).
		Bool("tools", hasTools).
		Msg("Incoming request")

	// Zero-config (no providers) or empty chain → forward straight to Anthropic.
	chain := h.router.RouteChain(req.Model, complexity)
	if len(h.providers) == 0 || len(chain) == 0 {
		h.forwardDirectAnthropic(w, r, req, body, startTime, complexity)
		return
	}

	// Daily budget cap: once today's spend exceeds the limit, restrict to
	// free/local providers (paid tiers are skipped until the next day).
	if h.budget.Over() {
		if cheap := freeLocalOnly(chain); len(cheap) > 0 {
			chain = cheap
		} else {
			log.Warn().Msg("Daily budget exceeded but no free/local provider available — using a paid provider")
		}
	}

	// Walk the chain, failing over to the next provider on transport errors and
	// on retryable HTTP statuses (rate-limit / server errors). A provider's own
	// 4xx (e.g. 401 bad key) is relayed to the client as-is.
	for i, cand := range chain {
		active := h.providers[cand.Name]
		if active == nil {
			continue
		}
		resp, err := h.callUpstream(active, req, body, r.Header)
		if err != nil {
			log.Warn().Str("provider", cand.Name).Err(err).Msg("Provider unreachable, trying next")
			continue
		}
		if isRetryableStatus(resp.StatusCode) && i < len(chain)-1 {
			resp.Body.Close()
			log.Warn().Str("provider", cand.Name).Int("status", resp.StatusCode).Msg("Retryable error, failing over to next provider")
			continue
		}
		switch {
		case providers.IsOpenAICompatible(active.impl.Name()) && req.Stream:
			h.relayOpenAIStream(w, active, req, resp, startTime, complexity)
		case providers.IsOpenAICompatible(active.impl.Name()):
			h.relayOpenAI(w, active, req, resp, startTime, complexity)
		default:
			// Anthropic-format. Bedrock/Vertex return a full body (buffered);
			// native Anthropic streams through.
			if _, custom := active.impl.(providers.AnthropicNative); custom {
				h.relayAnthropicBuffered(w, active, req, resp, startTime, complexity)
			} else if req.Stream {
				h.relayAnthropicStream(w, r, active, req, resp, startTime, complexity)
			} else {
				h.relayAnthropicSync(w, active, req, resp, startTime, complexity)
			}
		}
		return
	}

	h.writeError(w, http.StatusBadGateway, "all providers unreachable")
}

// callUpstream issues the upstream HTTP request for the given provider.
// It returns a transport error only — provider HTTP errors come back in *http.Response.
func (h *Handler) callUpstream(active *activeProvider, req AnthropicRequest, body []byte, origHeaders http.Header) (*http.Response, error) {
	if providers.IsOpenAICompatible(active.impl.Name()) {
		oaiReq, err := TransformToOpenAI(req, active.impl.MapModel(req.Model))
		if err != nil {
			return nil, fmt.Errorf("request transform failed: %w", err)
		}
		oaiReq.Stream = req.Stream // stream upstream when the client streams
		if req.Stream {
			oaiReq.StreamOptions = &OpenAIStreamOptions{IncludeUsage: true}
		}
		payload, err := json.Marshal(oaiReq)
		if err != nil {
			return nil, err
		}
		httpReq, err := http.NewRequest("POST", active.impl.ChatCompletionsURL(), bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		h.authorize(active, httpReq, payload, active.apiKey)
		return h.httpClient.Do(httpReq)
	}

	// Anthropic-format providers (Anthropic, plus Bedrock/Vertex via AnthropicNative).
	url := active.impl.BaseURL() + "/v1/messages"
	sendBody := body
	if an, ok := active.impl.(providers.AnthropicNative); ok {
		url = an.MessagesURL(req.Model)
		sendBody = an.PrepareBody(body, req.Model)
	}
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(sendBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if _, ok := active.impl.(providers.Authorizer); ok {
		h.authorize(active, httpReq, sendBody, active.apiKey)
	} else {
		httpReq.Header.Set("x-api-key", h.resolveAnthropicKey(active, origHeaders))
		httpReq.Header.Set("anthropic-version", "2023-06-01")
		if v := origHeaders.Get("anthropic-beta"); v != "" {
			httpReq.Header.Set("anthropic-beta", v)
		}
		if v := origHeaders.Get("anthropic-version"); v != "" {
			httpReq.Header.Set("anthropic-version", v)
		}
	}
	return h.httpClient.Do(httpReq)
}

// authorize applies a provider's custom auth (Azure api-key, Vertex bearer,
// Bedrock SigV4) when it implements Authorizer; otherwise falls back to Bearer.
func (h *Handler) authorize(active *activeProvider, req *http.Request, body []byte, key string) {
	if az, ok := active.impl.(providers.Authorizer); ok {
		az.Authorize(req, body, key)
		return
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

// forwardDirectAnthropic is the zero-config path: forward to Anthropic using the
// client's (or the server env's) key, exactly like Sprint 1.
func (h *Handler) forwardDirectAnthropic(w http.ResponseWriter, r *http.Request, req AnthropicRequest, body []byte, startTime time.Time, complexity router.Complexity) {
	active := &activeProvider{impl: providers.NewAnthropic(""), apiKey: ""}
	resp, err := h.callUpstream(active, req, body, r.Header)
	if err != nil {
		log.Error().Err(err).Msg("Provider request failed")
		h.writeError(w, http.StatusBadGateway, fmt.Sprintf("provider error: %v", err))
		return
	}
	if req.Stream {
		h.relayAnthropicStream(w, r, active, req, resp, startTime, complexity)
	} else {
		h.relayAnthropicSync(w, active, req, resp, startTime, complexity)
	}
}

// relayAnthropicSync relays a non-streaming native-Anthropic response.
func (h *Handler) relayAnthropicSync(w http.ResponseWriter, active *activeProvider, req AnthropicRequest, resp *http.Response, startTime time.Time, complexity router.Complexity) {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read provider response")
		return
	}

	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set("X-Nexus-Provider", active.impl.Name())
	w.Header().Set("X-Nexus-Latency", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(respBody); err != nil {
		log.Warn().Err(err).Msg("Failed to write response to client")
	}

	u := anthropicUsageFull(respBody)
	h.logResult(active, req, complexity, u, resp.StatusCode, time.Since(startTime), false)
	log.Info().
		Str("provider", active.impl.Name()).
		Int("status", resp.StatusCode).
		Int("cache_read", u.CacheRead).
		Int64("latency_ms", time.Since(startTime).Milliseconds()).
		Str("complexity", complexity.String()).
		Msg("Request completed")
}

// relayAnthropicBuffered handles Anthropic-format providers that return a full
// (non-streaming) body — Bedrock/Vertex. It relays the JSON, or synthesizes the
// Anthropic SSE sequence when the client asked to stream.
func (h *Handler) relayAnthropicBuffered(w http.ResponseWriter, active *activeProvider, req AnthropicRequest, resp *http.Response, startTime time.Time, complexity router.Complexity) {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read provider response")
		return
	}

	if resp.StatusCode >= 400 {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Nexus-Provider", active.impl.Name())
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)
		h.logResult(active, req, complexity, tokenUsage{}, resp.StatusCode, time.Since(startTime), req.Stream)
		log.Warn().Str("provider", active.impl.Name()).Int("status", resp.StatusCode).Msg("Provider returned error")
		return
	}

	u := anthropicUsageFull(respBody)
	if req.Stream {
		var ar AnthropicResponse
		if json.Unmarshal(respBody, &ar) == nil && len(ar.Content) > 0 {
			if ar.Model == "" {
				ar.Model = req.Model
			}
			writeAnthropicSSE(w, active.impl.Name(), ar)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Nexus-Provider", active.impl.Name())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(respBody)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Nexus-Provider", active.impl.Name())
		w.Header().Set("X-Nexus-Latency", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBody)
	}

	h.logResult(active, req, complexity, u, http.StatusOK, time.Since(startTime), req.Stream)
	log.Info().
		Str("provider", active.impl.Name()).
		Int("in", u.In).Int("out", u.Out).
		Int64("latency_ms", time.Since(startTime).Milliseconds()).
		Bool("stream", req.Stream).
		Msg("Request completed (anthropic-native)")
}

// resolveAnthropicKey picks the API key for an Anthropic forward: the configured
// key if present, otherwise the client's key, otherwise the server's env key.
func (h *Handler) resolveAnthropicKey(active *activeProvider, origHeaders http.Header) string {
	if active != nil && active.apiKey != "" && active.apiKey != "nexus-local" {
		return active.apiKey
	}
	key := extractAPIKey(origHeaders)
	if key == "" || key == "nexus-local" {
		if env := os.Getenv("ANTHROPIC_API_KEY"); env != "" && env != "nexus-local" {
			key = env
		}
	}
	return key
}

// logResult records a completed request to storage and pushes live events.
func (h *Handler) logResult(active *activeProvider, req AnthropicRequest, complexity router.Complexity, u tokenUsage, status int, latency time.Duration, stream bool) {
	now := time.Now()
	pricing := active.impl.Pricing()
	cost := pricing.CalculateCostFull(u.In, u.Out, u.CacheRead, u.CacheWrite)
	cacheSaved := pricing.CacheReadSavings(u.CacheRead)
	h.budget.Add(cost)
	rec := &storage.Request{
		CreatedAt:        now,
		ModelAsked:       req.Model,
		ModelUsed:        active.impl.MapModel(req.Model),
		Provider:         active.impl.Name(),
		Complexity:       complexity.String(),
		InputTokens:      u.In,
		OutputTokens:     u.Out,
		CacheReadTokens:  u.CacheRead,
		CacheWriteTokens: u.CacheWrite,
		CostUSD:          cost,
		CacheSavedUSD:    cacheSaved,
		LatencyMS:        latency.Milliseconds(),
		Status:           status,
		Stream:           stream,
	}

	var id int64
	if h.db != nil {
		var err error
		if id, err = h.db.LogRequest(rec); err != nil {
			log.Warn().Err(err).Msg("Failed to log request")
		}
	}

	if h.broker != nil {
		h.broker.Publish("request", requestEvent{
			ID:           id,
			Provider:     rec.Provider,
			ModelAsked:   rec.ModelAsked,
			ModelUsed:    rec.ModelUsed,
			Complexity:   rec.Complexity,
			InputTokens:  u.In,
			OutputTokens: u.Out,
			CacheRead:    u.CacheRead,
			CacheWrite:   u.CacheWrite,
			CostUSD:      cost,
			CacheSavedUSD: cacheSaved,
			LatencyMS:    rec.LatencyMS,
			Status:       status,
			Timestamp:    now.Format(time.RFC3339),
		})
		h.publishStats()
	}
}

// publishStats computes today's aggregate stats and pushes them over SSE.
func (h *Handler) publishStats() {
	if h.db == nil || h.broker == nil {
		return
	}
	stats, err := h.db.GetStats("today")
	if err != nil {
		return
	}
	forecast, _ := h.db.GetCostForecast()
	h.broker.Publish("stats", map[string]interface{}{
		"total_requests":  stats.TotalRequests,
		"total_cost_usd":  stats.TotalCostUSD,
		"total_tokens":    stats.TotalInputTokens + stats.TotalOutputTokens,
		"forecast_usd":    forecast,
		"avg_latency_ms":  stats.AvgLatencyMS,
		"cache_saved_usd": stats.CacheSavedUSD,
		"cache_read_tokens": stats.CacheReadTokens,
	})
}

// serveCached writes a cached response and logs it as a (free, instant) cache hit.
func (h *Handler) serveCached(w http.ResponseWriter, e cacheEntry, start time.Time) {
	if e.ctype != "" {
		w.Header().Set("Content-Type", e.ctype)
	}
	w.Header().Set("X-Nexus-Provider", "cache")
	w.Header().Set("X-Nexus-Cache", "HIT")
	w.WriteHeader(e.status)
	_, _ = w.Write(e.body)

	latency := time.Since(start)
	if h.db != nil {
		_, _ = h.db.LogRequest(&storage.Request{
			CreatedAt: time.Now(), Provider: "cache",
			ModelAsked: orStr(e.model, "cache"), ModelUsed: "cache", Complexity: "cached",
			InputTokens: e.in, OutputTokens: e.out, CostUSD: 0, LatencyMS: latency.Milliseconds(), Status: e.status,
		})
	}
	if h.broker != nil {
		h.broker.Publish("request", requestEvent{
			Provider: "cache", ModelAsked: orStr(e.model, "—"), ModelUsed: "cache", Complexity: "cached",
			InputTokens: e.in, OutputTokens: e.out, CostUSD: 0, LatencyMS: latency.Milliseconds(),
			Status: e.status, Timestamp: time.Now().Format(time.RFC3339),
		})
		h.publishStats()
	}
	log.Info().Int64("latency_ms", latency.Milliseconds()).Int("in", e.in).Int("out", e.out).Msg("Cache hit ⚡ (free)")
}

func orStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// writeError writes a JSON error response in Anthropic's error shape.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    "proxy_error",
			"message": message,
		},
	})
}

// ─── Header & parsing helpers ──────────────────────────────────────────────

// hopByHopHeaders should not be forwarded verbatim between client and provider.
// Content-Length and Content-Encoding are dropped because the body is re-read
// (and transparently decompressed) by the Go transport before we relay it.
var hopByHopHeaders = map[string]bool{
	"Connection":        true,
	"Proxy-Connection":  true,
	"Keep-Alive":        true,
	"Transfer-Encoding": true,
	"Te":                true,
	"Trailer":           true,
	"Upgrade":           true,
	"Content-Length":    true,
	"Content-Encoding":  true,
}

// copyResponseHeaders copies non-hop-by-hop headers from src into dst.
func copyResponseHeaders(dst, src http.Header) {
	for k, vals := range src {
		if hopByHopHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

// extractAPIKey pulls the API key from x-api-key or a Bearer Authorization header.
func extractAPIKey(h http.Header) string {
	if key := h.Get("x-api-key"); key != "" {
		return key
	}
	if auth := h.Get("Authorization"); auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// isRetryableStatus reports whether an upstream HTTP status should trigger
// failover to the next provider in the chain (rate-limit / transient errors).
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}

// ─── Budget tracking ───────────────────────────────────────────────────────

// budgetTracker enforces a soft daily spend cap. When exceeded, the handler
// restricts routing to free/local providers until the next day.
type budgetTracker struct {
	mu    sync.Mutex
	limit float64
	day   string
	spent float64
}

func newBudgetTracker(limit, spentToday float64) *budgetTracker {
	return &budgetTracker{limit: limit, day: todayKey(), spent: spentToday}
}

func todayKey() string { return time.Now().Format("2006-01-02") }

// roll resets the running total when the day changes (caller holds b.mu).
func (b *budgetTracker) roll() {
	if d := todayKey(); d != b.day {
		b.day = d
		b.spent = 0
	}
}

func (b *budgetTracker) Add(cost float64) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.roll()
	b.spent += cost
}

func (b *budgetTracker) Over() bool {
	if b == nil || b.limit <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.roll()
	return b.spent >= b.limit
}

// freeLocalOnly filters a route chain down to free/local providers.
func freeLocalOnly(chain []*router.Provider) []*router.Provider {
	var out []*router.Provider
	for _, p := range chain {
		if p.Tier == "free" || p.Tier == "local" {
			out = append(out, p)
		}
	}
	return out
}

// parseAnthropicUsage extracts token usage from a non-streaming Anthropic response.
func parseAnthropicUsage(body []byte) (in, out int) {
	var r struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &r)
	return r.Usage.InputTokens, r.Usage.OutputTokens
}

// ─── Types ─────────────────────────────────────────────────────────────────

// requestEvent is the payload pushed to the dashboard after each request.
type requestEvent struct {
	ID            int64   `json:"id"`
	Provider      string  `json:"provider"`
	ModelAsked    string  `json:"model_asked"`
	ModelUsed     string  `json:"model_used"`
	Complexity    string  `json:"complexity"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	CacheRead     int     `json:"cache_read,omitempty"`
	CacheWrite    int     `json:"cache_write,omitempty"`
	CostUSD       float64 `json:"cost_usd"`
	CacheSavedUSD float64 `json:"cache_saved_usd,omitempty"`
	LatencyMS     int64   `json:"latency_ms"`
	Status        int     `json:"status"`
	Timestamp     string  `json:"timestamp"`
}

// AnthropicRequest represents an incoming Claude Code request
type AnthropicRequest struct {
	Model     string        `json:"model"`
	Messages  []Message     `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
	System    interface{}   `json:"system,omitempty"`
	Tools     []interface{} `json:"tools,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ProviderConfig holds provider connection details
type ProviderConfig struct {
	Name    string
	BaseURL string
	APIKey  string
	Model   string // overridden model name (if any)
}
