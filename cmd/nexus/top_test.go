package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRenderTopWaitingWhenDown(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	out := renderTop(client, "http://127.0.0.1:1") // nothing listening
	if !strings.Contains(out, "waiting for NEXUS") {
		t.Errorf("expected a waiting message when the dashboard is down, got:\n%s", out)
	}
}

func TestTrunc(t *testing.T) {
	if got := trunc("anthropic", 10); got != "anthropic" {
		t.Errorf("short string should be unchanged, got %q", got)
	}
	if got := trunc("claude-sonnet-4-6-extended", 12); got != "claude-sonn~" {
		t.Errorf("long string should truncate with ~, got %q", got)
	}
}

func TestHumanInt(t *testing.T) {
	if got := humanInt(950); got != "950" {
		t.Errorf("got %q", got)
	}
	if got := humanInt(205200); got != "205.2K" {
		t.Errorf("got %q", got)
	}
}

func TestTierColor(t *testing.T) {
	if tierColor("free") != aGreen || tierColor("premium") != aPurple || tierColor("standard") != aCyan {
		t.Error("tier colors mismatched")
	}
}
