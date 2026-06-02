package storage

import (
	"testing"
	"time"
)

func TestRedactedAggregates(t *testing.T) {
	db := newTestDB(t)
	now := time.Now()
	_, _ = db.LogRequest(&Request{CreatedAt: now, Provider: "groq", ModelAsked: "m", ModelUsed: "m", Complexity: "simple", Redacted: 3, Status: 200})
	_, _ = db.LogRequest(&Request{CreatedAt: now, Provider: "groq", ModelAsked: "m", ModelUsed: "m", Complexity: "simple", Redacted: 2, Status: 200})
	_, _ = db.LogRequest(&Request{CreatedAt: now, Provider: "groq", ModelAsked: "m", ModelUsed: "m", Complexity: "simple", Redacted: 0, Status: 200})

	st, err := db.GetStats("today")
	if err != nil {
		t.Fatal(err)
	}
	if st.RedactedTotal != 5 {
		t.Errorf("RedactedTotal = %d, want 5", st.RedactedTotal)
	}
}
