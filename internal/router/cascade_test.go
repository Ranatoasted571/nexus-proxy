package router

import "testing"

func cascadeRouter() *Router {
	r := New(StrategyAuto)
	r.AddProvider(&Provider{Name: "claude", Tier: "premium", Healthy: true})
	r.AddProvider(&Provider{Name: "groq", Tier: "free", Healthy: true})
	r.AddProvider(&Provider{Name: "deepseek", Tier: "standard", Healthy: true})
	r.AddProvider(&Provider{Name: "ollama", Tier: "local", Healthy: true})
	return r
}

func TestCascadeChainCheapFirst(t *testing.T) {
	chain := cascadeRouter().CascadeChain(ComplexityStandard)
	want := []string{"ollama", "groq", "deepseek", "claude"}
	if len(chain) != len(want) {
		t.Fatalf("chain len = %d, want %d", len(chain), len(want))
	}
	for i, name := range want {
		if chain[i].Name != name {
			t.Errorf("position %d = %q, want %q (cheap-first order)", i, chain[i].Name, name)
		}
	}
}

func TestCascadeChainCriticalStaysPremiumFirst(t *testing.T) {
	chain := cascadeRouter().CascadeChain(ComplexityCritical)
	if len(chain) == 0 || chain[0].Tier != "premium" {
		t.Fatalf("critical requests must start premium-first, got %+v", chain)
	}
}
