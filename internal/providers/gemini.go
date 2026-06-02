package providers

import (
	"fmt"
	"net/http"
	"time"
)

// Gemini provider — Google, free tier available
type Gemini struct {
	apiKey string
}

func NewGemini(apiKey string) *Gemini {
	return &Gemini{apiKey: apiKey}
}

func (g *Gemini) Name() string    { return "gemini" }
func (g *Gemini) BaseURL() string { return "https://generativelanguage.googleapis.com/v1beta/openai" }
func (g *Gemini) Tier() string    { return TierFree }

func (g *Gemini) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-opus-4-5", "claude-opus-4-6":
		return "gemini-2.0-flash-thinking-exp"
	case "claude-sonnet-4-6":
		return "gemini-2.0-flash"
	case "claude-haiku-4-5":
		return "gemini-2.0-flash-lite"
	default:
		return "gemini-2.0-flash"
	}
}

func (g *Gemini) Pricing() PricingInfo {
	return PricingInfo{
		InputPer1M:  0.0,  // Free tier (rate limited)
		OutputPer1M: 0.0,
	}
}

func (g *Gemini) ChatCompletionsURL() string { return g.BaseURL() + "/chat/completions" }

func (g *Gemini) HealthCheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", g.apiKey)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gemini health check failed: %d", resp.StatusCode)
	}
	return nil
}
