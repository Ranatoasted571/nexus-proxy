package proxy

import (
	"strings"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/router"
)

// applyRules returns the first matching rule's (provider, tier) override.
func applyRules(rules []config.Rule, model, ptext string, complexity router.Complexity, hasTools bool) (provider, tier string) {
	cx := complexity.String()
	for _, rule := range rules {
		if ruleMatches(rule, model, ptext, cx, hasTools) {
			return rule.UseProvider, rule.UseTier
		}
	}
	return "", ""
}

// ruleMatches reports whether all of a rule's non-empty conditions hold. A rule
// with no conditions never matches (it would otherwise capture everything).
func ruleMatches(rule config.Rule, model, ptext, complexity string, hasTools bool) bool {
	if rule.WhenModelContains == "" && rule.WhenPromptContains == "" &&
		rule.WhenComplexity == "" && rule.WhenHasTools == nil {
		return false
	}
	if rule.WhenModelContains != "" && !strings.Contains(strings.ToLower(model), strings.ToLower(rule.WhenModelContains)) {
		return false
	}
	if rule.WhenPromptContains != "" && !strings.Contains(strings.ToLower(ptext), strings.ToLower(rule.WhenPromptContains)) {
		return false
	}
	if rule.WhenComplexity != "" && !strings.EqualFold(rule.WhenComplexity, complexity) {
		return false
	}
	if rule.WhenHasTools != nil && *rule.WhenHasTools != hasTools {
		return false
	}
	return true
}

// filterByTier keeps only providers of the given tier; if none match, the
// original chain is returned (the rule can't be satisfied, so it's ignored).
func filterByTier(chain []*router.Provider, tier string) []*router.Provider {
	var out []*router.Provider
	for _, p := range chain {
		if p.Tier == tier {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return chain
	}
	return out
}

// estimateTokens is a rough char/4 token estimate for the cost guardrail.
func estimateTokens(text string) int { return len(text) / 4 }
