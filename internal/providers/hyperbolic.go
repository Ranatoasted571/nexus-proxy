package providers

// Hyperbolic provider — open-model inference (OpenAI-compatible).
type Hyperbolic struct{ apiKey string }

func NewHyperbolic(apiKey string) *Hyperbolic { return &Hyperbolic{apiKey: apiKey} }

func (h *Hyperbolic) Name() string               { return "hyperbolic" }
func (h *Hyperbolic) BaseURL() string            { return "https://api.hyperbolic.xyz/v1" }
func (h *Hyperbolic) Tier() string               { return TierStandard }
func (h *Hyperbolic) ChatCompletionsURL() string { return h.BaseURL() + "/chat/completions" }

func (h *Hyperbolic) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama/Llama-3.2-3B-Instruct"
	default:
		return "meta-llama/Llama-3.3-70B-Instruct"
	}
}

func (h *Hyperbolic) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.40, OutputPer1M: 0.40} }
func (h *Hyperbolic) HealthCheck() error   { return bearerHealthCheck(h.BaseURL()+"/models", h.apiKey) }
