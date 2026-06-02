package proxy

import (
	"encoding/json"
	"regexp"
	"strconv"
)

// tokenUsage is a normalized token breakdown across providers.
//
//   - In         — full-price input tokens
//   - Out        — output tokens
//   - CacheRead  — tokens served from the provider's prompt cache (billed at the
//     cache-read discount, typically ~0.1× input)
//   - CacheWrite — tokens written into the provider's prompt cache (billed at the
//     cache-write premium, ~1.25× input for Anthropic)
//
// For Anthropic, `input_tokens` already excludes cached tokens, so In maps
// straight across. For OpenAI-compatible providers the cached tokens are part of
// `prompt_tokens`, so we subtract them out — In is always the full-price portion.
type tokenUsage struct {
	In, Out, CacheRead, CacheWrite int
}

// billable is the total token count touched (for logging/aggregation).
func (u tokenUsage) billable() int { return u.In + u.Out + u.CacheRead + u.CacheWrite }

var (
	reCacheRead     = regexp.MustCompile(`"cache_read_input_tokens":\s*(\d+)`)
	reCacheCreation = regexp.MustCompile(`"cache_creation_input_tokens":\s*(\d+)`)
	reOAICacheHit   = regexp.MustCompile(`"prompt_cache_hit_tokens":\s*(\d+)`) // DeepSeek
	reOAICached     = regexp.MustCompile(`"cached_tokens":\s*(\d+)`)           // OpenAI prompt_tokens_details
)

func atoiBytes(b []byte) int { n, _ := strconv.Atoi(string(b)); return n }

// anthropicUsageFull parses a non-streaming Anthropic response body.
func anthropicUsageFull(body []byte) tokenUsage {
	var r struct {
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &r)
	return tokenUsage{
		In:         r.Usage.InputTokens,
		Out:        r.Usage.OutputTokens,
		CacheRead:  r.Usage.CacheReadInputTokens,
		CacheWrite: r.Usage.CacheCreationInputTokens,
	}
}

// streamUsageFull scrapes token counts from a captured Anthropic SSE stream.
// input/cache counts appear once (message_start); output_tokens appears multiple
// times (message_delta), so we take the final value.
func streamUsageFull(data []byte) tokenUsage {
	u := tokenUsage{}
	if m := reInputTokens.FindSubmatch(data); m != nil {
		u.In = atoiBytes(m[1])
	}
	if all := reOutputTokens.FindAllSubmatch(data, -1); len(all) > 0 {
		u.Out = atoiBytes(all[len(all)-1][1])
	}
	if m := reCacheRead.FindSubmatch(data); m != nil {
		u.CacheRead = atoiBytes(m[1])
	}
	if m := reCacheCreation.FindSubmatch(data); m != nil {
		u.CacheWrite = atoiBytes(m[1])
	}
	return u
}

// openAIUsageFull parses an OpenAI/DeepSeek response (full body or captured
// stream). DeepSeek reports prompt_cache_hit_tokens; OpenAI reports
// cached_tokens inside prompt_tokens_details. Both are part of prompt_tokens, so
// we move them into CacheRead and leave In as the uncached remainder.
func openAIUsageFull(data []byte) tokenUsage {
	in, out := parseOpenAIUsage(data)
	cached := 0
	if m := reOAICacheHit.FindSubmatch(data); m != nil {
		cached = atoiBytes(m[1])
	} else if m := reOAICached.FindSubmatch(data); m != nil {
		cached = atoiBytes(m[1])
	}
	if cached > in {
		cached = in
	}
	return tokenUsage{In: in - cached, Out: out, CacheRead: cached}
}
