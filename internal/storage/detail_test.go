package storage

import (
	"strings"
	"testing"
	"time"
)

func TestGetRequestDetailRoundtrip(t *testing.T) {
	db := newTestDB(t)
	id, err := db.LogRequest(&Request{
		CreatedAt:  time.Now(),
		ModelAsked: "claude-sonnet-4-6",
		ModelUsed:  "deepseek-chat",
		Provider:   "deepseek",
		Complexity: "standard",
		InputTokens: 10, OutputTokens: 5, CostUSD: 0.001, Status: 200,
		Prompt:   `{"messages":[{"role":"user","content":"hello"}]}`,
		Response: `{"content":[{"type":"text","text":"world"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := db.GetRequestDetail(id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.Prompt, "hello") {
		t.Errorf("prompt not stored: %q", d.Prompt)
	}
	if !strings.Contains(d.Response, "world") {
		t.Errorf("response not stored: %q", d.Response)
	}
	if d.Provider != "deepseek" {
		t.Errorf("provider = %q", d.Provider)
	}
}
