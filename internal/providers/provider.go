// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

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

// PricingInfo holds cost per million tokens.
type PricingInfo struct {
	InputPer1M  float64 // USD per 1M input tokens
	OutputPer1M float64 // USD per 1M output tokens
	// Optional cache pricing. When zero, sensible defaults are derived from the
	// input price: cache reads at 0.1× input (Anthropic/DeepSeek-style) and cache
	// writes at 1.25× input (Anthropic-style).
	CacheReadPer1M  float64
	CacheWritePer1M float64
	// Optional off-peak pricing (e.g. DeepSeek's discount window). When
	// OffPeakStartUTC != OffPeakEndUTC, the off-peak rates apply during the
	// [start, end) UTC hour window (wraps past midnight if start > end).
	OffPeakInputPer1M  float64
	OffPeakOutputPer1M float64
	OffPeakStartUTC    int
	OffPeakEndUTC      int
}

// inOffPeak reports whether t falls in this provider's off-peak window.
func (p PricingInfo) inOffPeak(t time.Time) bool {
	if p.OffPeakStartUTC == p.OffPeakEndUTC {
		return false
	}
	h := t.UTC().Hour()
	if p.OffPeakStartUTC < p.OffPeakEndUTC {
		return h >= p.OffPeakStartUTC && h < p.OffPeakEndUTC
	}
	return h >= p.OffPeakStartUTC || h < p.OffPeakEndUTC // wraps midnight
}

// CalculateCostFullAt prices a usage breakdown, applying the off-peak input/output
// rates when t falls in the off-peak window.
func (p PricingInfo) CalculateCostFullAt(in, out, cacheRead, cacheWrite int, t time.Time) float64 {
	pr := p
	if p.inOffPeak(t) {
		if p.OffPeakInputPer1M > 0 {
			pr.InputPer1M = p.OffPeakInputPer1M
		}
		if p.OffPeakOutputPer1M > 0 {
			pr.OutputPer1M = p.OffPeakOutputPer1M
		}
	}
	return pr.CalculateCostFull(in, out, cacheRead, cacheWrite)
}

func (p PricingInfo) cacheReadRate() float64 {
	if p.CacheReadPer1M > 0 {
		return p.CacheReadPer1M
	}
	return p.InputPer1M * 0.1
}

func (p PricingInfo) cacheWriteRate() float64 {
	if p.CacheWritePer1M > 0 {
		return p.CacheWritePer1M
	}
	return p.InputPer1M * 1.25
}

// CalculateCost returns the cost for plain input/output token counts.
func (p PricingInfo) CalculateCost(inputTokens, outputTokens int) float64 {
	return (float64(inputTokens)/1_000_000)*p.InputPer1M +
		(float64(outputTokens)/1_000_000)*p.OutputPer1M
}

// CalculateCostFull prices a full usage breakdown, applying the cache-read
// discount and cache-write premium on top of normal input/output pricing.
func (p PricingInfo) CalculateCostFull(in, out, cacheRead, cacheWrite int) float64 {
	return p.CalculateCost(in, out) +
		(float64(cacheRead)/1_000_000)*p.cacheReadRate() +
		(float64(cacheWrite)/1_000_000)*p.cacheWriteRate()
}

// CacheReadSavings reports how much the cache-read tokens saved versus paying
// the full input price for them — the "cache saved $X" headline number.
func (p PricingInfo) CacheReadSavings(cacheRead int) float64 {
	saved := (float64(cacheRead) / 1_000_000) * (p.InputPer1M - p.cacheReadRate())
	if saved < 0 {
		return 0
	}
	return saved
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
	case "baseten":
		return NewBaseten(apiKey), nil
	case "featherless":
		return NewFeatherless(apiKey), nil
	case "kluster":
		return NewKluster(apiKey), nil
	case "venice":
		return NewVenice(apiKey), nil
	case "friendli":
		return NewFriendli(apiKey), nil
	case "chutes":
		return NewChutes(apiKey), nil
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
	case "deepseek", "mistral", "together", "openrouter", "cohere", "fireworks", "perplexity", "deepinfra", "novita", "hyperbolic", "nebius", "moonshot", "zhipu", "ai21", "lambda", "baseten", "featherless", "kluster", "venice", "friendli", "chutes":
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
	case "baseten":
		return []string{"deepseek-ai/DeepSeek-V3.1"}
	case "featherless":
		return []string{"meta-llama/Meta-Llama-3.1-70B-Instruct"}
	case "kluster":
		return []string{"klusterai/Meta-Llama-3.3-70B-Instruct-Turbo"}
	case "venice":
		return []string{"llama-3.3-70b"}
	case "friendli":
		return []string{"meta-llama-3.3-70b-instruct"}
	case "chutes":
		return []string{"deepseek-ai/DeepSeek-V3"}
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
