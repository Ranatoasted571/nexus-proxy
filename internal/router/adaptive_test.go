package router

import "testing"

func adaptiveRouter() *Router {
	r := New(StrategyAuto)
	r.AddProvider(&Provider{Name: "a", Tier: "free", Healthy: true, Priority: 0})
	r.AddProvider(&Provider{Name: "b", Tier: "free", Healthy: true, Priority: 1})
	return r
}

func TestAdaptiveReordersWithinTier(t *testing.T) {
	r := adaptiveRouter()
	r.SetAdaptive(true)
	for i := 0; i < 5; i++ {
		r.RecordOutcome("b", ComplexityStandard, true)  // b is reliable
		r.RecordOutcome("a", ComplexityStandard, false) // a keeps failing
	}
	chain := r.CascadeChain(ComplexityStandard)
	if chain[0].Name != "b" {
		t.Errorf("adaptive should put the higher-success provider (b) first, got %s", chain[0].Name)
	}
}

func TestAdaptiveOffKeepsPriorityOrder(t *testing.T) {
	r := adaptiveRouter() // adaptive NOT enabled
	for i := 0; i < 5; i++ {
		r.RecordOutcome("b", ComplexityStandard, true)
		r.RecordOutcome("a", ComplexityStandard, false)
	}
	chain := r.CascadeChain(ComplexityStandard)
	if chain[0].Name != "a" {
		t.Errorf("without adaptive, priority order should win (a first), got %s", chain[0].Name)
	}
}

func TestSuccessRateNeutralPrior(t *testing.T) {
	r := adaptiveRouter()
	r.SetAdaptive(true)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if got := r.successRate("never-seen", ComplexitySimple); got != 0.5 {
		t.Errorf("unseen provider should have a neutral 0.5 prior, got %v", got)
	}
}
