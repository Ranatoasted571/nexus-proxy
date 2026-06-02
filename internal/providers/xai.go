package providers

// xAI provider — Grok models, OpenAI-compatible.
type XAI struct{ apiKey string }

func NewXAI(apiKey string) *XAI { return &XAI{apiKey: apiKey} }

func (x *XAI) Name() string               { return "xai" }
func (x *XAI) BaseURL() string            { return "https://api.x.ai/v1" }
func (x *XAI) Tier() string               { return TierPremium }
func (x *XAI) ChatCompletionsURL() string { return x.BaseURL() + "/chat/completions" }

func (x *XAI) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "grok-2-1212"
	default:
		return "grok-2-latest"
	}
}

func (x *XAI) Pricing() PricingInfo { return PricingInfo{InputPer1M: 2.00, OutputPer1M: 10.00} }
func (x *XAI) HealthCheck() error   { return bearerHealthCheck(x.BaseURL()+"/models", x.apiKey) }
