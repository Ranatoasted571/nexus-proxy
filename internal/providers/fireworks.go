package providers

// Fireworks provider — Fireworks AI, fast open-model inference (OpenAI-compatible).
type Fireworks struct{ apiKey string }

func NewFireworks(apiKey string) *Fireworks { return &Fireworks{apiKey: apiKey} }

func (f *Fireworks) Name() string               { return "fireworks" }
func (f *Fireworks) BaseURL() string            { return "https://api.fireworks.ai/inference/v1" }
func (f *Fireworks) Tier() string               { return TierStandard }
func (f *Fireworks) ChatCompletionsURL() string { return f.BaseURL() + "/chat/completions" }

func (f *Fireworks) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "accounts/fireworks/models/llama-v3p1-8b-instruct"
	default:
		return "accounts/fireworks/models/llama-v3p3-70b-instruct"
	}
}

func (f *Fireworks) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.90, OutputPer1M: 0.90} }
func (f *Fireworks) HealthCheck() error   { return bearerHealthCheck(f.BaseURL()+"/models", f.apiKey) }
