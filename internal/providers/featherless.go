package providers

// Featherless.ai — serverless access to thousands of open models (OpenAI-compatible).
// https://featherless.ai/docs/quickstart-guide
type Featherless struct{ apiKey string }

func NewFeatherless(apiKey string) *Featherless { return &Featherless{apiKey: apiKey} }

func (f *Featherless) Name() string               { return "featherless" }
func (f *Featherless) BaseURL() string            { return "https://api.featherless.ai/v1" }
func (f *Featherless) Tier() string               { return TierStandard }
func (f *Featherless) ChatCompletionsURL() string { return f.BaseURL() + "/chat/completions" }

func (f *Featherless) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/Meta-Llama-3.1-8B-Instruct"
	default:
		return "meta-llama/Meta-Llama-3.1-70B-Instruct"
	}
}

// Featherless is flat-rate subscription; per-token cost is nominal.
func (f *Featherless) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.10, OutputPer1M: 0.10} }
func (f *Featherless) HealthCheck() error   { return bearerHealthCheck(f.BaseURL()+"/models", f.apiKey) }
