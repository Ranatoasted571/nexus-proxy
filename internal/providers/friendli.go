package providers

// FriendliAI Serverless Endpoints — OpenAI-compatible inference for open models.
// https://friendli.ai/docs/guides/serverless_endpoints/openai-compatibility
type Friendli struct{ apiKey string }

func NewFriendli(apiKey string) *Friendli { return &Friendli{apiKey: apiKey} }

func (f *Friendli) Name() string               { return "friendli" }
func (f *Friendli) BaseURL() string            { return "https://api.friendli.ai/serverless/v1" }
func (f *Friendli) Tier() string               { return TierStandard }
func (f *Friendli) ChatCompletionsURL() string { return f.BaseURL() + "/chat/completions" }

func (f *Friendli) MapModel(claudeModel string) string {
	switch claudeModel {
	case "claude-haiku-4-5":
		return "meta-llama-3.1-8b-instruct"
	default:
		return "meta-llama-3.3-70b-instruct"
	}
}

func (f *Friendli) Pricing() PricingInfo { return PricingInfo{InputPer1M: 0.60, OutputPer1M: 0.60} }
func (f *Friendli) HealthCheck() error   { return bearerHealthCheck(f.BaseURL()+"/models", f.apiKey) }
