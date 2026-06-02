package providers

import (
	"fmt"
	"net/http"
	"time"
)

// Groq provider — free tier, very fast inference
type Groq struct {
	apiKey string
}

func NewGroq(apiKey string) *Groq {
	return &Groq{apiKey: apiKey}
}

func (g *Groq) Name() string    { return "groq" }
func (g *Groq) BaseURL() string { return "https://api.groq.com/openai" }
func (g *Groq) Tier() string    { return TierFree }

func (g *Groq) MapModel(claudeModel string) string {
	// Groq's free models — Llama 3.3 70B is the best option
	switch claudeModel {
	case "claude-opus-4-5", "claude-opus-4-6", "claude-sonnet-4-6":
		return "llama-3.3-70b-versatile"
	case "claude-haiku-4-5":
		return "llama-3.1-8b-instant"
	default:
		return "llama-3.3-70b-versatile"
	}
}

func (g *Groq) Pricing() PricingInfo {
	return PricingInfo{
		InputPer1M:  0.0,  // Free tier
		OutputPer1M: 0.0,
	}
}

func (g *Groq) ChatCompletionsURL() string { return g.BaseURL() + "/v1/chat/completions" }

func (g *Groq) HealthCheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.groq.com/openai/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("groq health check failed: %d", resp.StatusCode)
	}
	return nil
}
