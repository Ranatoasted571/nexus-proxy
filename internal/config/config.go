package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the on-disk NEXUS configuration (~/.nexus/config.toml).
type Config struct {
	Proxy     Proxy      `toml:"proxy"`
	Dashboard Dashboard  `toml:"dashboard"`
	Routing   Routing    `toml:"routing"`
	Providers []Provider `toml:"providers"`
}

type Proxy struct {
	Port int `toml:"port"`
}

type Dashboard struct {
	Port int `toml:"port"`
}

type Routing struct {
	Strategy          string  `toml:"strategy"`                    // auto | manual | cheapest | fastest
	DailyBudgetUSD    float64 `toml:"daily_budget_usd,omitempty"`  // 0 = unlimited; over budget → free/local only
	SemanticCache     bool    `toml:"semantic_cache,omitempty"`    // near-match response caching for tool-less requests
	SemanticThreshold float64 `toml:"semantic_threshold,omitempty"` // cosine threshold (0 ⇒ 0.95)
	Cascade           bool    `toml:"cascade,omitempty"`           // cheap-first cascade with verification (feature 3)
}

// Provider is a single configured LLM provider.
type Provider struct {
	Name    string `toml:"name"`
	Type    string `toml:"type,omitempty"`     // "openai-compatible" for a custom endpoint; empty = built-in by name
	APIKey  string `toml:"api_key,omitempty"`  // literal, or "env:VAR_NAME" to read from the environment
	BaseURL string `toml:"base_url,omitempty"` // required for custom/ollama providers
	Models  []string `toml:"models,omitempty"`
	Tier    string   `toml:"tier,omitempty"`

	// ModelMap optionally overrides which provider model a Claude model maps to,
	// e.g. {"claude-sonnet-4-6" = "llama-3.3-70b"}. Use "default" as a catch-all.
	ModelMap    map[string]string `toml:"model_map,omitempty"`
	InputPer1M  float64           `toml:"input_per_1m,omitempty"`  // optional pricing override (USD/1M)
	OutputPer1M float64           `toml:"output_per_1m,omitempty"` // optional pricing override (USD/1M)

	// Enterprise providers:
	Region     string `toml:"region,omitempty"`      // AWS Bedrock / Google Vertex region
	Project    string `toml:"project,omitempty"`     // Google Vertex project ID
	APIVersion string `toml:"api_version,omitempty"` // Azure OpenAI api-version
}

// DefaultPath returns ~/.nexus/config.toml.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "config.toml")
}

// Default returns a config with sensible zero-config defaults.
func Default() *Config {
	return &Config{
		Proxy:     Proxy{Port: 3000},
		Dashboard: Dashboard{Port: 2222},
		Routing:   Routing{Strategy: "auto"},
	}
}

// Load reads the config from path (or DefaultPath if empty). A missing file is
// not an error — defaults are returned so NEXUS works with zero config.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	cfg := Default()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	if cfg.Proxy.Port == 0 {
		cfg.Proxy.Port = 3000
	}
	if cfg.Dashboard.Port == 0 {
		cfg.Dashboard.Port = 2222
	}
	if cfg.Routing.Strategy == "" {
		cfg.Routing.Strategy = "auto"
	}
	return cfg, nil
}

// Save writes the config to path (or DefaultPath if empty), creating the
// parent directory if needed.
func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// Upsert adds a provider, or replaces an existing one with the same name.
func (c *Config) Upsert(p Provider) {
	for i, existing := range c.Providers {
		if strings.EqualFold(existing.Name, p.Name) {
			c.Providers[i] = p
			return
		}
	}
	c.Providers = append(c.Providers, p)
}

// ResolveKey resolves an api_key value: "env:VAR" reads VAR from the
// environment; anything else is returned verbatim.
func ResolveKey(v string) string {
	if strings.HasPrefix(v, "env:") {
		return os.Getenv(strings.TrimPrefix(v, "env:"))
	}
	return v
}
