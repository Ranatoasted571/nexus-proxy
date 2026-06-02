package providers

// AI21 provider — AI21 Labs / Jamba models (OpenAI-compatible).
type AI21 struct{ apiKey string }

func NewAI21(apiKey string) *AI21 { return &AI21{apiKey: apiKey} }

func (a *AI21) Name() string               { return "ai21" }
func (a *AI21) BaseURL() string            { return "https://api.ai21.com/studio/v1" }
func (a *AI21) Tier() string               { return TierStandard }
func (a *AI21) ChatCompletionsURL() string { return a.BaseURL() + "/chat/completions" }

func (a *AI21) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "jamba-mini-1.7"
	default:
		return "jamba-large-1.7"
	}
}

func (a *AI21) Pricing() PricingInfo { return PricingInfo{InputPer1M: 2.00, OutputPer1M: 8.00} }
func (a *AI21) HealthCheck() error   { return bearerHealthCheck(a.BaseURL()+"/models", a.apiKey) }
