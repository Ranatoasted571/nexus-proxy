package router

import (
	"strings"
	"testing"
)

func userMsg(content string) []map[string]interface{} {
	return []map[string]interface{}{{"role": "user", "content": content}}
}

func TestClassifyPolicy(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		content  string
		hasTools bool
		want     Complexity
	}{
		{"trivial tool-less", "claude-sonnet-4-6", "what is a closure?", false, ComplexitySimple},
		{"trivial with tools floors at standard", "claude-sonnet-4-6", "what is a closure?", true, ComplexityStandard},
		{"ordinary coding", "claude-sonnet-4-6", "refactor this function to be cleaner", true, ComplexityStandard},
		{"coding intent keeps off free", "claude-haiku-4-5", "fix the bug in parse()", false, ComplexityStandard},
		{"architecture is complex", "claude-haiku-4-5", "Design the system architecture for our platform", false, ComplexityComplex},
		{"security is critical", "claude-haiku-4-5", "there is a sql injection in the login form", false, ComplexityCritical},
		{"xss is critical (word boundary)", "claude-haiku-4-5", "patch the xss in comments", false, ComplexityCritical},
		{"opus pick is complex", "claude-opus-4-5", "rename this variable", true, ComplexityComplex},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyRequest(tc.model, userMsg(tc.content), tc.hasTools); got != tc.want {
				t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestClassifyLargeContextIsComplex(t *testing.T) {
	big := strings.Repeat("some context line about the code base ", 500) // ~4.7k tokens
	if got := ClassifyRequest("claude-haiku-4-5", userMsg(big), false); got != ComplexityComplex {
		t.Errorf("large context: got %s, want complex", got)
	}
}

func TestClassifyNoCriticalFalsePositive(t *testing.T) {
	// "across" / "assess" contain no standalone xss/csrf token.
	if got := ClassifyRequest("claude-haiku-4-5", userMsg("print the list across pages"), false); got == ComplexityCritical {
		t.Errorf("benign text should not be critical, got %s", got)
	}
}

func TestClassifyLastUserTextSkipsAssistant(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "user", "content": "earlier question"},
		{"role": "assistant", "content": "here is a long answer about system architecture and design patterns"},
		{"role": "user", "content": "thanks"},
	}
	// The last *user* turn is the short "thanks"; the assistant's architecture talk
	// must not pull this to complex (it has no tools/code).
	if got := ClassifyRequest("claude-sonnet-4-6", msgs, false); got == ComplexityComplex {
		t.Errorf("should not classify complex from assistant text, got %s", got)
	}
}
