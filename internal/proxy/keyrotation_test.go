package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
)

func TestPickKeyRotation(t *testing.T) {
	a := &activeProvider{keys: []string{"a", "b", "c"}, cool: make([]time.Time, 3)}
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		k, _ := a.pickKey()
		seen[k] = true
	}
	if len(seen) != 3 {
		t.Fatalf("round-robin should touch all 3 keys, saw %v", seen)
	}
}

func TestPickKeySkipsCooling(t *testing.T) {
	a := &activeProvider{keys: []string{"a", "b"}, cool: make([]time.Time, 2)}
	a.penalize(0, time.Hour) // "a" is cooling
	for i := 0; i < 5; i++ {
		k, idx := a.pickKey()
		if k == "a" || idx == 0 {
			t.Fatalf("cooling key must be skipped, got %q (idx %d)", k, idx)
		}
	}
}

func TestPickKeyFallbackToApiKey(t *testing.T) {
	a := &activeProvider{apiKey: "solo"}
	if k, idx := a.pickKey(); k != "solo" || idx != -1 {
		t.Fatalf("no pool should fall back to apiKey, got %q %d", k, idx)
	}
}

func TestResolveProviderKeys(t *testing.T) {
	if ks := resolveProviderKeys(config.Provider{APIKeys: []string{"a", "b"}, APIKey: "ignored"}); len(ks) != 2 || ks[0] != "a" {
		t.Fatalf("api_keys should win: %v", ks)
	}
	if ks := resolveProviderKeys(config.Provider{APIKey: "solo"}); len(ks) != 1 || ks[0] != "solo" {
		t.Fatalf("single key: %v", ks)
	}
	t.Setenv("NEXUS_TEST_KEY", "envval")
	if ks := resolveProviderKeys(config.Provider{APIKey: "env:NEXUS_TEST_KEY"}); ks[0] != "envval" {
		t.Fatalf("env resolution: %v", ks)
	}
}

func TestKeyRotationOn429(t *testing.T) {
	var badHits, goodHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer good" {
			atomic.AddInt32(&goodHits, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "x", "object": "chat.completion", "model": "m",
				"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
					"message": map[string]interface{}{"role": "assistant", "content": "ok"}}},
				"usage": map[string]interface{}{"prompt_tokens": 5, "completion_tokens": 2},
			})
			return
		}
		atomic.AddInt32(&badHits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	h := buildTestHandler(t, []testProv{{"p", "free", srv.URL}})
	ap := h.providers["p"]
	ap.keys = []string{"bad", "good"}
	ap.cool = make([]time.Time, 2)
	ap.rr = 1 // force the first pick to be index 0 ("bad")

	rec := doMessages(h, `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"hi"}]}`)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if badHits < 1 || goodHits != 1 {
		t.Fatalf("bad=%d good=%d; expected a 429 on the bad key then rotation to the good key", badHits, goodHits)
	}
}
