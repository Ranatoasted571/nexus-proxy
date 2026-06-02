package config

import "testing"

func TestDiscoverFromEnv(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "gsk_testkey")
	t.Setenv("ANTHROPIC_API_KEY", "nexus-local") // placeholder → must be skipped

	disc := DiscoverFromEnv(nil)
	found := map[string]string{}
	for _, p := range disc {
		found[p.Name] = p.APIKey
	}
	if found["groq"] != "env:GROQ_API_KEY" {
		t.Errorf("groq should be discovered as env:GROQ_API_KEY, got %q", found["groq"])
	}
	if _, ok := found["anthropic"]; ok {
		t.Error("ANTHROPIC_API_KEY=nexus-local must not be discovered")
	}
}

func TestDiscoverSkipsExisting(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "gsk_testkey")
	disc := DiscoverFromEnv([]Provider{{Name: "groq"}})
	for _, p := range disc {
		if p.Name == "groq" {
			t.Error("already-configured groq must not be re-discovered")
		}
	}
}
