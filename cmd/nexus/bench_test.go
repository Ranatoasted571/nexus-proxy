package main

import "testing"

func TestCosineWords(t *testing.T) {
	if c := cosineWords("the quick brown fox", "the quick brown fox"); c < 0.999 {
		t.Errorf("identical text should be ~1.0, got %v", c)
	}
	if c := cosineWords("the quick brown fox", "completely unrelated phrase here"); c > 0.4 {
		t.Errorf("unrelated text should be low, got %v", c)
	}
	if c := cosineWords("", "anything"); c != 0 {
		t.Errorf("empty should be 0, got %v", c)
	}
}

func TestExtractText(t *testing.T) {
	if got := extractText(`{"content":[{"type":"text","text":"hello"},{"type":"text","text":" world"}]}`); got != "hello world" {
		t.Errorf("anthropic extract = %q", got)
	}
	if got := extractText(`{"choices":[{"message":{"role":"assistant","content":"hi there"}}]}`); got != "hi there" {
		t.Errorf("openai extract = %q", got)
	}
	if got := extractText(`not json`); got != "" {
		t.Errorf("invalid json should yield empty, got %q", got)
	}
}

func TestAvg(t *testing.T) {
	if avg(10, 0) != 0 {
		t.Error("avg with n=0 should be 0")
	}
	if avg(10, 4) != 2.5 {
		t.Errorf("avg = %v", avg(10, 4))
	}
}
