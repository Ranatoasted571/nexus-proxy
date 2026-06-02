package providers

import (
	"fmt"
	"net/http"
	"time"
)

// Perplexity provider — Sonar models with built-in web search (OpenAI-compatible).
type Perplexity struct{ apiKey string }

func NewPerplexity(apiKey string) *Perplexity { return &Perplexity{apiKey: apiKey} }

func (p *Perplexity) Name() string               { return "perplexity" }
func (p *Perplexity) BaseURL() string            { return "https://api.perplexity.ai" }
func (p *Perplexity) Tier() string               { return TierStandard }
func (p *Perplexity) ChatCompletionsURL() string { return p.BaseURL() + "/chat/completions" }

func (p *Perplexity) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "sonar"
	default:
		return "sonar-pro"
	}
}

func (p *Perplexity) Pricing() PricingInfo { return PricingInfo{InputPer1M: 1.00, OutputPer1M: 1.00} }

// Perplexity has no public models endpoint, so we just check reachability.
func (p *Perplexity) HealthCheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(p.BaseURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("perplexity unreachable: %d", resp.StatusCode)
	}
	return nil
}
