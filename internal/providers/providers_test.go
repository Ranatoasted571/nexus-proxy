package providers

import (
	"net/http"
	"strings"
	"testing"
)

func TestFromConfigKnownAndUnknown(t *testing.T) {
	known := []string{
		"anthropic", "openai", "deepseek", "groq", "gemini", "mistral", "together",
		"openrouter", "cohere", "xai", "fireworks", "perplexity", "deepinfra",
		"cerebras", "sambanova", "novita", "hyperbolic", "nebius", "nvidia",
		"moonshot", "zhipu", "ai21", "lambda", "baseten", "featherless", "kluster",
		"venice", "friendli", "chutes", "ollama",
	}
	for _, name := range known {
		p, err := FromConfig(name, "key", "", nil)
		if err != nil {
			t.Errorf("%s: unexpected error %v", name, err)
			continue
		}
		if p.Name() != name {
			t.Errorf("name mismatch: %s vs %s", p.Name(), name)
		}
		if p.ChatCompletionsURL() == "" && name != "anthropic" {
			t.Errorf("%s: expected a chat completions URL", name)
		}
	}
	if _, err := FromConfig("bogus", "", "", nil); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestOpenAICompatibility(t *testing.T) {
	if IsOpenAICompatible("anthropic") {
		t.Error("anthropic should NOT be OpenAI-compatible")
	}
	for _, n := range []string{"groq", "deepseek", "gemini", "ollama"} {
		if !IsOpenAICompatible(n) {
			t.Errorf("%s should be OpenAI-compatible", n)
		}
	}
}

func TestChatCompletionsURL(t *testing.T) {
	if got := NewAnthropic("k").ChatCompletionsURL(); got != "" {
		t.Errorf("anthropic chat URL should be empty, got %q", got)
	}
	if got := NewGroq("k").ChatCompletionsURL(); got == "" {
		t.Error("groq chat URL should be set")
	}
	if got := NewOllama("", "").ChatCompletionsURL(); got != "http://localhost:11434/v1/chat/completions" {
		t.Errorf("ollama chat URL = %q", got)
	}
}

func TestCalculateCost(t *testing.T) {
	p := PricingInfo{InputPer1M: 3.0, OutputPer1M: 15.0}
	if got := p.CalculateCost(1_000_000, 1_000_000); got != 18.0 {
		t.Errorf("cost = %v, want 18", got)
	}
	if got := p.CalculateCost(0, 0); got != 0 {
		t.Errorf("zero cost = %v, want 0", got)
	}
	free := PricingInfo{}
	if got := free.CalculateCost(5000, 5000); got != 0 {
		t.Errorf("free cost = %v, want 0", got)
	}
}

func TestDefaultTier(t *testing.T) {
	cases := map[string]string{
		"groq": TierFree, "gemini": TierFree, "deepseek": TierStandard,
		"anthropic": TierPremium, "ollama": TierLocal,
	}
	for name, want := range cases {
		if got := DefaultTier(name); got != want {
			t.Errorf("DefaultTier(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestMapModel(t *testing.T) {
	if got := NewGroq("k").MapModel("claude-haiku-4-5"); got != "llama-3.1-8b-instant" {
		t.Errorf("groq haiku mapping = %q", got)
	}
	if got := NewAnthropic("k").MapModel("claude-opus-4-5"); got != "claude-opus-4-5" {
		t.Errorf("anthropic should pass through, got %q", got)
	}
}

func TestGenericProvider(t *testing.T) {
	p, err := New(Spec{
		Name: "myllm", Type: "openai-compatible", BaseURL: "https://llm.local/v1/",
		APIKey: "k", Tier: "free", Models: []string{"my-model"},
		ModelMap: map[string]string{"claude-haiku-4-5": "tiny"}, InputPer1M: 1, OutputPer1M: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "myllm" || p.Tier() != "free" {
		t.Errorf("name/tier wrong: %s/%s", p.Name(), p.Tier())
	}
	if p.ChatCompletionsURL() != "https://llm.local/v1/chat/completions" {
		t.Errorf("chat URL = %q", p.ChatCompletionsURL())
	}
	if p.MapModel("claude-haiku-4-5") != "tiny" {
		t.Errorf("model_map not applied: %q", p.MapModel("claude-haiku-4-5"))
	}
	if p.MapModel("claude-opus-4-5") != "my-model" {
		t.Errorf("default model fallback failed: %q", p.MapModel("claude-opus-4-5"))
	}
	if c := p.Pricing().CalculateCost(1_000_000, 1_000_000); c != 3 {
		t.Errorf("cost = %v, want 3", c)
	}
}

func TestGenericRequiresBaseURL(t *testing.T) {
	if _, err := New(Spec{Name: "x", Type: "custom"}); err == nil {
		t.Error("expected error when base_url missing for a custom provider")
	}
}

func TestBuiltinModelMapOverride(t *testing.T) {
	p, err := New(Spec{Name: "groq", ModelMap: map[string]string{"claude-sonnet-4-6": "custom-model"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.MapModel("claude-sonnet-4-6") != "custom-model" {
		t.Errorf("override not applied: %q", p.MapModel("claude-sonnet-4-6"))
	}
	if p.MapModel("claude-haiku-4-5") != "llama-3.1-8b-instant" {
		t.Errorf("built-in default lost: %q", p.MapModel("claude-haiku-4-5"))
	}
	if p.Name() != "groq" {
		t.Errorf("name = %q", p.Name())
	}
}

func TestPricingOverride(t *testing.T) {
	p, _ := New(Spec{Name: "groq", InputPer1M: 5, OutputPer1M: 7})
	if pr := p.Pricing(); pr.InputPer1M != 5 || pr.OutputPer1M != 7 {
		t.Errorf("pricing override failed: %+v", pr)
	}
}

func TestAzureProvider(t *testing.T) {
	p, err := New(Spec{Type: "azure", Name: "azure", BaseURL: "https://x.openai.azure.com/openai/deployments/gpt4o", APIVersion: "2024-10-21"})
	if err != nil {
		t.Fatal(err)
	}
	if !IsOpenAICompatible("azure") {
		t.Error("azure should be OpenAI-compatible")
	}
	if !strings.Contains(p.ChatCompletionsURL(), "api-version=2024-10-21") {
		t.Errorf("azure chat URL = %s", p.ChatCompletionsURL())
	}
	req, _ := http.NewRequest("POST", "http://x", nil)
	p.(Authorizer).Authorize(req, nil, "secret")
	if req.Header.Get("api-key") != "secret" {
		t.Error("azure should set the api-key header")
	}
	if _, err := New(Spec{Type: "azure", Name: "a"}); err == nil {
		t.Error("azure without base_url should error")
	}
}

func TestVertexProvider(t *testing.T) {
	p, err := New(Spec{Type: "vertex", Name: "vertex", Region: "us-east5", Project: "proj", APIKey: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if IsOpenAICompatible("vertex") {
		t.Error("vertex should be Anthropic-format")
	}
	an, ok := p.(AnthropicNative)
	if !ok {
		t.Fatal("vertex should implement AnthropicNative")
	}
	if !strings.Contains(an.MessagesURL("claude-sonnet-4-6"), "rawPredict") {
		t.Errorf("vertex URL = %s", an.MessagesURL("claude-sonnet-4-6"))
	}
	body := string(an.PrepareBody([]byte(`{"model":"claude","stream":true,"max_tokens":5}`), "claude-sonnet-4-6"))
	if strings.Contains(body, `"model"`) || strings.Contains(body, `"stream"`) {
		t.Errorf("vertex body should strip model/stream: %s", body)
	}
	if !strings.Contains(body, "vertex-2023-10-16") {
		t.Errorf("vertex body missing anthropic_version: %s", body)
	}
	if _, err := New(Spec{Type: "vertex", Name: "v"}); err == nil {
		t.Error("vertex without region/project should error")
	}
}

func TestBedrockProvider(t *testing.T) {
	p, err := New(Spec{Type: "bedrock", Name: "bedrock", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	bn := p.(AnthropicNative)
	if !strings.Contains(bn.MessagesURL("claude-haiku-4-5"), "/invoke") {
		t.Errorf("bedrock URL = %s", bn.MessagesURL("claude-haiku-4-5"))
	}
	if !strings.Contains(string(bn.PrepareBody([]byte(`{"model":"x","max_tokens":5}`), "claude-haiku-4-5")), "bedrock-2023-05-31") {
		t.Error("bedrock body missing anthropic_version")
	}
	if _, err := New(Spec{Type: "bedrock", Name: "b"}); err == nil {
		t.Error("bedrock without region should error")
	}
}
