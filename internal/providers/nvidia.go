package providers

// NVIDIA provider — NVIDIA NIM (build.nvidia.com), free credits (OpenAI-compatible).
type NVIDIA struct{ apiKey string }

func NewNVIDIA(apiKey string) *NVIDIA { return &NVIDIA{apiKey: apiKey} }

func (n *NVIDIA) Name() string               { return "nvidia" }
func (n *NVIDIA) BaseURL() string            { return "https://integrate.api.nvidia.com/v1" }
func (n *NVIDIA) Tier() string               { return TierFree }
func (n *NVIDIA) ChatCompletionsURL() string { return n.BaseURL() + "/chat/completions" }

func (n *NVIDIA) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta/llama-3.1-8b-instruct"
	default:
		return "meta/llama-3.3-70b-instruct"
	}
}

func (n *NVIDIA) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.0, OutputPer1M: 0.0} }
func (n *NVIDIA) HealthCheck() error   { return bearerHealthCheck(n.BaseURL()+"/models", n.apiKey) }
