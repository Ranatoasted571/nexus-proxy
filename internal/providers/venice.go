package providers

// Venice.ai — privacy-focused, OpenAI-compatible inference (no logging, uncensored
// open models). https://docs.venice.ai/api-reference/api-spec
type Venice struct{ apiKey string }

func NewVenice(apiKey string) *Venice { return &Venice{apiKey: apiKey} }

func (v *Venice) Name() string               { return "venice" }
func (v *Venice) BaseURL() string            { return "https://api.venice.ai/api/v1" }
func (v *Venice) Tier() string               { return TierStandard }
func (v *Venice) ChatCompletionsURL() string { return v.BaseURL() + "/chat/completions" }

func (v *Venice) MapModel(string) string { return "llama-3.3-70b" }

func (v *Venice) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.70, OutputPer1M: 2.80} }
func (v *Venice) HealthCheck() error   { return bearerHealthCheck(v.BaseURL()+"/models", v.apiKey) }
