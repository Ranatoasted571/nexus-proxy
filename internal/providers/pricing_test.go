package providers

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestCalculateCostFull_DefaultCacheRates(t *testing.T) {
	// DeepSeek-like input price; cache read defaults to 0.1× input, write 1.25×.
	p := PricingInfo{InputPer1M: 0.27, OutputPer1M: 1.10}

	// 1M cache-read tokens cost 0.1 × 0.27 = 0.027.
	if c := p.CalculateCostFull(0, 0, 1_000_000, 0); !approx(c, 0.027) {
		t.Fatalf("cache read cost = %v, want 0.027", c)
	}
	// 1M cache-write tokens cost 1.25 × 0.27 = 0.3375.
	if c := p.CalculateCostFull(0, 0, 0, 1_000_000); !approx(c, 0.3375) {
		t.Fatalf("cache write cost = %v, want 0.3375", c)
	}
	// Plain input/output still adds on top.
	if c := p.CalculateCostFull(1_000_000, 1_000_000, 0, 0); !approx(c, 0.27+1.10) {
		t.Fatalf("plain cost = %v, want 1.37", c)
	}
}

func TestCacheReadSavings(t *testing.T) {
	p := PricingInfo{InputPer1M: 3.0, OutputPer1M: 15.0} // Anthropic Sonnet-like
	// Saved = (3.0 - 0.3) per 1M = 2.7.
	if s := p.CacheReadSavings(1_000_000); !approx(s, 2.7) {
		t.Fatalf("savings = %v, want 2.7", s)
	}
	if s := p.CacheReadSavings(0); s != 0 {
		t.Fatalf("zero savings expected, got %v", s)
	}
}

func TestCacheRateOverride(t *testing.T) {
	// Explicit cache rates win over the derived defaults.
	p := PricingInfo{InputPer1M: 1.0, OutputPer1M: 2.0, CacheReadPer1M: 0.5}
	if c := p.CalculateCostFull(0, 0, 1_000_000, 0); !approx(c, 0.5) {
		t.Fatalf("override cache read = %v, want 0.5", c)
	}
}
