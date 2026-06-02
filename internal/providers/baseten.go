package providers

// Baseten Model APIs — OpenAI-compatible hosting for open models.
// https://docs.baseten.co/development/model-apis/overview
type Baseten struct{ apiKey string }

func NewBaseten(apiKey string) *Baseten { return &Baseten{apiKey: apiKey} }

func (b *Baseten) Name() string               { return "baseten" }
func (b *Baseten) BaseURL() string            { return "https://inference.baseten.co/v1" }
func (b *Baseten) Tier() string               { return TierStandard }
func (b *Baseten) ChatCompletionsURL() string { return b.BaseURL() + "/chat/completions" }

func (b *Baseten) MapModel(string) string { return "deepseek-ai/DeepSeek-V3.1" }

func (b *Baseten) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.50, OutputPer1M: 1.50} }
func (b *Baseten) HealthCheck() error   { return bearerHealthCheck(b.BaseURL()+"/models", b.apiKey) }
