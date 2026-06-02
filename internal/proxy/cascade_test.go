package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestVerifyResponse(t *testing.T) {
	good := `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"here you go"}}]}`
	empty := `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":""}}]}`
	toolOK := `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"t","type":"function","function":{"name":"f","arguments":"{\"a\":1}"}}]}}]}`
	toolBad := `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"t","type":"function","function":{"name":"f","arguments":"{not json"}}]}}]}`

	if !verifyResponse("groq", []byte(good)) {
		t.Error("non-empty content should verify")
	}
	if verifyResponse("groq", []byte(empty)) {
		t.Error("empty content + no tools should fail")
	}
	if !verifyResponse("groq", []byte(toolOK)) {
		t.Error("valid tool call should verify")
	}
	if verifyResponse("groq", []byte(toolBad)) {
		t.Error("malformed tool args should fail")
	}

	// Anthropic-format
	if !verifyResponse("anthropic", []byte(`{"content":[{"type":"text","text":"ok"}]}`)) {
		t.Error("anthropic text should verify")
	}
	if verifyResponse("anthropic", []byte(`{"content":[]}`)) {
		t.Error("empty anthropic content should fail")
	}
}

// countingOAIServer serves a fixed content and counts hits.
func countingOAIServer(content string, hits *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": "m",
			"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": content}}},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
}

func TestCascadeEscalatesOnWeakOutput(t *testing.T) {
	var cheapHits, strongHits int32
	cheap := countingOAIServer("", &cheapHits)    // empty → fails verification
	strong := countingOAIServer("solid answer", &strongHits)
	defer cheap.Close()
	defer strong.Close()

	h := buildTestHandler(t, []testProv{{"cheap", "free", cheap.URL}, {"strong", "premium", strong.URL}})
	h.cascade = true

	rec := doMessages(h, `{"model":"claude-sonnet-4-6","max_tokens":100,"messages":[{"role":"user","content":"refactor this"}]}`)

	if cheapHits != 1 || strongHits != 1 {
		t.Fatalf("cheap=%d strong=%d; expected cascade to try cheap then escalate to strong", cheapHits, strongHits)
	}
	if got := rec.Header().Get("X-Nexus-Provider"); got != "strong" {
		t.Errorf("served by %q, expected the escalated 'strong' provider", got)
	}
	var resp AnthropicResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Content) == 0 || resp.Content[0].Text != "solid answer" {
		t.Errorf("expected the strong provider's answer, got %+v", resp.Content)
	}
}

func TestCascadeAcceptsCheapWhenGood(t *testing.T) {
	var cheapHits, strongHits int32
	cheap := countingOAIServer("cheap is fine", &cheapHits)
	strong := countingOAIServer("expensive", &strongHits)
	defer cheap.Close()
	defer strong.Close()

	h := buildTestHandler(t, []testProv{{"cheap", "free", cheap.URL}, {"strong", "premium", strong.URL}})
	h.cascade = true

	rec := doMessages(h, `{"model":"claude-sonnet-4-6","max_tokens":100,"messages":[{"role":"user","content":"refactor this"}]}`)

	if cheapHits != 1 || strongHits != 0 {
		t.Fatalf("cheap=%d strong=%d; a good cheap answer must NOT escalate", cheapHits, strongHits)
	}
	if got := rec.Header().Get("X-Nexus-Provider"); got != "cheap" {
		t.Errorf("served by %q, expected the cheap provider", got)
	}
}
