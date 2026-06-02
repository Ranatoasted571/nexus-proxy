package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLogAndGetRecent(t *testing.T) {
	db := newTestDB(t)
	id, err := db.LogRequest(&Request{
		CreatedAt:    time.Now(),
		ModelAsked:   "claude-haiku-4-5",
		ModelUsed:    "llama-3.3-70b-versatile",
		Provider:     "groq",
		Complexity:   "simple",
		InputTokens:  10,
		OutputTokens: 5,
		CostUSD:      0,
		LatencyMS:    12,
		Status:       200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Errorf("expected positive row id, got %d", id)
	}

	reqs, err := db.GetRecentRequests(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Provider != "groq" || reqs[0].InputTokens != 10 || reqs[0].Status != 200 {
		t.Errorf("unexpected row: %+v", reqs[0])
	}
}

func TestStatsToday(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 3; i++ {
		if _, err := db.LogRequest(&Request{
			CreatedAt: time.Now(), ModelAsked: "m", ModelUsed: "m", Provider: "deepseek",
			Complexity: "standard", InputTokens: 100, OutputTokens: 50, CostUSD: 0.001, LatencyMS: 100, Status: 200,
		}); err != nil {
			t.Fatal(err)
		}
	}
	s, err := db.GetStats("today")
	if err != nil {
		t.Fatal(err)
	}
	if s.TotalRequests != 3 {
		t.Errorf("expected 3 requests today, got %d", s.TotalRequests)
	}
	if s.TotalInputTokens != 300 || s.TotalOutputTokens != 150 {
		t.Errorf("token totals wrong: in=%d out=%d", s.TotalInputTokens, s.TotalOutputTokens)
	}
}

func TestSavings(t *testing.T) {
	db := newTestDB(t)
	// A Sonnet-asked request that actually cost $0 (routed to a free provider),
	// 1M input + 1M output → Claude baseline = $3 + $15 = $18.
	if _, err := db.LogRequest(&Request{
		CreatedAt: time.Now(), ModelAsked: "claude-sonnet-4-6", ModelUsed: "llama-3.3-70b-versatile",
		Provider: "groq", Complexity: "standard", InputTokens: 1_000_000, OutputTokens: 1_000_000,
		CostUSD: 0, Status: 200,
	}); err != nil {
		t.Fatal(err)
	}
	sv, err := db.GetSavings("today")
	if err != nil {
		t.Fatal(err)
	}
	if sv.BaselineUSD < 17.9 || sv.BaselineUSD > 18.1 {
		t.Errorf("baseline = %v, want ~18", sv.BaselineUSD)
	}
	if sv.SavedUSD < 17.9 {
		t.Errorf("saved = %v, want ~18", sv.SavedUSD)
	}
	if sv.PercentSaved < 99 {
		t.Errorf("percent saved = %v, want ~100", sv.PercentSaved)
	}
}

func TestProviderBreakdown(t *testing.T) {
	db := newTestDB(t)
	mk := func(p string) *Request {
		return &Request{CreatedAt: time.Now(), ModelAsked: "m", ModelUsed: "m", Provider: p, Complexity: "simple", Status: 200}
	}
	db.LogRequest(mk("groq"))
	db.LogRequest(mk("groq"))
	db.LogRequest(mk("anthropic"))

	bd, err := db.GetProviderBreakdown()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, p := range bd {
		counts[p.Provider] = p.Requests
	}
	if counts["groq"] != 2 || counts["anthropic"] != 1 {
		t.Errorf("breakdown wrong: %+v", counts)
	}
}
