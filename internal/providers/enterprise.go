package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// pricingOr returns the spec's pricing, or the given defaults when unset.
func pricingOr(spec Spec, inDef, outDef float64) PricingInfo {
	if spec.InputPer1M == 0 && spec.OutputPer1M == 0 {
		return PricingInfo{InputPer1M: inDef, OutputPer1M: outDef}
	}
	return PricingInfo{InputPer1M: spec.InputPer1M, OutputPer1M: spec.OutputPer1M}
}

// rewriteAnthropicBody strips fields Bedrock/Vertex don't accept (model, stream)
// and injects the platform anthropic_version.
func rewriteAnthropicBody(body []byte, anthropicVersion string) []byte {
	var m map[string]interface{}
	if json.Unmarshal(body, &m) != nil {
		return body
	}
	delete(m, "model")
	delete(m, "stream")
	m["anthropic_version"] = anthropicVersion
	if out, err := json.Marshal(m); err == nil {
		return out
	}
	return body
}

// ─── Azure OpenAI (OpenAI-compatible, api-key header) ───────────────────────

type Azure struct {
	name, baseURL, apiVersion, tier, apiKey string
	models                                  []string
	modelMap                                map[string]string
	pricing                                 PricingInfo
}

func newAzure(spec Spec) (*Azure, error) {
	if spec.BaseURL == "" {
		return nil, fmt.Errorf("azure provider %q: base_url (the deployment endpoint) is required", spec.Name)
	}
	return &Azure{
		name:       orDefault(spec.Name, "azure"),
		baseURL:    strings.TrimRight(spec.BaseURL, "/"),
		apiVersion: orDefault(spec.APIVersion, "2024-10-21"),
		tier:       orDefault(spec.Tier, TierPremium),
		apiKey:     spec.APIKey,
		models:     spec.Models,
		modelMap:   spec.ModelMap,
		pricing:    pricingOr(spec, 2.50, 10.00),
	}, nil
}

func (a *Azure) Name() string               { return a.name }
func (a *Azure) BaseURL() string            { return a.baseURL }
func (a *Azure) Tier() string               { return a.tier }
func (a *Azure) Pricing() PricingInfo       { return a.pricing }
func (a *Azure) ChatCompletionsURL() string { return a.baseURL + "/chat/completions?api-version=" + a.apiVersion }
func (a *Azure) HealthCheck() error         { return reachableHealthCheck(a.baseURL) }

func (a *Azure) MapModel(claudeModel string) string {
	if m := mapFrom(a.modelMap, claudeModel); m != "" {
		return m
	}
	if len(a.models) > 0 {
		return a.models[0]
	}
	return claudeModel
}

// Azure authenticates with an "api-key" header rather than a Bearer token.
func (a *Azure) Authorize(req *http.Request, _ []byte, apiKey string) {
	req.Header.Set("api-key", apiKey)
}

// ─── Google Vertex AI (Anthropic-native, bearer access token) ───────────────

type Vertex struct {
	name, region, project, tier, baseURL string
	modelMap                             map[string]string
	pricing                              PricingInfo
}

func newVertex(spec Spec) (*Vertex, error) {
	if spec.Region == "" || spec.Project == "" {
		return nil, fmt.Errorf("vertex provider %q: region and project are required", spec.Name)
	}
	base := strings.TrimRight(spec.BaseURL, "/")
	if base == "" {
		base = fmt.Sprintf("https://%s-aiplatform.googleapis.com", spec.Region)
	}
	return &Vertex{
		name:     orDefault(spec.Name, "vertex"),
		region:   spec.Region,
		project:  spec.Project,
		tier:     orDefault(spec.Tier, TierPremium),
		baseURL:  base,
		modelMap: spec.ModelMap,
		pricing:  pricingOr(spec, 3.00, 15.00),
	}, nil
}

func (v *Vertex) Name() string               { return v.name }
func (v *Vertex) BaseURL() string            { return v.baseURL }
func (v *Vertex) Tier() string               { return v.tier }
func (v *Vertex) Pricing() PricingInfo       { return v.pricing }
func (v *Vertex) ChatCompletionsURL() string { return "" } // Anthropic-native
func (v *Vertex) HealthCheck() error         { return reachableHealthCheck(v.BaseURL()) }

func (v *Vertex) MapModel(claudeModel string) string {
	if m := mapFrom(v.modelMap, claudeModel); m != "" {
		return m
	}
	switch claudeModel {
	case "claude-haiku-4-5":
		return "claude-3-5-haiku@20241022"
	case "claude-opus-4-5", "claude-opus-4-6":
		return "claude-3-opus@20240229"
	default:
		return "claude-3-5-sonnet-v2@20241022"
	}
}

func (v *Vertex) MessagesURL(claudeModel string) string {
	return v.baseURL + fmt.Sprintf("/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:rawPredict",
		v.project, v.region, v.MapModel(claudeModel))
}

func (v *Vertex) PrepareBody(body []byte, _ string) []byte {
	return rewriteAnthropicBody(body, "vertex-2023-10-16")
}

func (v *Vertex) Authorize(req *http.Request, _ []byte, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

// ─── AWS Bedrock (Anthropic-native, SigV4) ──────────────────────────────────

type Bedrock struct {
	name, region, tier, baseURL string
	modelMap                    map[string]string
	pricing                     PricingInfo
}

func newBedrock(spec Spec) (*Bedrock, error) {
	if spec.Region == "" {
		return nil, fmt.Errorf("bedrock provider %q: region is required", spec.Name)
	}
	base := strings.TrimRight(spec.BaseURL, "/")
	if base == "" {
		base = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", spec.Region)
	}
	return &Bedrock{
		name:     orDefault(spec.Name, "bedrock"),
		region:   spec.Region,
		tier:     orDefault(spec.Tier, TierPremium),
		baseURL:  base,
		modelMap: spec.ModelMap,
		pricing:  pricingOr(spec, 3.00, 15.00),
	}, nil
}

func (b *Bedrock) Name() string               { return b.name }
func (b *Bedrock) BaseURL() string            { return b.baseURL }
func (b *Bedrock) Tier() string               { return b.tier }
func (b *Bedrock) Pricing() PricingInfo       { return b.pricing }
func (b *Bedrock) ChatCompletionsURL() string { return "" } // Anthropic-native
func (b *Bedrock) HealthCheck() error         { return reachableHealthCheck(b.BaseURL()) }

func (b *Bedrock) MapModel(claudeModel string) string {
	if m := mapFrom(b.modelMap, claudeModel); m != "" {
		return m
	}
	switch claudeModel {
	case "claude-haiku-4-5":
		return "anthropic.claude-3-5-haiku-20241022-v1:0"
	case "claude-opus-4-5", "claude-opus-4-6":
		return "anthropic.claude-3-opus-20240229-v1:0"
	default:
		return "anthropic.claude-3-5-sonnet-20241022-v2:0"
	}
}

func (b *Bedrock) MessagesURL(claudeModel string) string {
	model := strings.ReplaceAll(b.MapModel(claudeModel), ":", "%3A")
	return b.baseURL + "/model/" + model + "/invoke"
}

func (b *Bedrock) PrepareBody(body []byte, _ string) []byte {
	return rewriteAnthropicBody(body, "bedrock-2023-05-31")
}

// Authorize signs the request with AWS SigV4 using credentials from the
// standard AWS_* environment variables.
func (b *Bedrock) Authorize(req *http.Request, body []byte, _ string) {
	signV4(req, body,
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_SESSION_TOKEN"),
		b.region, "bedrock", time.Now())
}
