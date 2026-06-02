package providers

// DeepInfra provider — cheap open-model hosting (OpenAI-compatible).
type DeepInfra struct{ apiKey string }

func NewDeepInfra(apiKey string) *DeepInfra { return &DeepInfra{apiKey: apiKey} }

func (d *DeepInfra) Name() string               { return "deepinfra" }
func (d *DeepInfra) BaseURL() string            { return "https://api.deepinfra.com/v1/openai" }
func (d *DeepInfra) Tier() string               { return TierStandard }
func (d *DeepInfra) ChatCompletionsURL() string { return d.BaseURL() + "/chat/completions" }

func (d *DeepInfra) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/Meta-Llama-3.1-8B-Instruct"
	default:
		return "meta-llama/Llama-3.3-70B-Instruct"
	}
}

func (d *DeepInfra) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.40, OutputPer1M: 0.40} }
func (d *DeepInfra) HealthCheck() error   { return bearerHealthCheck(d.BaseURL()+"/models", d.apiKey) }
