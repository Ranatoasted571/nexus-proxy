package providers

// OpenRouter provider — aggregator that routes to many model providers.
type OpenRouter struct{ apiKey string }

func NewOpenRouter(apiKey string) *OpenRouter { return &OpenRouter{apiKey: apiKey} }

func (o *OpenRouter) Name() string               { return "openrouter" }
func (o *OpenRouter) BaseURL() string            { return "https://openrouter.ai/api/v1" }
func (o *OpenRouter) Tier() string               { return TierStandard }
func (o *OpenRouter) ChatCompletionsURL() string { return o.BaseURL() + "/chat/completions" }

func (o *OpenRouter) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/llama-3.2-3b-instruct"
	default:
		return "meta-llama/llama-3.3-70b-instruct"
	}
}

func (o *OpenRouter) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.60, OutputPer1M: 0.60} }
func (o *OpenRouter) HealthCheck() error   { return bearerHealthCheck(o.BaseURL()+"/models", o.apiKey) }
