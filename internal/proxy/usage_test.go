package proxy

import "testing"

func TestAnthropicUsageFull(t *testing.T) {
	body := []byte(`{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":900,"cache_creation_input_tokens":200}}`)
	u := anthropicUsageFull(body)
	if u.In != 100 || u.Out != 50 || u.CacheRead != 900 || u.CacheWrite != 200 {
		t.Fatalf("got %+v", u)
	}
}

func TestOpenAIUsageFull_DeepSeek(t *testing.T) {
	// DeepSeek reports prompt_cache_hit_tokens; it is part of prompt_tokens.
	body := []byte(`{"usage":{"prompt_tokens":1000,"completion_tokens":50,"prompt_cache_hit_tokens":800}}`)
	u := openAIUsageFull(body)
	if u.In != 200 || u.Out != 50 || u.CacheRead != 800 {
		t.Fatalf("got %+v", u)
	}
}

func TestOpenAIUsageFull_OpenAI(t *testing.T) {
	// OpenAI reports cached_tokens inside prompt_tokens_details.
	body := []byte(`{"usage":{"prompt_tokens":1000,"completion_tokens":50,"prompt_tokens_details":{"cached_tokens":640}}}`)
	u := openAIUsageFull(body)
	if u.In != 360 || u.Out != 50 || u.CacheRead != 640 {
		t.Fatalf("got %+v", u)
	}
}

func TestOpenAIUsageFull_NoCache(t *testing.T) {
	body := []byte(`{"usage":{"prompt_tokens":300,"completion_tokens":40}}`)
	u := openAIUsageFull(body)
	if u.In != 300 || u.Out != 40 || u.CacheRead != 0 {
		t.Fatalf("got %+v", u)
	}
}

func TestStreamUsageFull(t *testing.T) {
	// message_start carries input + cache; message_delta carries output (twice).
	stream := []byte(`event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":120,"cache_read_input_tokens":4000,"cache_creation_input_tokens":300,"output_tokens":0}}}

event: message_delta
data: {"type":"message_delta","usage":{"output_tokens":10}}

event: message_delta
data: {"type":"message_delta","usage":{"output_tokens":85}}
`)
	u := streamUsageFull(stream)
	if u.In != 120 || u.Out != 85 || u.CacheRead != 4000 || u.CacheWrite != 300 {
		t.Fatalf("got %+v", u)
	}
}
