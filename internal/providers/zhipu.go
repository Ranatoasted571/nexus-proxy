package providers

// Zhipu provider — Zhipu AI / GLM models (OpenAI-compatible).
type Zhipu struct{ apiKey string }

func NewZhipu(apiKey string) *Zhipu { return &Zhipu{apiKey: apiKey} }

func (z *Zhipu) Name() string               { return "zhipu" }
func (z *Zhipu) BaseURL() string            { return "https://open.bigmodel.cn/api/paas/v4" }
func (z *Zhipu) Tier() string               { return TierStandard }
func (z *Zhipu) ChatCompletionsURL() string { return z.BaseURL() + "/chat/completions" }

func (z *Zhipu) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "glm-4-flash"
	default:
		return "glm-4-plus"
	}
}

func (z *Zhipu) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.50, OutputPer1M: 0.50} }
func (z *Zhipu) HealthCheck() error   { return bearerHealthCheck(z.BaseURL()+"/models", z.apiKey) }
