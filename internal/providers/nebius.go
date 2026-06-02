package providers

// Nebius provider — Nebius AI Studio (OpenAI-compatible).
type Nebius struct{ apiKey string }

func NewNebius(apiKey string) *Nebius { return &Nebius{apiKey: apiKey} }

func (n *Nebius) Name() string               { return "nebius" }
func (n *Nebius) BaseURL() string            { return "https://api.studio.nebius.ai/v1" }
func (n *Nebius) Tier() string               { return TierStandard }
func (n *Nebius) ChatCompletionsURL() string { return n.BaseURL() + "/chat/completions" }

func (n *Nebius) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/Llama-3.2-3B-Instruct"
	default:
		return "meta-llama/Llama-3.3-70B-Instruct"
	}
}

func (n *Nebius) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.13, OutputPer1M: 0.40} }
func (n *Nebius) HealthCheck() error   { return bearerHealthCheck(n.BaseURL()+"/models", n.apiKey) }
