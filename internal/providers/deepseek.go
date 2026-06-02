package providers

import (
	"fmt"
	"net/http"
	"time"
)

// DeepSeek provider — cheap, OpenAI-compatible
type DeepSeek struct {
	apiKey string
}

func NewDeepSeek(apiKey string) *DeepSeek {
	return &DeepSeek{apiKey: apiKey}
}

func (d *DeepSeek) Name() string    { return "deepseek" }
func (d *DeepSeek) BaseURL() string { return "https://api.deepseek.com" }
func (d *DeepSeek) Tier() string    { return TierStandard }

func (d *DeepSeek) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-opus-4-5", "claude-opus-4-6":
		return "deepseek-chat"
	case "claude-sonnet-4-6":
		return "deepseek-chat"
	case "claude-haiku-4-5":
		return "deepseek-chat"
	default:
		return "deepseek-chat"
	}
}

func (d *DeepSeek) Pricing() PricingInfo {
	return PricingInfo{
		InputPer1M:  0.27,  // $0.27 per 1M input tokens
		OutputPer1M: 1.10,  // $1.10 per 1M output tokens
	}
}

func (d *DeepSeek) ChatCompletionsURL() string { return d.BaseURL() + "/v1/chat/completions" }

func (d *DeepSeek) HealthCheck() error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.deepseek.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deepseek health check failed: %d", resp.StatusCode)
	}
	return nil
}
