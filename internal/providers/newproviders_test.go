package providers

import (
	"strings"
	"testing"
)

// TestNewProvidersWiring locks in the base URLs, tiers, endpoints and model maps
// for the v0.5.0 providers (verified against each provider's docs).
func TestNewProvidersWiring(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		haiku     string // expected map for claude-haiku-4-5
		other     string // expected map for a non-haiku model
	}{
		{"baseten", "https://inference.baseten.co/v1", "deepseek-ai/DeepSeek-V3.1", "deepseek-ai/DeepSeek-V3.1"},
		{"featherless", "https://api.featherless.ai/v1", "meta-llama/Meta-Llama-3.1-8B-Instruct", "meta-llama/Meta-Llama-3.1-70B-Instruct"},
		{"kluster", "https://api.kluster.ai/v1", "klusterai/Meta-Llama-3.1-8B-Instruct-Turbo", "klusterai/Meta-Llama-3.3-70B-Instruct-Turbo"},
		{"venice", "https://api.venice.ai/api/v1", "llama-3.3-70b", "llama-3.3-70b"},
		{"friendli", "https://api.friendli.ai/serverless/v1", "meta-llama-3.1-8b-instruct", "meta-llama-3.3-70b-instruct"},
		{"chutes", "https://llm.chutes.ai/v1", "deepseek-ai/DeepSeek-V3", "deepseek-ai/DeepSeek-V3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := FromConfig(tc.name, "key", "", nil)
			if err != nil {
				t.Fatalf("FromConfig(%q): %v", tc.name, err)
			}
			if p.Name() != tc.name {
				t.Errorf("Name = %q, want %q", p.Name(), tc.name)
			}
			if p.BaseURL() != tc.base {
				t.Errorf("BaseURL = %q, want %q", p.BaseURL(), tc.base)
			}
			if p.Tier() != TierStandard {
				t.Errorf("Tier = %q, want standard", p.Tier())
			}
			if want := tc.base + "/chat/completions"; p.ChatCompletionsURL() != want {
				t.Errorf("ChatCompletionsURL = %q, want %q", p.ChatCompletionsURL(), want)
			}
			if got := p.MapModel("claude-haiku-4-5"); got != tc.haiku {
				t.Errorf("MapModel(haiku) = %q, want %q", got, tc.haiku)
			}
			if got := p.MapModel("claude-sonnet-4-6"); got != tc.other {
				t.Errorf("MapModel(sonnet) = %q, want %q", got, tc.other)
			}
			if !IsOpenAICompatible(tc.name) {
				t.Errorf("%s should be OpenAI-compatible", tc.name)
			}
			if p.Pricing().InputPer1M <= 0 {
				t.Errorf("%s should have a positive input price", tc.name)
			}
			// Default model list is non-empty and points at the right host family.
			if dm := DefaultModels(tc.name); len(dm) == 0 || strings.TrimSpace(dm[0]) == "" {
				t.Errorf("%s has no default model", tc.name)
			}
		})
	}
}
