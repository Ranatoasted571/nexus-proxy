package providers

import (
	"fmt"
	"net/http"
	"time"
)

// Anthropic provider — the real Claude, premium tier
type Anthropic struct {
	apiKey string
}

func NewAnthropic(apiKey string) *Anthropic {
	return &Anthropic{apiKey: apiKey}
}

func (a *Anthropic) Name() string    { return "anthropic" }
func (a *Anthropic) BaseURL() string { return "https://api.anthropic.com" }
func (a *Anthropic) Tier() string    { return TierPremium }

func (a *Anthropic) MapModel(claudeModel string) string {
	// Pass through — Anthropic uses native Claude models
	return claudeModel
}

func (a *Anthropic) Pricing() PricingInfo {
	// Using Sonnet pricing as default
	return PricingInfo{
		InputPer1M:  3.00,  // $3.00 per 1M input tokens (Sonnet)
		OutputPer1M: 15.00, // $15.00 per 1M output tokens (Sonnet)
	}
}

// ChatCompletionsURL is empty — Anthropic uses its native /v1/messages API.
func (a *Anthropic) ChatCompletionsURL() string { return "" }

func (a *Anthropic) HealthCheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("anthropic health check failed: %d", resp.StatusCode)
	}
	return nil
}
