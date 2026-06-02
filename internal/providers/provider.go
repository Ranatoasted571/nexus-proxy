package providers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Provider defines the interface all LLM providers must implement
type Provider interface {
	Name() string
	BaseURL() string
	Tier() string
	MapModel(claudeModel string) string
	Pricing() PricingInfo
	HealthCheck() error
	// ChatCompletionsURL returns the OpenAI-compatible chat completions endpoint
	// for this provider. It is empty for native-Anthropic providers.
	ChatCompletionsURL() string
}

// PricingInfo holds cost per million tokens
type PricingInfo struct {
	InputPer1M  float64 // USD
	OutputPer1M float64 // USD
}

// CalculateCost returns the cost for a given token count
func (p PricingInfo) CalculateCost(inputTokens, outputTokens int) float64 {
	return (float64(inputTokens)/1_000_000)*p.InputPer1M +
		(float64(outputTokens)/1_000_000)*p.OutputPer1M
}

// ─── Model Mapping ─────────────────────────────────────────────────────────

// StandardModelMap defines the default Claude → provider model mapping
// Override per provider as needed
var StandardModelMap = map[string]string{
	"claude-opus-4-5":   "big", // provider-specific, override
	"claude-opus-4-6":   "big",
	"claude-sonnet-4-6": "medium",
	"claude-haiku-4-5":  "small",
}

// Tier constants
const (
	TierFree     = "free"
	TierStandard = "standard"
	TierPremium  = "premium"
	TierLocal    = "local"
)

// ─── Construction & metadata ───────────────────────────────────────────────

// IsOpenAICompatible reports whether a provider speaks the OpenAI chat API.
// Anthropic-format providers (Anthropic, Bedrock, Vertex) return false.
func IsOpenAICompatible(name string) bool {
	switch strings.ToLower(name) {
	case "anthropic", "bedrock", "vertex":
		return false
	}
	return true
}

// FromConfig builds a concrete Provider from config values.
func FromConfig(name, apiKey, baseURL string, models []string) (Provider, error) {
	model := ""
	if len(models) > 0 {
		model = models[0]
	}
	switch strings.ToLower(name) {
	case "anthropic":
		return NewAnthropic(apiKey), nil
	case "openai":
		return NewOpenAI(apiKey), nil
	case "deepseek":
		return NewDeepSeek(apiKey), nil
	case "groq":
		return NewGroq(apiKey), nil
	case "gemini":
		return NewGemini(apiKey), nil
	case "mistral":
		return NewMistral(apiKey), nil
	case "together":
		return NewTogether(apiKey), nil
	case "openrouter":
		return NewOpenRouter(apiKey), nil
	case "cohere":
		return NewCohere(apiKey), nil
	case "xai":
		return NewXAI(apiKey), nil
	case "fireworks":
		return NewFireworks(apiKey), nil
	case "perplexity":
		return NewPerplexity(apiKey), nil
	case "deepinfra":
		return NewDeepInfra(apiKey), nil
	case "cerebras":
		return NewCerebras(apiKey), nil
	case "sambanova":
		return NewSambaNova(apiKey), nil
	case "novita":
		return NewNovita(apiKey), nil
	case "hyperbolic":
		return NewHyperbolic(apiKey), nil
	case "nebius":
		return NewNebius(apiKey), nil
	case "nvidia":
		return NewNVIDIA(apiKey), nil
	case "moonshot":
		return NewMoonshot(apiKey), nil
	case "zhipu":
		return NewZhipu(apiKey), nil
	case "ai21":
		return NewAI21(apiKey), nil
	case "lambda":
		return NewLambda(apiKey), nil
	case "ollama":
		return NewOllama(baseURL, model), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

// DefaultTier returns the default tier for a known provider name.
func DefaultTier(name string) string {
	switch strings.ToLower(name) {
	case "anthropic", "openai", "xai":
		return TierPremium
	case "deepseek", "mistral", "together", "openrouter", "cohere", "fireworks", "perplexity", "deepinfra", "novita", "hyperbolic", "nebius", "moonshot", "zhipu", "ai21", "lambda":
		return TierStandard
	case "groq", "gemini", "cerebras", "sambanova", "nvidia":
		return TierFree
	case "ollama":
		return TierLocal
	default:
		return TierStandard
	}
}

// DefaultModels returns sensible default model names for a known provider.
func DefaultModels(name string) []string {
	switch strings.ToLower(name) {
	case "anthropic":
		return []string{"claude-opus-4-5", "claude-sonnet-4-6", "claude-haiku-4-5"}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini"}
	case "deepseek":
		return []string{"deepseek-chat"}
	case "groq":
		return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant"}
	case "gemini":
		return []string{"gemini-2.0-flash"}
	case "mistral":
		return []string{"mistral-large-latest", "mistral-small-latest"}
	case "together":
		return []string{"meta-llama/Llama-3.3-70B-Instruct-Turbo"}
	case "openrouter":
		return []string{"meta-llama/llama-3.3-70b-instruct"}
	case "cohere":
		return []string{"command-r-plus", "command-r"}
	case "xai":
		return []string{"grok-2-latest"}
	case "fireworks":
		return []string{"accounts/fireworks/models/llama-v3p3-70b-instruct"}
	case "perplexity":
		return []string{"sonar", "sonar-pro"}
	case "deepinfra":
		return []string{"meta-llama/Llama-3.3-70B-Instruct"}
	case "cerebras":
		return []string{"llama-3.3-70b", "llama3.1-8b"}
	case "sambanova":
		return []string{"Meta-Llama-3.3-70B-Instruct"}
	case "novita":
		return []string{"meta-llama/llama-3.3-70b-instruct"}
	case "hyperbolic":
		return []string{"meta-llama/Llama-3.3-70B-Instruct"}
	case "nebius":
		return []string{"meta-llama/Llama-3.3-70B-Instruct"}
	case "nvidia":
		return []string{"meta/llama-3.3-70b-instruct"}
	case "moonshot":
		return []string{"moonshot-v1-32k", "moonshot-v1-8k"}
	case "zhipu":
		return []string{"glm-4-plus", "glm-4-flash"}
	case "ai21":
		return []string{"jamba-large-1.7", "jamba-mini-1.7"}
	case "lambda":
		return []string{"llama-3.3-70b-instruct-fp8"}
	case "ollama":
		return []string{"codellama:13b"}
	default:
		return nil
	}
}

// ─── Shared helpers ────────────────────────────────────────────────────────

// bearerHealthCheck does a GET against an OpenAI-style /models endpoint with a
// Bearer token and reports an error on any non-2xx/3xx response.
func bearerHealthCheck(url, apiKey string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}

// reachableHealthCheck treats any response below 500 as healthy (used by
// providers without a public, auth-free models endpoint).
func reachableHealthCheck(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("unreachable: %d", resp.StatusCode)
	}
	return nil
}
