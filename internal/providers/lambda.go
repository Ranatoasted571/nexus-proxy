package providers

// Lambda provider — Lambda Inference API (OpenAI-compatible).
type Lambda struct{ apiKey string }

func NewLambda(apiKey string) *Lambda { return &Lambda{apiKey: apiKey} }

func (l *Lambda) Name() string               { return "lambda" }
func (l *Lambda) BaseURL() string            { return "https://api.lambda.ai/v1" }
func (l *Lambda) Tier() string               { return TierStandard }
func (l *Lambda) ChatCompletionsURL() string { return l.BaseURL() + "/chat/completions" }

func (l *Lambda) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "llama-3.1-8b-instruct"
	default:
		return "llama-3.3-70b-instruct-fp8"
	}
}

func (l *Lambda) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.20, OutputPer1M: 0.20} }
func (l *Lambda) HealthCheck() error   { return bearerHealthCheck(l.BaseURL()+"/models", l.apiKey) }
