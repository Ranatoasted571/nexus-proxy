package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

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
