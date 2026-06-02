package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

func TestRoutes_Dashboard(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, broker: NewSSEBroker()}
	h := s.Routes()

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec
	}

	// SPA index from the embedded dashboard.
	if rec := get("/"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("/ (SPA) = %d", rec.Code)
	}
	for _, p := range []string{"/api/stats", "/api/requests", "/api/providers", "/api/timeseries", "/api/breakdown", "/api/savings"} {
		if rec := get(p); rec.Code != 200 {
			t.Errorf("%s = %d", p, rec.Code)
		}
	}
	if rec := get("/api/savings/card.svg"); rec.Code != 200 || rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Errorf("/api/savings/card.svg = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}

func TestDashboardSavings(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Sonnet-asked, routed free at $0 → big savings.
	_, _ = db.LogRequest(&storage.Request{
		CreatedAt: time.Now(), Provider: "groq", ModelAsked: "claude-sonnet-4-6", ModelUsed: "llama",
		Complexity: "standard", InputTokens: 1_000_000, OutputTokens: 1_000_000, CostUSD: 0, Status: 200,
	})
	s := &Server{db: db}

	rec := httptest.NewRecorder()
	s.handleSavings(rec, httptest.NewRequest("GET", "/api/savings?period=today", nil))
	var sv map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &sv); err != nil {
		t.Fatalf("savings JSON: %v", err)
	}
	if saved, _ := sv["saved_usd"].(float64); saved < 17.9 {
		t.Errorf("saved_usd = %v, want ~18", sv["saved_usd"])
	}

	rec = httptest.NewRecorder()
	s.handleSavingsCard(rec, httptest.NewRequest("GET", "/api/savings/card.svg?period=today", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("card Content-Type = %q, want image/svg+xml", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<svg") || !strings.Contains(body, "saved") || !strings.Contains(body, "nexus-proxy") {
		t.Errorf("savings card missing expected content:\n%s", body)
	}
}

func TestDashboardAPI(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 3; i++ {
		if _, err := db.LogRequest(&storage.Request{
			CreatedAt: time.Now(), Provider: "groq", ModelAsked: "claude-haiku-4-5",
			ModelUsed: "llama-3.3-70b-versatile", Complexity: "simple",
			InputTokens: 10, OutputTokens: 5, CostUSD: 0.001, LatencyMS: 12, Status: 200,
		}); err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{db: db}

	num := func(v interface{}) float64 { f, _ := v.(float64); return f }
	call := func(fn func(w *httptest.ResponseRecorder)) map[string]interface{} {
		rec := httptest.NewRecorder()
		fn(rec)
		var m map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatalf("bad JSON: %v (%s)", err, rec.Body.String())
		}
		return m
	}

	stats := call(func(rec *httptest.ResponseRecorder) {
		s.handleStats(rec, httptest.NewRequest("GET", "/api/stats?period=today", nil))
	})
	if num(stats["total_requests"]) != 3 {
		t.Errorf("stats total_requests = %v, want 3", stats["total_requests"])
	}

	reqs := call(func(rec *httptest.ResponseRecorder) {
		s.handleRequests(rec, httptest.NewRequest("GET", "/api/requests", nil))
	})
	if num(reqs["total"]) != 3 {
		t.Errorf("requests total = %v, want 3", reqs["total"])
	}

	ts := call(func(rec *httptest.ResponseRecorder) {
		s.handleTimeseries(rec, httptest.NewRequest("GET", "/api/timeseries", nil))
	})
	if ts["series"] == nil {
		t.Error("timeseries response missing 'series'")
	}

	bd := call(func(rec *httptest.ResponseRecorder) {
		s.handleBreakdown(rec, httptest.NewRequest("GET", "/api/breakdown", nil))
	})
	cx, _ := bd["complexity"].(map[string]interface{})
	if num(cx["simple"]) != 3 {
		t.Errorf("breakdown complexity.simple = %v, want 3", cx["simple"])
	}
	provs, _ := bd["providers"].([]interface{})
	if len(provs) != 1 {
		t.Errorf("breakdown providers = %d, want 1", len(provs))
	}
}
