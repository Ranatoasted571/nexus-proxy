package storage

import (
	"testing"
	"time"
)

func TestCacheTokensPersistAndAggregate(t *testing.T) {
	db := newTestDB(t)
	_, err := db.LogRequest(&Request{
		CreatedAt:        time.Now(),
		ModelAsked:       "claude-sonnet-4-6",
		ModelUsed:        "deepseek-chat",
		Provider:         "deepseek",
		Complexity:       "standard",
		InputTokens:      100,
		OutputTokens:     50,
		CacheReadTokens:  4000,
		CacheWriteTokens: 300,
		CostUSD:          0.01,
		CacheSavedUSD:    0.05,
		LatencyMS:        80,
		Status:           200,
	})
	if err != nil {
		t.Fatal(err)
	}

	st, err := db.GetStats("today")
	if err != nil {
		t.Fatal(err)
	}
	if st.CacheReadTokens != 4000 || st.CacheWriteTokens != 300 {
		t.Errorf("cache tokens not aggregated: %+v", st)
	}
	if st.CacheSavedUSD != 0.05 {
		t.Errorf("cache saved = %v, want 0.05", st.CacheSavedUSD)
	}
}

func TestSavingsCountsCachedTokensInBaseline(t *testing.T) {
	db := newTestDB(t)
	// 1M cache-read tokens on Sonnet baseline (3/1M input) should add ~$3 to the
	// baseline even though actual cost is tiny — proving cached tokens are counted.
	if _, err := db.LogRequest(&Request{
		CreatedAt:       time.Now(),
		ModelAsked:      "claude-sonnet-4-6",
		ModelUsed:       "deepseek-chat",
		Provider:        "deepseek",
		Complexity:      "standard",
		InputTokens:     0,
		OutputTokens:    0,
		CacheReadTokens: 1_000_000,
		CostUSD:         0.02,
		CacheSavedUSD:   0.24,
		Status:          200,
	}); err != nil {
		t.Fatal(err)
	}

	s, err := db.GetSavings("today")
	if err != nil {
		t.Fatal(err)
	}
	if s.BaselineUSD < 2.99 || s.BaselineUSD > 3.01 {
		t.Errorf("baseline should ~= 3.0 (1M input @ $3/1M), got %v", s.BaselineUSD)
	}
	if s.CacheSavedUSD != 0.24 {
		t.Errorf("cache saved = %v, want 0.24", s.CacheSavedUSD)
	}
	if s.SavedUSD <= 0 {
		t.Errorf("expected positive savings, got %v", s.SavedUSD)
	}
}
