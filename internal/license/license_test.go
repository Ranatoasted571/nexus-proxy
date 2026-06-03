// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

package license

import "testing"

func TestActiveIsCommunityOpen(t *testing.T) {
	l := Active()
	if l == nil {
		t.Fatal("Active() returned nil")
	}
	if !l.IsCommunity() {
		t.Errorf("edition = %s, want community", l.Edition)
	}
	if l.License != "Apache-2.0" {
		t.Errorf("SPDX = %q, want Apache-2.0", l.License)
	}
	if l.Notice == "" {
		t.Error("Notice should be a non-empty human summary")
	}
}

func TestAllowAllFeaturesToday(t *testing.T) {
	l := Active()
	for _, f := range []Feature{
		FeatureProxy, FeatureCascade, FeatureAdaptive, FeatureSemanticCache,
		FeaturePrivacyFW, FeatureInspect, FeatureBench, FeatureMCP, FeatureTeam, FeatureRulesEngine,
	} {
		if !l.Allow(f) {
			t.Errorf("Allow(%q) = false, want true (community is fully open today)", f)
		}
	}
}

// The NEXUS_LICENSE_KEY env var is reserved for a future enterprise build.
// Today it must never change behaviour — confirm that explicitly.
func TestLicenseKeyEnvIsNoopToday(t *testing.T) {
	t.Setenv("NEXUS_LICENSE_KEY", "anything-here-2026")
	l := Active()
	if !l.IsCommunity() {
		t.Errorf("env var must not change edition today; got %s", l.Edition)
	}
	if !l.Allow(FeatureCascade) {
		t.Error("env var must not gate features today")
	}
}

func TestNilLicenseAllowsEverything(t *testing.T) {
	var l *License
	if !l.IsCommunity() {
		t.Error("nil license should report as community")
	}
	if !l.Allow(FeatureProxy) {
		t.Error("nil license should allow features (fail-open)")
	}
}
