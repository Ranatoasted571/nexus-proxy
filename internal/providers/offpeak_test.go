package providers

import (
	"testing"
	"time"
)

func TestOffPeakPricing(t *testing.T) {
	// Off-peak window [1,5) UTC; off-peak input is half price.
	p := PricingInfo{InputPer1M: 1.0, OutputPer1M: 2.0, OffPeakInputPer1M: 0.5, OffPeakOutputPer1M: 1.0, OffPeakStartUTC: 1, OffPeakEndUTC: 5}
	off := time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)  // inside window
	peak := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // outside

	if c := p.CalculateCostFullAt(1_000_000, 0, 0, 0, off); !approx(c, 0.5) {
		t.Errorf("off-peak 1M input = %v, want 0.5", c)
	}
	if c := p.CalculateCostFullAt(1_000_000, 0, 0, 0, peak); !approx(c, 1.0) {
		t.Errorf("peak 1M input = %v, want 1.0", c)
	}
	if c := p.CalculateCostFullAt(0, 1_000_000, 0, 0, off); !approx(c, 1.0) {
		t.Errorf("off-peak 1M output = %v, want 1.0", c)
	}
}

func TestOffPeakWrapsMidnight(t *testing.T) {
	p := PricingInfo{OffPeakStartUTC: 22, OffPeakEndUTC: 2} // 22:00 → 02:00 UTC
	if !p.inOffPeak(time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)) {
		t.Error("23:00 should be off-peak")
	}
	if !p.inOffPeak(time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)) {
		t.Error("01:00 should be off-peak")
	}
	if p.inOffPeak(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Error("12:00 should be peak")
	}
}

func TestNoOffPeakWindow(t *testing.T) {
	p := PricingInfo{InputPer1M: 1.0, OffPeakInputPer1M: 0.5} // start==end==0 → no window
	if c := p.CalculateCostFullAt(1_000_000, 0, 0, 0, time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC)); !approx(c, 1.0) {
		t.Errorf("no window should always use peak, got %v", c)
	}
}
