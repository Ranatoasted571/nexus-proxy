package proxy

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/router"
)

func TestRuleMatches(t *testing.T) {
	yes := true
	if !ruleMatches(config.Rule{WhenPromptContains: "prod"}, "m", "deploy to prod", "simple", false) {
		t.Error("prompt substring should match")
	}
	if ruleMatches(config.Rule{WhenPromptContains: "prod"}, "m", "hello world", "simple", false) {
		t.Error("non-matching prompt should not match")
	}
	if !ruleMatches(config.Rule{WhenModelContains: "opus"}, "claude-opus-4-5", "", "complex", false) {
		t.Error("model substring should match")
	}
	if !ruleMatches(config.Rule{WhenComplexity: "complex"}, "m", "", "complex", false) {
		t.Error("complexity should match")
	}
	if !ruleMatches(config.Rule{WhenHasTools: &yes}, "m", "", "simple", true) {
		t.Error("hasTools should match")
	}
	if ruleMatches(config.Rule{}, "m", "x", "simple", false) {
		t.Error("an empty rule must never match")
	}
}

func TestApplyRulesFirstMatch(t *testing.T) {
	rules := []config.Rule{
		{WhenComplexity: "simple", UseProvider: "groq"},
		{WhenModelContains: "opus", UseTier: "premium"},
	}
	if p, tier := applyRules(rules, "claude-haiku", "hi", router.ComplexitySimple, false); p != "groq" || tier != "" {
		t.Errorf("first matching rule should win: got %q / %q", p, tier)
	}
}

func TestFilterByTier(t *testing.T) {
	chain := []*router.Provider{{Name: "a", Tier: "free"}, {Name: "b", Tier: "premium"}}
	if out := filterByTier(chain, "premium"); len(out) != 1 || out[0].Name != "b" {
		t.Errorf("filter premium = %v", out)
	}
	if out := filterByTier(chain, "local"); len(out) != 2 {
		t.Error("no-match tier should return the original chain")
	}
}

func TestEstimateTokens(t *testing.T) {
	if estimateTokens("12345678") != 2 {
		t.Errorf("got %d", estimateTokens("12345678"))
	}
}

func TestRuleForcesProvider(t *testing.T) {
	var aHits, bHits int32
	a := countingOAIServer("a", &aHits)
	b := countingOAIServer("b", &bHits)
	defer a.Close()
	defer b.Close()
	h := buildTestHandler(t, []testProv{{"a", "free", a.URL}, {"b", "premium", b.URL}})
	h.rules = []config.Rule{{WhenPromptContains: "production", UseProvider: "b"}}

	// A short prompt would normally route to free 'a'; the rule pins it to 'b'.
	doMessages(h, `{"model":"claude-haiku-4-5","max_tokens":50,"messages":[{"role":"user","content":"deploy to production now"}]}`)
	if bHits != 1 || aHits != 0 {
		t.Fatalf("rule should force provider b; a=%d b=%d", aHits, bHits)
	}
}

func TestGuardrailDowngradesToFree(t *testing.T) {
	var cheapHits, premHits int32
	cheap := countingOAIServer("c", &cheapHits)
	prem := countingOAIServer("p", &premHits)
	defer cheap.Close()
	defer prem.Close()

	rt := router.New(router.StrategyAuto)
	mk := func(name, tier, url string, price float64) *activeProvider {
		impl, _ := providers.New(providers.Spec{Name: name, Type: "openai-compatible", BaseURL: url, Tier: tier, InputPer1M: price, OutputPer1M: price})
		rt.AddProvider(&router.Provider{Name: name, Tier: tier, Healthy: true})
		return &activeProvider{impl: impl, apiKey: "k"}
	}
	h := &Handler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		router:     rt,
		providers: map[string]*activeProvider{
			"cheap": mk("cheap", "free", cheap.URL, 0),
			"prem":  mk("prem", "premium", prem.URL, 100), // $100/1M ⇒ any real request exceeds a tiny cap
		},
	}
	h.maxReqUSD = 0.001

	body := `{"model":"claude-opus-4-5","max_tokens":1000,"messages":[{"role":"user","content":"` +
		strings.Repeat("design the production architecture ", 40) + `"}]}`
	doMessages(h, body)

	if cheapHits != 1 || premHits != 0 {
		t.Fatalf("guardrail should downgrade the pricey request to free; cheap=%d prem=%d", cheapHits, premHits)
	}
}
