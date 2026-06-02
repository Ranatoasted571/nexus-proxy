package providers

// Mistral provider — Mistral AI, OpenAI-compatible.
type Mistral struct{ apiKey string }

func NewMistral(apiKey string) *Mistral { return &Mistral{apiKey: apiKey} }

func (m *Mistral) Name() string               { return "mistral" }
func (m *Mistral) BaseURL() string            { return "https://api.mistral.ai/v1" }
func (m *Mistral) Tier() string               { return TierStandard }
func (m *Mistral) ChatCompletionsURL() string { return m.BaseURL() + "/chat/completions" }

func (m *Mistral) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "mistral-small-latest"
	default:
		return "mistral-large-latest"
	}
}

func (m *Mistral) Pricing() PricingInfo { return PricingInfo{InputPer1M: 2.00, OutputPer1M: 6.00} }
func (m *Mistral) HealthCheck() error   { return bearerHealthCheck(m.BaseURL()+"/models", m.apiKey) }
