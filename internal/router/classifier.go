package router

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// ClassifyRequest analyzes a request and returns its complexity. The policy:
//
//   - security / production / urgency keywords  → Critical (premium, e.g. Opus)
//   - architecture / large-build keywords       → Complex  (premium, e.g. Sonnet)
//   - very large context, very long sessions,
//     or an explicitly-chosen Opus model        → Complex
//   - a trivial, tool-less, non-code ask         → Simple   (free tier)
//   - everything else (ordinary coding, tool
//     use, code edits)                           → Standard (cheap + capable)
//
// This deliberately routes the bulk of agentic coding to the cheap Standard tier
// (which handles tool calls reliably), keeps the free tier for genuinely trivial
// chat, and reserves premium models for hard, large, or critical work.
func ClassifyRequest(model string, messages []map[string]interface{}, hasTools bool) Complexity {
	// Intent keywords come from the *user's* words, not the assistant's verbose
	// replies (an assistant explaining "architecture" must not escalate the next
	// turn). Size signals use the whole payload.
	allText := extractAllText(messages)
	userLower := strings.ToLower(extractUserText(messages))

	if hasCriticalKeywords(userLower) {
		return ComplexityCritical
	}
	if hasComplexKeywords(userLower) {
		return ComplexityComplex
	}

	// Big context or long agentic sessions lean complex; an explicit Opus pick
	// means Claude Code already escalated.
	if estimateTokens(allText) > 4000 || len(messages) > 24 || modelToComplexity(model) == ComplexityCritical {
		return ComplexityComplex
	}

	hasCode := strings.Count(allText, "```") >= 2
	standard := hasStandardKeywords(userLower)

	// Trivial, tool-less, non-code ask → free tier. Tool-using requests stay at
	// least Standard, since the free tier is less reliable at tool calls.
	if !hasTools && !hasCode && !standard && isShortAsk(lastUserText(messages)) {
		return ComplexitySimple
	}
	return ComplexityStandard
}

var reShortCritical = regexp.MustCompile(`\b(xss|csrf|rce|ssrf|cve)\b`)

func hasCriticalKeywords(lower string) bool {
	phrases := []string{
		"security vulnerability", "sql injection", "authentication bypass",
		"production down", "production outage", "production incident", "prod is down",
		"data breach", "data loss", "critical bug", "hotfix", "urgent", "emergency", "outage",
	}
	for _, kw := range phrases {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return reShortCritical.MatchString(lower)
}

func hasComplexKeywords(lower string) bool {
	complex := []string{
		"architecture", "design pattern", "system design", "microservice",
		"scalability", "performance optimization", "refactor the entire",
		"redesign", "implement from scratch", "build a full", "build the entire",
		"create a complete", "comprehensive", "step by step plan", "step-by-step plan",
		"roadmap", "high-level design", "trade-offs", "tradeoffs",
	}
	for _, kw := range complex {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// hasStandardKeywords flags ordinary, real coding intent — enough to keep a short
// tool-less coding ask off the free tier (but not premium).
func hasStandardKeywords(lower string) bool {
	std := []string{
		"refactor", "debug", "implement", "fix the", "fix this", "the bug",
		"write a", "unit test", "stack trace", "type error", "compile",
		"optimize", "rename", "migrate", "add a function", "add a method",
		"why does", "why isn't", "error:", "exception", "traceback",
	}
	for _, kw := range std {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// isShortAsk reports whether text reads like a short, casual question.
func isShortAsk(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return true
	}
	return len(strings.Fields(t)) <= 16 && utf8.RuneCountInString(t) < 200
}

// modelToComplexity maps Claude model names to a complexity hint.
func modelToComplexity(model string) Complexity {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return ComplexityCritical
	case strings.Contains(lower, "sonnet"):
		return ComplexityComplex
	case strings.Contains(lower, "haiku"):
		return ComplexitySimple
	default:
		return ComplexityStandard
	}
}

// lastUserText returns the text of the most recent user message.
func lastUserText(messages []map[string]interface{}) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if r, _ := messages[i]["role"].(string); r != "user" && r != "" {
			continue
		}
		return msgText(messages[i])
	}
	if len(messages) > 0 {
		return msgText(messages[len(messages)-1])
	}
	return ""
}

func msgText(msg map[string]interface{}) string {
	switch c := msg["content"].(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, block := range c {
			if b, ok := block.(map[string]interface{}); ok {
				if b["type"] == "text" {
					if t, ok := b["text"].(string); ok {
						parts = append(parts, t)
					}
				}
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// extractUserText pulls text from user messages only — the human's intent.
func extractUserText(messages []map[string]interface{}) string {
	var parts []string
	for _, msg := range messages {
		if r, _ := msg["role"].(string); r != "user" && r != "" {
			continue
		}
		if s := msgText(msg); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// extractAllText pulls all text content from every message.
func extractAllText(messages []map[string]interface{}) string {
	var parts []string
	for _, msg := range messages {
		if s := msgText(msg); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// estimateTokens is a rough char/4 token estimate.
func estimateTokens(text string) int { return utf8.RuneCountInString(text) / 4 }
