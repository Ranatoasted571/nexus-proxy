package proxy

import "testing"

func TestTransformToOpenAI_StringMessages(t *testing.T) {
	req := AnthropicRequest{
		Model:     "claude-haiku-4-5",
		MaxTokens: 100,
		Messages:  []Message{{Role: "user", Content: "hello"}},
	}
	oai, err := TransformToOpenAI(req, "llama-3.3-70b-versatile")
	if err != nil {
		t.Fatal(err)
	}
	if oai.Model != "llama-3.3-70b-versatile" {
		t.Errorf("model = %q", oai.Model)
	}
	if len(oai.Messages) != 1 || oai.Messages[0].Role != "user" || oai.Messages[0].Content != "hello" {
		t.Errorf("messages = %+v", oai.Messages)
	}
}

func TestTransformToOpenAI_SystemString(t *testing.T) {
	req := AnthropicRequest{
		Model:    "m",
		System:   "be helpful",
		Messages: []Message{{Role: "user", Content: "hi"}},
	}
	oai, _ := TransformToOpenAI(req, "x")
	if len(oai.Messages) != 2 || oai.Messages[0].Role != "system" || oai.Messages[0].Content != "be helpful" {
		t.Errorf("system message not prepended: %+v", oai.Messages)
	}
}

func TestTransformToOpenAI_ContentBlocks(t *testing.T) {
	req := AnthropicRequest{
		Model: "m",
		Messages: []Message{{Role: "user", Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "part1 "},
			map[string]interface{}{"type": "text", "text": "part2"},
		}}},
	}
	oai, _ := TransformToOpenAI(req, "x")
	if oai.Messages[0].Content != "part1 part2" {
		t.Errorf("content blocks not concatenated: %q", oai.Messages[0].Content)
	}
}

func TestMapStopReason(t *testing.T) {
	cases := map[string]string{
		"stop":       "end_turn",
		"length":     "max_tokens",
		"tool_calls": "tool_use",
		"unexpected": "end_turn",
	}
	for in, want := range cases {
		if got := mapStopReason(in); got != want {
			t.Errorf("mapStopReason(%q) = %q, want %q", in, got, want)
		}
	}
}
