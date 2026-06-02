package providers

// SambaNova provider — fast inference with a free tier (OpenAI-compatible).
type SambaNova struct{ apiKey string }

func NewSambaNova(apiKey string) *SambaNova { return &SambaNova{apiKey: apiKey} }

func (s *SambaNova) Name() string               { return "sambanova" }
func (s *SambaNova) BaseURL() string            { return "https://api.sambanova.ai/v1" }
func (s *SambaNova) Tier() string               { return TierFree }
func (s *SambaNova) ChatCompletionsURL() string { return s.BaseURL() + "/chat/completions" }

func (s *SambaNova) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "Meta-Llama-3.1-8B-Instruct"
	default:
		return "Meta-Llama-3.3-70B-Instruct"
	}
}

func (s *SambaNova) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.0, OutputPer1M: 0.0} }
func (s *SambaNova) HealthCheck() error   { return bearerHealthCheck(s.BaseURL()+"/models", s.apiKey) }
