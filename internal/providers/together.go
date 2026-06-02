package providers

// Together provider — Together AI, hosts many open models (OpenAI-compatible).
type Together struct{ apiKey string }

func NewTogether(apiKey string) *Together { return &Together{apiKey: apiKey} }

func (t *Together) Name() string               { return "together" }
func (t *Together) BaseURL() string            { return "https://api.together.xyz/v1" }
func (t *Together) Tier() string               { return TierStandard }
func (t *Together) ChatCompletionsURL() string { return t.BaseURL() + "/chat/completions" }

func (t *Together) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/Llama-3.2-3B-Instruct-Turbo"
	default:
		return "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	}
}

func (t *Together) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.88, OutputPer1M: 0.88} }
func (t *Together) HealthCheck() error   { return bearerHealthCheck(t.BaseURL()+"/models", t.apiKey) }
