package providers

// Novita provider — Novita AI, open-model hosting (OpenAI-compatible).
type Novita struct{ apiKey string }

func NewNovita(apiKey string) *Novita { return &Novita{apiKey: apiKey} }

func (n *Novita) Name() string               { return "novita" }
func (n *Novita) BaseURL() string            { return "https://api.novita.ai/v3/openai" }
func (n *Novita) Tier() string               { return TierStandard }
func (n *Novita) ChatCompletionsURL() string { return n.BaseURL() + "/chat/completions" }

func (n *Novita) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/llama-3.2-3b-instruct"
	default:
		return "meta-llama/llama-3.3-70b-instruct"
	}
}

func (n *Novita) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.39, OutputPer1M: 0.39} }
func (n *Novita) HealthCheck() error   { return bearerHealthCheck(n.BaseURL()+"/models", n.apiKey) }
