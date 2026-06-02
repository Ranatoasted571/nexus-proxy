package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactDetectsSecrets(t *testing.T) {
	r := &redactor{}
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"key sk-ant-api03ABCDEFGHIJKLMNOPQRST mail bob@acme.io aws AKIAIOSFODNN7EXAMPLE"}]}`)
	out, m := r.redact(body)
	s := string(out)
	for _, secret := range []string{"sk-ant-api03ABCDEFGHIJKLMNOPQRST", "bob@acme.io", "AKIAIOSFODNN7EXAMPLE"} {
		if strings.Contains(s, secret) {
			t.Errorf("redacted body still contains %q", secret)
		}
	}
	if len(m) < 3 {
		t.Errorf("expected ≥3 redactions, got %d: %v", len(m), m)
	}
}

func TestRedactNoFalsePositiveOnPlainCode(t *testing.T) {
	r := &redactor{}
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"func add(a, b int) int { return a + b }"}]}`)
	if _, m := r.redact(body); len(m) != 0 {
		t.Errorf("plain code should not be redacted, got %v", m)
	}
}

func TestRestoringWriterSplitPlaceholder(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newRestoringWriter(rec, map[string]string{"[NX_REDACTED_APIKEY_1]": "sk-ant-secret"})
	// placeholder split across two writes
	_, _ = rw.Write([]byte("before [NX_RED"))
	_, _ = rw.Write([]byte("ACTED_APIKEY_1] after"))
	rw.flush()
	if got := rec.Body.String(); got != "before sk-ant-secret after" {
		t.Errorf("split-placeholder restore failed: %q", got)
	}
}

// TestFirewallMaskOutboundAndRestore is the full round-trip: the provider must
// never see the secret, but the client must get it back.
func TestFirewallMaskOutboundAndRestore(t *testing.T) {
	var providerSaw string
	// mock echoes the (redacted) user content back as the assistant message
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		providerSaw = string(raw)
		var oreq struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &oreq)
		content := ""
		for _, m := range oreq.Messages {
			if m.Role == "user" {
				content = m.Content
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "x", "object": "chat.completion", "model": "m",
			"choices": []interface{}{map[string]interface{}{"index": 0, "finish_reason": "stop",
				"message": map[string]interface{}{"role": "assistant", "content": content}}},
			"usage": map[string]interface{}{"prompt_tokens": 5, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	h := buildTestHandler(t, []testProv{{"p", "free", srv.URL}})
	h.firewall = &redactor{}

	secret := "sk-ant-api03ZZZZYYYYXXXXWWWWVVVV"
	body := `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"deploy with ` + secret + `"}]}`
	rec := doMessages(h, body)

	if strings.Contains(providerSaw, secret) {
		t.Errorf("SECRET LEAKED to provider:\n%s", providerSaw)
	}
	if !strings.Contains(providerSaw, "[NX_REDACTED_APIKEY_1]") {
		t.Errorf("provider should have seen a placeholder, saw:\n%s", providerSaw)
	}
	var resp AnthropicResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Content) == 0 || !strings.Contains(resp.Content[0].Text, secret) {
		t.Errorf("client response should have the secret restored, got %+v", resp.Content)
	}
}
