package providers

import (
	"fmt"
	"net/http"
	"time"
)

// Ollama provider — fully local, zero cost
type Ollama struct {
	baseURL string
	model   string
}

func NewOllama(baseURL, model string) *Ollama {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "codellama:13b"
	}
	return &Ollama{baseURL: baseURL, model: model}
}

func (o *Ollama) Name() string    { return "ollama" }
func (o *Ollama) BaseURL() string { return o.baseURL + "/v1" }
func (o *Ollama) Tier() string    { return TierLocal }

func (o *Ollama) MapModel(_ string) string {
	return o.model
}

func (o *Ollama) Pricing() PricingInfo {
	return PricingInfo{
		InputPer1M:  0.0,
		OutputPer1M: 0.0,
	}
}

func (o *Ollama) ChatCompletionsURL() string { return o.BaseURL() + "/chat/completions" }

func (o *Ollama) HealthCheck() error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(o.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("ollama not reachable at %s: %w", o.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama health check failed: %d", resp.StatusCode)
	}
	return nil
}
