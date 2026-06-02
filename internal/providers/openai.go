package providers

// OpenAI provider — GPT models, the original OpenAI-compatible API.
type OpenAI struct{ apiKey string }

func NewOpenAI(apiKey string) *OpenAI { return &OpenAI{apiKey: apiKey} }

func (o *OpenAI) Name() string                { return "openai" }
func (o *OpenAI) BaseURL() string             { return "https://api.openai.com/v1" }
func (o *OpenAI) Tier() string                { return TierPremium }
func (o *OpenAI) ChatCompletionsURL() string  { return o.BaseURL() + "/chat/completions" }

func (o *OpenAI) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "gpt-4o-mini"
	default:
		return "gpt-4o"
	}
}

func (o *OpenAI) Pricing() PricingInfo { return PricingInfo{InputPer1M: 2.50, OutputPer1M: 10.00} }
func (o *OpenAI) HealthCheck() error   { return bearerHealthCheck(o.BaseURL()+"/models", o.apiKey) }
