package providers

// kluster.ai — OpenAI-compatible inference for open models (Llama, DeepSeek, Qwen).
// https://docs.kluster.ai/get-started/openai-compatibility/
type Kluster struct{ apiKey string }

func NewKluster(apiKey string) *Kluster { return &Kluster{apiKey: apiKey} }

func (k *Kluster) Name() string               { return "kluster" }
func (k *Kluster) BaseURL() string            { return "https://api.kluster.ai/v1" }
func (k *Kluster) Tier() string               { return TierStandard }
func (k *Kluster) ChatCompletionsURL() string { return k.BaseURL() + "/chat/completions" }

func (k *Kluster) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "klusterai/Meta-Llama-3.1-8B-Instruct-Turbo"
	default:
		return "klusterai/Meta-Llama-3.3-70B-Instruct-Turbo"
	}
}

func (k *Kluster) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.20, OutputPer1M: 0.30} }
func (k *Kluster) HealthCheck() error   { return bearerHealthCheck(k.BaseURL()+"/models", k.apiKey) }
