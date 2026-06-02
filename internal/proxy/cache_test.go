package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCacheKeyNormalized(t *testing.T) {
	a := cacheKey("m", []byte(`{"model":"x","stream":false}`))
	b := cacheKey("m", []byte("{ \"stream\":  false , \"model\": \"x\" }"))
	if a != b {
		t.Errorf("normalized cache keys should match despite formatting:\n%s\n%s", a, b)
	}
	if c := cacheKey("c", []byte(`{"model":"x","stream":false}`)); c == a {
		t.Error("different endpoint prefixes must not collide")
	}
}

func TestCacheHitOnIdenticalRequest(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": "m",
			"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": "cached-me"}}},
			"usage": map[string]interface{}{"prompt_tokens": 5, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	h := buildTestHandler(t, []testProv{{"p", "free", srv.URL}})
	h.cache = newResponseCache(time.Minute, 100, false, 0)

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`
	rec1 := chatCompletions(h, body)
	rec2 := chatCompletions(h, body)

	if calls != 1 {
		t.Errorf("upstream called %d times; the 2nd identical request must be served from cache (want 1)", calls)
	}
	if rec1.Header().Get("X-Nexus-Cache") == "HIT" {
		t.Error("first request should be a cache MISS")
	}
	if rec2.Header().Get("X-Nexus-Cache") != "HIT" {
		t.Errorf("second request should be a cache HIT, got %q", rec2.Header().Get("X-Nexus-Cache"))
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Error("cached response body should match the original")
	}
}
