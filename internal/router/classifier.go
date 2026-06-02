package router

import (
	"strings"
	"unicode/utf8"
)

// ClassifyRequest analyzes a request and returns its complexity
func ClassifyRequest(model string, messages []map[string]interface{}, hasTools bool) Complexity {
	// 1. Critical keywords override everything
	allText := extractAllText(messages)
	if hasCriticalKeywords(allText) {
		return ComplexityCritical
	}

	// 2. Complex keywords
	if hasComplexKeywords(allText) {
		return ComplexityComplex
	}

	// 3. Model hint from Claude Code
	// Claude Code itself picks the model based on task complexity
	// We can use this as a signal
	modelComplexity := modelToComplexity(model)

	// 4. Content signals
	tokenCount := estimateTokens(allText)
	hasCodeBlocks := strings.Contains(allText, "```")
	hasToolUse := hasTools
	conversationLength := len(messages)

	// Score-based classification
	score := 0

	// Token count signals
	if tokenCount > 2000 {
		score += 3
	} else if tokenCount > 500 {
		score += 1
	}

	// Content signals
	if hasCodeBlocks {
		score += 1
	}
	if hasToolUse {
		score += 2
	}
	if conversationLength > 10 {
		score += 1
	}

	// Model hint
	switch modelComplexity {
	case ComplexityCritical:
		score += 4
	case ComplexityComplex:
		score += 2
	case ComplexityStandard:
		score += 1
	}

	// Map score to complexity
	switch {
	case score >= 6:
		return ComplexityComplex
	case score >= 3:
		return ComplexityStandard
	default:
		return ComplexitySimple
	}
}

// hasCriticalKeywords checks for urgent/production/security signals
func hasCriticalKeywords(text string) bool {
	critical := []string{
		"security vulnerability",
		"sql injection",
		"xss",
		"csrf",
		"authentication bypass",
		"production down",
		"production issue",
		"data breach",
		"data loss",
		"critical bug",
		"hotfix",
		"urgent",
		"emergency",
		"outage",
	}
	lower := strings.ToLower(text)
	for _, kw := range critical {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// hasComplexKeywords checks for architecture/planning signals
func hasComplexKeywords(text string) bool {
	complex := []string{
		"architecture",
		"design pattern",
		"system design",
		"microservices",
		"scalability",
		"performance optimization",
		"refactor the entire",
		"redesign",
		"implement from scratch",
		"build a full",
		"create a complete",
		"comprehensive",
		"step by step plan",
		"roadmap",
	}
	lower := strings.ToLower(text)
	for _, kw := range complex {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// modelToComplexity maps Claude model names to complexity levels
// Claude Code uses specific models for specific task types
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

// extractAllText pulls all text content from messages
func extractAllText(messages []map[string]interface{}) string {
	var parts []string
	for _, msg := range messages {
		switch c := msg["content"].(type) {
		case string:
			parts = append(parts, c)
		case []interface{}:
			for _, block := range c {
				if b, ok := block.(map[string]interface{}); ok {
					if b["type"] == "text" {
						if t, ok := b["text"].(string); ok {
							parts = append(parts, t)
						}
					}
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

// estimateTokens gives a rough token count estimate
// Rule of thumb: ~4 chars per token for English
func estimateTokens(text string) int {
	charCount := utf8.RuneCountInString(text)
	return charCount / 4
}
