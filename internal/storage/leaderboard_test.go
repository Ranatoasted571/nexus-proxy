package storage

import (
	"testing"
	"time"
)

func TestGetLeaderboard(t *testing.T) {
	db := newTestDB(t)
	now := time.Now()
	// alice: a Sonnet-asked request routed free → ~$3 saved (1M input @ $3/1M).
	_, _ = db.LogRequest(&Request{CreatedAt: now, User: "alice", ModelAsked: "claude-sonnet-4-6",
		Provider: "groq", InputTokens: 1_000_000, OutputTokens: 0, CostUSD: 0, Status: 200})
	// bob: a tiny Haiku request → negligible savings.
	_, _ = db.LogRequest(&Request{CreatedAt: now, User: "bob", ModelAsked: "claude-haiku-4-5",
		Provider: "groq", InputTokens: 1000, OutputTokens: 0, CostUSD: 0, Status: 200})
	// unattributed
	_, _ = db.LogRequest(&Request{CreatedAt: now, ModelAsked: "claude-haiku-4-5", Provider: "groq", Status: 200})

	lb, err := db.GetLeaderboard("today")
	if err != nil {
		t.Fatal(err)
	}
	if len(lb) < 2 {
		t.Fatalf("expected ≥2 entries, got %d", len(lb))
	}
	if lb[0].User != "alice" {
		t.Errorf("alice saved the most and should rank first, got %q", lb[0].User)
	}
	if lb[0].SavedUSD < 2.9 {
		t.Errorf("alice's savings should be ~$3, got %v", lb[0].SavedUSD)
	}
	if lb[0].Requests != 1 {
		t.Errorf("alice should have 1 request, got %d", lb[0].Requests)
	}
}
