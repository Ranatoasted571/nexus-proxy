package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/router"
)

func chatCompletions(h *Handler, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	h.HandleChatCompletions(rec, req)
	return rec
}

func TestGateway_OpenAIPassthrough(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]interface{}
		json.NewDecoder(r.Body).Decode(&m)
		gotModel, _ = m["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": gotModel,
			"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": "hello from gateway"}}},
			"usage": map[string]interface{}{"prompt_tokens": 9, "completion_tokens": 4},
		})
	}))
	defer srv.Close()
	impl, err := providers.New(providers.Spec{Name: "groqish", Type: "openai-compatible", BaseURL: srv.URL, Tier: "free", Models: []string{"llama-x"}})
	if err != nil {
		t.Fatal(err)
	}
	rt := router.New(router.StrategyAuto)
	rt.AddProvider(&router.Provider{Name: "groqish", Tier: "free", Healthy: true})
	h := &Handler{httpClient: &http.Client{Timeout: 10 * time.Second}, router: rt, providers: map[string]*activeProvider{"groqish": {impl: impl, apiKey: "k"}}}

	rec := chatCompletions(h, `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("X-Nexus-Provider") != "groqish" {
		t.Errorf("provider header = %q", rec.Header().Get("X-Nexus-Provider"))
	}
	if gotModel != "llama-x" {
		t.Errorf("gateway should swap to the provider's model (llama-x), upstream saw %q", gotModel)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("not OpenAI JSON: %v", err)
	}
	choices, _ := resp["choices"].([]interface{})
	if len(choices) == 0 {
		t.Fatalf("no choices in %s", rec.Body.String())
	}
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg["content"] != "hello from gateway" {
		t.Errorf("content = %v", msg["content"])
	}
}

func TestGateway_StreamPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"hi"},"index":0}]}`+"\n\n")
		fl.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()
	h := buildTestHandler(t, []testProv{{"groqish", "free", srv.URL}})

	rec := chatCompletions(h, `{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"hi"}]}`)

	out := rec.Body.String()
	if !strings.Contains(out, `"content":"hi"`) || !strings.Contains(out, "[DONE]") {
		t.Errorf("stream not passed through:\n%s", out)
	}
}

func TestGateway_AnthropicProviderToOpenAI(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "x")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "y")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(anthropicResponseJSON("claude-says-hi")))
	}))
	defer srv.Close()
	impl, err := providers.New(providers.Spec{Type: "bedrock", Name: "bedrock", Region: "us-east-1", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	h := anthropicNativeHandler(impl, "")

	rec := chatCompletions(h, `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("not OpenAI JSON: %v\n%s", err, rec.Body.String())
	}
	if resp["object"] != "chat.completion" {
		t.Errorf("object = %v, want chat.completion", resp["object"])
	}
	choices := resp["choices"].([]interface{})
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg["content"] != "claude-says-hi" {
		t.Errorf("converted content = %v", msg["content"])
	}
}

func TestGateway_Models(t *testing.T) {
	h := buildTestHandler(t, nil)
	rec := httptest.NewRecorder()
	h.HandleModels(rec, httptest.NewRequest("GET", "/v1/models", nil))
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["object"] != "list" {
		t.Errorf("object = %v, want list", resp["object"])
	}
	if data, _ := resp["data"].([]interface{}); len(data) == 0 {
		t.Error("models list is empty")
	}
}

func TestTransformOpenAIToAnthropic(t *testing.T) {
	o := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "be brief"},
			{Role: "user", Content: "hi"},
		},
	}
	a := TransformOpenAIToAnthropic(o)
	if a.System != "be brief" {
		t.Errorf("system = %v", a.System)
	}
	if len(a.Messages) != 1 || a.Messages[0].Role != "user" {
		t.Errorf("messages = %+v (system should be lifted out)", a.Messages)
	}
	if a.MaxTokens == 0 {
		t.Error("max_tokens should default to non-zero for Anthropic")
	}
}
