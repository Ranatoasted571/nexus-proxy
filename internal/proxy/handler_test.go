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

// ─── test helpers ───────────────────────────────────────────────────────────

type testProv struct {
	name, tier, url string
}

// buildTestHandler wires a Handler with custom OpenAI-compatible providers
// pointing at in-process httptest servers — exercising the real handler path.
func buildTestHandler(t *testing.T, provs []testProv) *Handler {
	t.Helper()
	rt := router.New(router.StrategyAuto)
	active := map[string]*activeProvider{}
	for _, p := range provs {
		impl, err := providers.New(providers.Spec{Name: p.name, Type: "openai-compatible", BaseURL: p.url, Tier: p.tier})
		if err != nil {
			t.Fatalf("build provider %s: %v", p.name, err)
		}
		active[p.name] = &activeProvider{impl: impl, apiKey: "k"}
		rt.AddProvider(&router.Provider{Name: p.name, Tier: p.tier, Healthy: true})
	}
	return &Handler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		router:     rt,
		providers:  active,
	}
}

func openAIServer(status int, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"err"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": "m",
			"choices": []interface{}{map[string]interface{}{
				"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": content},
			}},
			"usage": map[string]interface{}{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
}

func doMessages(h *Handler, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	h.HandleMessages(rec, req)
	return rec
}

// ─── tests ──────────────────────────────────────────────────────────────────

func TestHandleMessages_OpenAIProvider(t *testing.T) {
	srv := openAIServer(http.StatusOK, "Hello from provider")
	defer srv.Close()
	h := buildTestHandler(t, []testProv{{"test", "free", srv.URL}})

	rec := doMessages(h, `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"hi"}]}`)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("X-Nexus-Provider"); got != "test" {
		t.Errorf("X-Nexus-Provider = %q", got)
	}
	var resp AnthropicResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, rec.Body.String())
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "Hello from provider" {
		t.Errorf("content = %+v", resp.Content)
	}
	if resp.Model != "claude-haiku-4-5" {
		t.Errorf("model = %q (should echo the requested Claude model)", resp.Model)
	}
}

func TestHandleMessages_Streaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		for _, c := range []string{
			`{"choices":[{"delta":{"content":"Hello "},"finish_reason":null}]}`,
			`{"choices":[{"delta":{"content":"world"},"finish_reason":null}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			`{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":2}}`,
		} {
			fmt.Fprintf(w, "data: %s\n\n", c)
			fl.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()
	h := buildTestHandler(t, []testProv{{"test", "free", srv.URL}})

	rec := doMessages(h, `{"model":"claude-haiku-4-5","stream":true,"max_tokens":50,"messages":[{"role":"user","content":"hi"}]}`)

	out := rec.Body.String()
	for _, want := range []string{"event: message_start", "text_delta", "Hello ", "world", "event: message_stop"} {
		if !strings.Contains(out, want) {
			t.Errorf("stream missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestHandleMessages_Failover(t *testing.T) {
	bad := openAIServer(http.StatusServiceUnavailable, "")
	defer bad.Close()
	good := openAIServer(http.StatusOK, "from good")
	defer good.Close()
	// Both free → chain follows add order: bad first, then good.
	h := buildTestHandler(t, []testProv{{"bad", "free", bad.URL}, {"good", "free", good.URL}})

	rec := doMessages(h, `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"hi"}]}`)

	if rec.Code != 200 {
		t.Fatalf("status = %d (expected failover to good)", rec.Code)
	}
	if got := rec.Header().Get("X-Nexus-Provider"); got != "good" {
		t.Errorf("X-Nexus-Provider = %q, want good (failover)", got)
	}
	var resp AnthropicResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Content) == 0 || resp.Content[0].Text != "from good" {
		t.Errorf("content = %+v, want 'from good'", resp.Content)
	}
}

func TestHandleMessages_BudgetForcesFree(t *testing.T) {
	prem := openAIServer(http.StatusOK, "premium-reply")
	defer prem.Close()
	free := openAIServer(http.StatusOK, "free-reply")
	defer free.Close()
	h := buildTestHandler(t, []testProv{{"prem", "premium", prem.URL}, {"free", "free", free.URL}})
	h.budget = newBudgetTracker(0.01, 1.0) // already 1.0 spent → over the 0.01 cap

	// A complex prompt normally routes to premium first; over budget → free only.
	rec := doMessages(h, `{"model":"claude-sonnet-4-6","max_tokens":50,"messages":[{"role":"user","content":"Design the complete system architecture for a scalable platform"}]}`)

	if got := rec.Header().Get("X-Nexus-Provider"); got != "free" {
		t.Errorf("over budget should route to free, got provider %q", got)
	}
}

func TestHandleMessages_AzureProvider(t *testing.T) {
	var gotAPIKey, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("api-key")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": "azure-reply"}}},
			"usage": map[string]interface{}{"prompt_tokens": 3, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	impl, err := providers.New(providers.Spec{Type: "azure", Name: "azure", BaseURL: srv.URL, APIKey: "secret-key", APIVersion: "2024-10-21"})
	if err != nil {
		t.Fatal(err)
	}
	rt := router.New(router.StrategyAuto)
	rt.AddProvider(&router.Provider{Name: "azure", Tier: "premium", Healthy: true})
	h := &Handler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		router:     rt,
		providers:  map[string]*activeProvider{"azure": {impl: impl, apiKey: "secret-key"}},
	}

	rec := doMessages(h, `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"hi"}]}`)

	if gotAPIKey != "secret-key" {
		t.Errorf("Azure should authenticate via the api-key header, got %q", gotAPIKey)
	}
	if gotAuth != "" {
		t.Errorf("Azure should NOT send a Bearer Authorization header, got %q", gotAuth)
	}
	var resp AnthropicResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Content) == 0 || resp.Content[0].Text != "azure-reply" {
		t.Errorf("content = %+v, want 'azure-reply'", resp.Content)
	}
}

func TestHandleMessages_ZeroConfigBadRequest(t *testing.T) {
	h := buildTestHandler(t, nil) // no providers → zero-config (forwards to Anthropic)
	rec := doMessages(h, `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON should be 400, got %d", rec.Code)
	}
}
