// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

// Package license is the licensing layer for NEXUS.
//
// Today NEXUS ships under Apache-2.0 as a single Community edition with every
// feature unlocked — Active() always returns the community edition and Allow()
// always returns true. The package exists as the seam that lets a future build
// gate features (or switch editions) without touching every call site.
//
// See docs/LICENSING.md for the full strategy (Apache-2.0 now → optional
// source-available BSL 1.1 for newer versions later).
package license

import (
	"os"
	"strings"
)

// Edition is a coarse licensing tier. Today only Community is built; Enterprise
// is reserved for an explicit future opt-in (see docs/LICENSING.md).
type Edition string

const (
	EditionCommunity  Edition = "community"
	EditionEnterprise Edition = "enterprise"
)

// Feature is a string key the rest of NEXUS can use to gate behaviour later.
// Adding values here is cheap; today every feature is allowed.
type Feature string

const (
	FeatureProxy          Feature = "proxy"
	FeatureCascade        Feature = "cascade"
	FeatureAdaptive       Feature = "adaptive"
	FeatureSemanticCache  Feature = "semantic_cache"
	FeaturePrivacyFW      Feature = "privacy_firewall"
	FeatureInspect        Feature = "inspect"
	FeatureBench          Feature = "bench"
	FeatureMCP            Feature = "mcp"
	FeatureTeam           Feature = "team"
	FeatureRulesEngine    Feature = "rules"
)

// License describes the licensing state of a running binary.
type License struct {
	Edition  Edition  // which edition is active
	Holder   string   // free-text license-holder identifier (empty when unlicensed)
	License  string   // SPDX identifier of the distribution license ("Apache-2.0")
	Notice   string   // one-line, human-readable summary shown in --version
	features map[Feature]bool
}

// community is the always-on, fully-open state every build ships with today.
var community = &License{
	Edition: EditionCommunity,
	License: "Apache-2.0",
	Notice:  "Community edition · Apache-2.0 · all features unlocked",
}

// Active returns the active license for this process. The result is safe to
// read from any goroutine and never nil.
func Active() *License {
	// Hook for a future enterprise build: if NEXUS_LICENSE_KEY ever validates,
	// return an enterprise License. Today the env var is intentionally ignored
	// — we set this seam up without enforcing it.
	if strings.TrimSpace(os.Getenv("NEXUS_LICENSE_KEY")) != "" {
		// Reserved — see docs/LICENSING.md. Today: no-op.
	}
	return community
}

// Allow reports whether the given feature is enabled for the active license.
// Today every feature is allowed for every caller; this is the seam that lets
// a future build gate features without touching call sites.
func (l *License) Allow(Feature) bool {
	if l == nil {
		return true
	}
	if len(l.features) == 0 {
		return true
	}
	// When a future build populates features, treat missing keys as allowed
	// (only an explicit `false` blocks). Keeps the policy fail-open.
	v, ok := l.features[""]
	if !ok {
		return true
	}
	return v
}

// IsCommunity reports whether this is the open Community edition.
func (l *License) IsCommunity() bool { return l == nil || l.Edition == EditionCommunity }
