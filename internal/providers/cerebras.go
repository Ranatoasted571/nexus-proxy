package providers

// Cerebras provider — extremely fast inference, free tier (OpenAI-compatible).
type Cerebras struct{ apiKey string }

func NewCerebras(apiKey string) *Cerebras { return &Cerebras{apiKey: apiKey} }

func (c *Cerebras) Name() string               { return "cerebras" }
func (c *Cerebras) BaseURL() string            { return "https://api.cerebras.ai/v1" }
func (c *Cerebras) Tier() string               { return TierFree }
func (c *Cerebras) ChatCompletionsURL() string { return c.BaseURL() + "/chat/completions" }

func (c *Cerebras) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "llama3.1-8b"
	default:
		return "llama-3.3-70b"
	}
}

func (c *Cerebras) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.0, OutputPer1M: 0.0} }
func (c *Cerebras) HealthCheck() error   { return bearerHealthCheck(c.BaseURL()+"/models", c.apiKey) }
