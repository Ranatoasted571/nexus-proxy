package providers

// Moonshot provider — Moonshot AI / Kimi (OpenAI-compatible).
type Moonshot struct{ apiKey string }

func NewMoonshot(apiKey string) *Moonshot { return &Moonshot{apiKey: apiKey} }

func (m *Moonshot) Name() string               { return "moonshot" }
func (m *Moonshot) BaseURL() string            { return "https://api.moonshot.ai/v1" }
func (m *Moonshot) Tier() string               { return TierStandard }
func (m *Moonshot) ChatCompletionsURL() string { return m.BaseURL() + "/chat/completions" }

func (m *Moonshot) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "moonshot-v1-8k"
	default:
		return "moonshot-v1-32k"
	}
}

func (m *Moonshot) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.50, OutputPer1M: 2.00} }
func (m *Moonshot) HealthCheck() error   { return bearerHealthCheck(m.BaseURL()+"/models", m.apiKey) }
