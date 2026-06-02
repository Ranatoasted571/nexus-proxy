package providers

// Chutes.ai — decentralized, very cheap OpenAI-compatible inference (DeepSeek, Llama,
// Qwen, Mistral). https://chutes.ai/docs/getting-started/quickstart
type Chutes struct{ apiKey string }

func NewChutes(apiKey string) *Chutes { return &Chutes{apiKey: apiKey} }

func (c *Chutes) Name() string               { return "chutes" }
func (c *Chutes) BaseURL() string            { return "https://llm.chutes.ai/v1" }
func (c *Chutes) Tier() string               { return TierStandard }
func (c *Chutes) ChatCompletionsURL() string { return c.BaseURL() + "/chat/completions" }

func (c *Chutes) MapModel(string) string { return "deepseek-ai/DeepSeek-V3" }

func (c *Chutes) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.20, OutputPer1M: 0.20} }
func (c *Chutes) HealthCheck() error   { return bearerHealthCheck(c.BaseURL()+"/models", c.apiKey) }
