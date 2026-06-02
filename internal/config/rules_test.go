package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRulesParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	content := `
[routing]
max_request_usd = 0.25

[[rules]]
when_prompt_contains = "production"
use_provider = "anthropic"

[[rules]]
when_model_contains = "haiku"
use_tier = "free"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].WhenPromptContains != "production" || cfg.Rules[0].UseProvider != "anthropic" {
		t.Errorf("rule 0 = %+v", cfg.Rules[0])
	}
	if cfg.Rules[1].UseTier != "free" {
		t.Errorf("rule 1 = %+v", cfg.Rules[1])
	}
	if cfg.Routing.MaxRequestUSD != 0.25 {
		t.Errorf("max_request_usd = %v", cfg.Routing.MaxRequestUSD)
	}
}
