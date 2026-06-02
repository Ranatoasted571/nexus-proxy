package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

// Server serves the dashboard UI and API.
type Server struct {
	port   int
	broker *SSEBroker
	db     *storage.DB
	srv    *http.Server
}

// NewServer creates a new dashboard server.
func NewServer(port int, broker *SSEBroker, db *storage.DB) *Server {
	return &Server{
		port:   port,
		broker: broker,
		db:     db,
	}
}

// Start starts the dashboard HTTP server.
func (s *Server) Start() error {
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.Routes(),
	}
	return s.srv.ListenAndServe()
}

// Routes builds the dashboard HTTP handler (router + CORS). Exposed for tests.
func (s *Server) Routes() http.Handler {
	r := mux.NewRouter()

	// SSE endpoint — live updates.
	if s.broker != nil {
		r.Handle("/events", s.broker).Methods("GET")
	}

	// API endpoints.
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/stats", s.handleStats).Methods("GET")
	api.HandleFunc("/requests", s.handleRequests).Methods("GET")
	api.HandleFunc("/providers", s.handleProviders).Methods("GET")
	api.HandleFunc("/timeseries", s.handleTimeseries).Methods("GET")
	api.HandleFunc("/breakdown", s.handleBreakdown).Methods("GET")
	api.HandleFunc("/savings", s.handleSavings).Methods("GET")
	api.HandleFunc("/savings/card.svg", s.handleSavingsCard).Methods("GET")

	// Serve the embedded dashboard UI (Svelte build, or the committed fallback).
	if sub, err := distFileSystem(); err == nil {
		r.PathPrefix("/").Handler(http.FileServer(http.FS(sub)))
	} else {
		r.PathPrefix("/").HandlerFunc(s.handleSPAFallback)
	}

	return corsMiddleware(r)
}

// Shutdown gracefully stops the dashboard server.
func (s *Server) Shutdown() error {
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "today"
	}

	if s.db == nil {
		writeJSON(w, emptyStats(period))
		return
	}
	stats, err := s.db.GetStats(period)
	if err != nil {
		writeJSON(w, emptyStats(period))
		return
	}
	forecast, _ := s.db.GetCostForecast()
	writeJSON(w, map[string]interface{}{
		"period":         period,
		"total_requests": stats.TotalRequests,
		"total_cost_usd": stats.TotalCostUSD,
		"total_tokens":   stats.TotalInputTokens + stats.TotalOutputTokens,
		"forecast_usd":   forecast,
		"avg_latency_ms": stats.AvgLatencyMS,
	})
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	out := []RequestEvent{}
	if s.db != nil {
		if reqs, err := s.db.GetRecentRequests(50); err == nil {
			for _, q := range reqs {
				out = append(out, RequestEvent{
					ID:           q.ID,
					Provider:     q.Provider,
					ModelAsked:   q.ModelAsked,
					ModelUsed:    q.ModelUsed,
					Complexity:   q.Complexity,
					InputTokens:  q.InputTokens,
					OutputTokens: q.OutputTokens,
					CostUSD:      q.CostUSD,
					LatencyMS:    q.LatencyMS,
					Status:       q.Status,
					Timestamp:    q.CreatedAt.Local().Format(time.RFC3339),
				})
			}
		}
	}
	writeJSON(w, map[string]interface{}{"requests": out, "total": len(out)})
}

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	provs := []map[string]interface{}{}
	if cfg, err := config.Load(config.DefaultPath()); err == nil {
		for _, p := range cfg.Providers {
			provs = append(provs, map[string]interface{}{
				"name":    p.Name,
				"tier":    p.Tier,
				"healthy": true, // live health checks are surfaced via `nexus status`
			})
		}
	}
	writeJSON(w, map[string]interface{}{"providers": provs})
}

func (s *Server) handleTimeseries(w http.ResponseWriter, r *http.Request) {
	series := []storage.TimeBucket{}
	if s.db != nil {
		if ts, err := s.db.GetHourlySeries(); err == nil {
			series = ts
		}
	}
	writeJSON(w, map[string]interface{}{"series": series})
}

func (s *Server) handleBreakdown(w http.ResponseWriter, r *http.Request) {
	provs := []map[string]interface{}{}
	complexity := map[string]int{}
	if s.db != nil {
		if bd, err := s.db.GetProviderBreakdown(); err == nil {
			for _, p := range bd {
				provs = append(provs, map[string]interface{}{
					"provider": p.Provider,
					"requests": p.Requests,
					"cost_usd": p.TotalCostUSD,
					"tokens":   p.TotalTokens,
				})
			}
		}
		if cx, err := s.db.GetComplexityBreakdown(); err == nil {
			complexity = cx
		}
	}
	writeJSON(w, map[string]interface{}{"providers": provs, "complexity": complexity})
}

func (s *Server) handleSavings(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}
	if s.db != nil {
		if sv, err := s.db.GetSavings(period); err == nil {
			writeJSON(w, sv)
			return
		}
	}
	writeJSON(w, &storage.Savings{Period: period})
}

func (s *Server) handleSavingsCard(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}
	sv := &storage.Savings{Period: period}
	if s.db != nil {
		if x, err := s.db.GetSavings(period); err == nil {
			sv = x
		}
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(savingsSVG(sv)))
}

// savingsSVG renders a shareable savings card (640x360, social-friendly).
func savingsSVG(sv *storage.Savings) string {
	label := map[string]string{"today": "today", "week": "this week", "month": "this month"}[sv.Period]
	if label == "" {
		label = sv.Period
	}
	pct := int(sv.PercentSaved + 0.5)
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360" viewBox="0 0 640 360" font-family="Inter,-apple-system,Segoe UI,sans-serif">
  <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#7c3aed"/><stop offset="1" stop-color="#06b6d4"/></linearGradient></defs>
  <rect width="640" height="360" fill="#050816"/>
  <rect width="640" height="6" fill="url(#g)"/>
  <text x="40" y="80" font-size="40" font-weight="800" letter-spacing="-1" fill="#7c3aed">NE<tspan fill="#06b6d4">X</tspan>US</text>
  <text x="40" y="106" font-size="15" fill="#64748b">smart proxy for Claude Code</text>
  <text x="40" y="214" font-size="84" font-weight="800" fill="#10b981">$%s</text>
  <text x="40" y="252" font-size="22" fill="#e2e8f0">saved %s vs. Claude</text>
  <text x="40" y="300" font-size="17" fill="#94a3b8">%d requests routed  •  %d%% cheaper</text>
  <text x="40" y="338" font-size="14" fill="#475569">github.com/lynuxis2026-pixel/nexus-proxy</text>
</svg>`, formatUSD(sv.SavedUSD), label, sv.Requests, pct)
}

func formatUSD(v float64) string {
	switch {
	case v >= 100:
		return fmt.Sprintf("%.0f", v)
	case v >= 1:
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%.4f", v)
	}
}

// handleSPAFallback is only used if the embedded filesystem is unavailable.
func (s *Server) handleSPAFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html><html><head><title>NEXUS</title></head>
<body style="background:#050816;color:#e2e8f0;font-family:monospace;padding:40px">
<h1>NEXUS</h1><p>Dashboard assets not embedded. Run <code>make build</code>.</p>
<p><a style="color:#7c3aed" href="/api/stats">/api/stats</a></p></body></html>`))
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func emptyStats(period string) map[string]interface{} {
	return map[string]interface{}{
		"period":         period,
		"total_requests": 0,
		"total_cost_usd": 0.0,
		"total_tokens":   0,
		"forecast_usd":   0.0,
		"avg_latency_ms": 0,
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
