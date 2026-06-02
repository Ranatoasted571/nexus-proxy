package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

// bench command — replay your real captured traffic across every provider and
// report cost × latency × agreement, with a recommendation.
var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark every provider on your real captured traffic (needs --inspect history)",
	RunE:  runBench,
}

var (
	flagBenchPort      int
	flagBenchN         int
	flagBenchProviders string
	flagBenchPrompt    string
)

func runBench(cmd *cobra.Command, args []string) error {
	base := fmt.Sprintf("http://localhost:%d", flagBenchPort)
	if !proxyHealthy(base) {
		return fmt.Errorf("NEXUS isn't running on :%d — start it with `nexus start`", flagBenchPort)
	}
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return err
	}
	defer db.Close()
	cfg, _ := config.Load(config.DefaultPath())

	// Which providers to bench.
	var provNames []string
	if flagBenchProviders != "" {
		for _, n := range strings.Split(flagBenchProviders, ",") {
			if n = strings.TrimSpace(n); n != "" {
				provNames = append(provNames, n)
			}
		}
	} else {
		for _, p := range cfg.Providers {
			provNames = append(provNames, p.Name)
		}
	}
	if len(provNames) == 0 {
		return fmt.Errorf("no providers configured — add some with `nexus add`")
	}

	pricing := map[string]providers.PricingInfo{}
	for _, pc := range cfg.Providers {
		if impl, e := providers.New(specFromConfig(pc)); e == nil {
			pricing[pc.Name] = impl.Pricing()
		}
	}

	// Build the sample set: an ad-hoc prompt, or the most recent inspected requests.
	type sample struct{ prompt, reference string }
	var samples []sample
	if flagBenchPrompt != "" {
		q, _ := json.Marshal(flagBenchPrompt)
		samples = append(samples, sample{prompt: `{"model":"claude-sonnet-4-6","max_tokens":512,"messages":[{"role":"user","content":` + string(q) + `}]}`})
	} else {
		recents, _ := db.GetRecentRequests(300)
		for _, r := range recents {
			d, e := db.GetRequestDetail(r.ID)
			if e != nil || d.Prompt == "" {
				continue
			}
			samples = append(samples, sample{prompt: d.Prompt, reference: extractText(d.Response)})
			if len(samples) >= flagBenchN {
				break
			}
		}
	}
	if len(samples) == 0 {
		return fmt.Errorf("no captured requests to benchmark — run `nexus start --inspect` for a while, or pass --prompt \"...\"")
	}

	type agg struct {
		n              int
		cost, latency  float64
		toks           int
		agree          float64
		agreeN         int
	}
	res := map[string]*agg{}
	for _, n := range provNames {
		res[n] = &agg{}
	}

	color.Cyan("Benchmarking %d request(s) × %d provider(s) on :%d …\n", len(samples), len(provNames), flagBenchPort)
	for _, s := range samples {
		for _, name := range provNames {
			out, in, outTok, lat, e := benchOne(base, s.prompt, name)
			if e != nil {
				continue
			}
			a := res[name]
			a.n++
			if pr, ok := pricing[name]; ok {
				a.cost += pr.CalculateCost(in, outTok)
			}
			a.latency += float64(lat)
			a.toks += outTok
			if s.reference != "" {
				a.agree += cosineWords(out, s.reference)
				a.agreeN++
			}
		}
	}

	// Sort by average cost ascending.
	sort.Slice(provNames, func(i, j int) bool {
		return avg(res[provNames[i]].cost, res[provNames[i]].n) < avg(res[provNames[j]].cost, res[provNames[j]].n)
	})

	color.Cyan("\n  📊 Provider Report Card — benchmarked on %d of your real request(s)\n", len(samples))
	color.White("  %-12s %4s %12s %9s %8s %11s", "PROVIDER", "OK", "avg cost", "avg lat", "out tok", "agreement")
	color.White("  %s", strings.Repeat("─", 62))

	bestValue, bestAgreeName := "", ""
	bestAgree := -1.0
	haveAgreement := false
	for _, name := range provNames {
		a := res[name]
		if a.n == 0 {
			color.Red("  %-12s %4s   (no successful responses)", name, "0")
			continue
		}
		agreeStr, agreePct := "—", -1.0
		if a.agreeN > 0 {
			agreePct = a.agree / float64(a.agreeN) * 100
			agreeStr = fmt.Sprintf("%.0f%%", agreePct)
			haveAgreement = true
			if agreePct > bestAgree {
				bestAgree, bestAgreeName = agreePct, name
			}
		}
		color.White("  %-12s %4d %11s$ %7.0fms %8.0f %11s",
			name, a.n, fmt.Sprintf("%.5f", avg(a.cost, a.n)), avg(a.latency, a.n), float64(a.toks)/float64(a.n), agreeStr)
		if bestValue == "" && (agreePct < 0 || agreePct >= 70) {
			bestValue = name // cheapest provider (sorted) that stays close enough
		}
	}
	if bestValue == "" && len(provNames) > 0 {
		bestValue = provNames[0] // fall back to cheapest overall
	}

	fmt.Println()
	if bestValue != "" {
		color.Green("  🏆 Best value: %s — the cheapest provider that stays close to your originals.", bestValue)
	}
	if haveAgreement && bestAgreeName != "" {
		color.White("  🎯 Closest to your original outputs: %s (%.0f%% agreement).", bestAgreeName, bestAgree)
	}
	color.HiBlack("\n  agreement = word-overlap vs your original captured response (a rough quality proxy).")
	if !haveAgreement {
		color.HiBlack("  Tip: run `nexus start --inspect` for a while so bench can score agreement on your own traffic.")
	}
	return nil
}

// benchOne replays one prompt against a pinned provider and returns its output
// text, input/output tokens and latency (ms).
func benchOne(base, prompt, provider string) (out string, in, outTok int, latencyMS int64, err error) {
	var pm map[string]interface{}
	if json.Unmarshal([]byte(prompt), &pm) == nil {
		pm["stream"] = false
		if b, e := json.Marshal(pm); e == nil {
			prompt = string(b)
		}
	}
	req, _ := http.NewRequest("POST", base+"/v1/messages", bytes.NewReader([]byte(prompt)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", "nexus-local")
	req.Header.Set("X-Nexus-Provider", provider)
	start := time.Now()
	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		return "", 0, 0, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", 0, 0, 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	var ar struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(raw, &ar)
	var sb strings.Builder
	for _, c := range ar.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), ar.Usage.InputTokens, ar.Usage.OutputTokens, time.Since(start).Milliseconds(), nil
}

func avg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func extractText(body string) string {
	var m map[string]interface{}
	if json.Unmarshal([]byte(body), &m) != nil {
		return ""
	}
	var sb strings.Builder
	if content, ok := m["content"].([]interface{}); ok { // anthropic
		for _, c := range content {
			if cm, ok := c.(map[string]interface{}); ok {
				if t, ok := cm["text"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
	}
	if choices, ok := m["choices"].([]interface{}); ok && sb.Len() == 0 { // openai
		if c0, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := c0["message"].(map[string]interface{}); ok {
				if t, ok := msg["content"].(string); ok {
					sb.WriteString(t)
				}
			}
		}
	}
	return sb.String()
}

// cosineWords is a dependency-free word-frequency cosine similarity (0..1),
// used as a rough "agreement" proxy between two outputs.
func cosineWords(a, b string) float64 {
	va, vb := wordFreq(a), wordFreq(b)
	if len(va) == 0 || len(vb) == 0 {
		return 0
	}
	var dot, na, nb float64
	for w, c := range va {
		na += c * c
		if d, ok := vb[w]; ok {
			dot += c * d
		}
	}
	for _, c := range vb {
		nb += c * c
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func wordFreq(s string) map[string]float64 {
	m := map[string]float64{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		m[w]++
	}
	return m
}
