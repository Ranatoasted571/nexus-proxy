package providers

// Cohere provider — Command models via Cohere's OpenAI-compatibility endpoint.
type Cohere struct{ apiKey string }

func NewCohere(apiKey string) *Cohere { return &Cohere{apiKey: apiKey} }

func (c *Cohere) Name() string               { return "cohere" }
func (c *Cohere) BaseURL() string            { return "https://api.cohere.ai/compatibility/v1" }
func (c *Cohere) Tier() string               { return TierStandard }
func (c *Cohere) ChatCompletionsURL() string { return c.BaseURL() + "/chat/completions" }

func (c *Cohere) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "command-r"
	default:
		return "command-r-plus"
	}
}

func (c *Cohere) Pricing() PricingInfo { return PricingInfo{InputPer1M: 2.50, OutputPer1M: 10.00} }
func (c *Cohere) HealthCheck() error   { return bearerHealthCheck(c.BaseURL()+"/models", c.apiKey) }
