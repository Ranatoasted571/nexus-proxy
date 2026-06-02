package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// ANSI styling (raw codes keep `nexus top` dependency-free and portable).
const (
	aReset  = "\033[0m"
	aBold   = "\033[1m"
	aDim    = "\033[2m"
	aPurple = "\033[38;5;141m"
	aCyan   = "\033[38;5;44m"
	aGreen  = "\033[38;5;42m"
	aYellow = "\033[38;5;214m"
	aRed    = "\033[38;5;203m"
	aGray   = "\033[38;5;245m"
)

type topStats struct {
	TotalRequests int     `json:"total_requests"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	TotalTokens   int     `json:"total_tokens"`
	ForecastUSD   float64 `json:"forecast_usd"`
	AvgLatencyMS  float64 `json:"avg_latency_ms"`
	CacheSavedUSD float64 `json:"cache_saved_usd"`
}

type topReq struct {
	Provider     string  `json:"provider"`
	ModelUsed    string  `json:"model_used"`
	Complexity   string  `json:"complexity"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	LatencyMS    int64   `json:"latency_ms"`
	Status       int     `json:"status"`
}

func getJSON(client *http.Client, url string, v interface{}) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

func runTop(cmd *cobra.Command, args []string) error {
	base := fmt.Sprintf("http://localhost:%d", flagTopPort)
	client := &http.Client{Timeout: 3 * time.Second}

	if flagTopOnce {
		fmt.Print(renderTop(client, base))
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Print("\033[?25l") // hide cursor
	restore := func() { fmt.Print("\033[?25h" + aReset) }
	defer restore()

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	draw := func() { fmt.Print("\033[H\033[2J" + renderTop(client, base)) }
	draw()
	for {
		select {
		case <-ctx.Done():
			restore()
			fmt.Println()
			return nil
		case <-ticker.C:
			draw()
		}
	}
}

func renderTop(client *http.Client, base string) string {
	var b strings.Builder
	line := aGray + strings.Repeat("─", 78) + aReset

	fmt.Fprintf(&b, " %s%sNEXUS%s %stop%s   %s%s%s\n", aBold, aPurple, aReset, aCyan, aReset, aGray, base, aReset)
	fmt.Fprintln(&b, line)

	var st topStats
	if err := getJSON(client, base+"/api/stats", &st); err != nil {
		fmt.Fprintf(&b, " %s⚠ waiting for NEXUS at %s — is it running?  (nexus start)%s\n", aYellow, base, aReset)
		return b.String()
	}

	fmt.Fprintf(&b, " %srequests%s %s%-7d%s  %scost%s %s$%-9.4f%s  %scache saved%s %s$%-9.4f%s\n",
		aGray, aReset, aBold, st.TotalRequests, aReset,
		aGray, aReset, aGreen, st.TotalCostUSD, aReset,
		aGray, aReset, aCyan, st.CacheSavedUSD, aReset)
	fmt.Fprintf(&b, " %stokens%s   %s%-7s%s  %sfcast/mo%s $%-6.2f  %savg lat%s %.0fms\n",
		aGray, aReset, aBold, humanInt(st.TotalTokens), aReset,
		aGray, aReset, st.ForecastUSD, aGray, aReset, st.AvgLatencyMS)

	// providers
	var pv struct {
		Providers []struct {
			Name string `json:"name"`
			Tier string `json:"tier"`
		} `json:"providers"`
	}
	_ = getJSON(client, base+"/api/providers", &pv)
	fmt.Fprintln(&b, line)
	if len(pv.Providers) == 0 {
		fmt.Fprintf(&b, " %sPROVIDERS%s %snone — add one: nexus add groq <key>%s\n", aGray, aReset, aYellow, aReset)
	} else {
		names := make([]string, 0, len(pv.Providers))
		for _, p := range pv.Providers {
			names = append(names, tierColor(p.Tier)+p.Name+aReset)
		}
		fmt.Fprintf(&b, " %sPROVIDERS%s %s\n", aGray, aReset, strings.Join(names, "  "))
	}

	// recent requests
	var rw struct {
		Requests []topReq `json:"requests"`
	}
	_ = getJSON(client, base+"/api/requests", &rw)
	fmt.Fprintln(&b, line)
	fmt.Fprintf(&b, " %s%-10s %-22s %-9s %8s %10s %7s%s\n", aGray, "PROVIDER", "MODEL", "TASK", "TOKENS", "COST", "LAT", aReset)

	if len(rw.Requests) == 0 {
		fmt.Fprintf(&b, " %sno requests yet — point Claude Code at NEXUS and go%s\n", aDim, aReset)
	}
	for i, q := range rw.Requests {
		if i >= 12 {
			break
		}
		stCol := aGreen
		if q.Status >= 400 {
			stCol = aRed
		}
		fmt.Fprintf(&b, " %s%-10s%s %-22s %s%-9s%s %8d %s$%9.4f%s %s%5dms%s\n",
			stCol, trunc(q.Provider, 10), aReset,
			trunc(q.ModelUsed, 22),
			cplxColor(q.Complexity), trunc(q.Complexity, 9), aReset,
			q.InputTokens+q.OutputTokens,
			aGreen, q.CostUSD, aReset,
			aGray, q.LatencyMS, aReset)
	}
	fmt.Fprintln(&b, line)
	fmt.Fprintf(&b, " %sCtrl+C to quit · refreshes every 1.5s%s\n", aDim, aReset)
	return b.String()
}

func tierColor(t string) string {
	switch t {
	case "free", "local":
		return aGreen
	case "standard":
		return aCyan
	case "premium":
		return aPurple
	}
	return aGray
}

func cplxColor(c string) string {
	switch c {
	case "simple", "cached":
		return aGreen
	case "standard":
		return aCyan
	case "complex":
		return aPurple
	case "critical":
		return aRed
	}
	return aGray
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "~"
}

func humanInt(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
