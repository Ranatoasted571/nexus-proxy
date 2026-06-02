package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

// mcp command — expose NEXUS usage/savings as an MCP server (stdio transport),
// so Claude Code (or any MCP client) can ask "how much have I saved today?".
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run NEXUS as an MCP server (stdio) — query usage/savings from Claude Code",
	RunE:  runMCP,
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func runMCP(cmd *cobra.Command, args []string) error {
	db, err := storage.New(storage.DefaultDBPath())
	if err != nil {
		return err
	}
	defer db.Close()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var req mcpRequest
		if json.Unmarshal([]byte(line), &req) != nil {
			continue
		}
		resp := mcpHandle(db, &req)
		if resp == nil { // notification — no reply
			continue
		}
		b, _ := json.Marshal(resp)
		_, _ = out.Write(b)
		_ = out.WriteByte('\n')
		_ = out.Flush()
	}
	return in.Err()
}

func mcpHandle(db *storage.DB, req *mcpRequest) *mcpResponse {
	if len(req.ID) == 0 { // JSON-RPC notification
		return nil
	}
	resp := &mcpResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "nexus", "version": Version},
		}
	case "tools/list":
		resp.Result = map[string]interface{}{"tools": mcpTools()}
	case "tools/call":
		resp.Result = mcpCall(db, req.Params)
	case "ping":
		resp.Result = map[string]interface{}{}
	default:
		resp.Error = &mcpError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func periodSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"period": map[string]interface{}{
				"type": "string", "enum": []string{"today", "week", "month"},
				"description": "Time window (default: today)",
			},
		},
	}
}

func mcpTools() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "nexus_stats", "description": "NEXUS usage stats: requests, cost, tokens, cache savings, avg latency for a period.", "inputSchema": periodSchema()},
		{"name": "nexus_savings", "description": "How much NEXUS saved vs running everything on Claude — saved $ and % cheaper.", "inputSchema": periodSchema()},
		{"name": "nexus_recent", "description": "The most recent proxied requests (provider, model, complexity, cost, latency).", "inputSchema": map[string]interface{}{
			"type": "object", "properties": map[string]interface{}{"limit": map[string]interface{}{"type": "integer", "description": "How many (default 10)"}},
		}},
		{"name": "nexus_providers", "description": "The configured providers and their tiers.", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "nexus_cost_breakdown", "description": "Cost and request count grouped by provider.", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
	}
}

func mcpCall(db *storage.DB, params json.RawMessage) map[string]interface{} {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	_ = json.Unmarshal(params, &p)
	period, _ := p.Arguments["period"].(string)
	if period == "" {
		period = "today"
	}

	var text string
	switch p.Name {
	case "nexus_stats":
		s, err := db.GetStats(period)
		text = jsonTextOrErr(s, err)
	case "nexus_savings":
		s, err := db.GetSavings(period)
		text = jsonTextOrErr(s, err)
	case "nexus_recent":
		limit := 10
		if f, ok := p.Arguments["limit"].(float64); ok && f > 0 {
			limit = int(f)
		}
		rs, err := db.GetRecentRequests(limit)
		if err != nil {
			text = "error: " + err.Error()
			break
		}
		type item struct {
			Provider, ModelAsked, ModelUsed, Complexity string
			CostUSD                                     float64
			LatencyMS                                   int64
			Status                                      int
		}
		out := make([]item, 0, len(rs))
		for _, r := range rs {
			out = append(out, item{r.Provider, r.ModelAsked, r.ModelUsed, r.Complexity, r.CostUSD, r.LatencyMS, r.Status})
		}
		text = jsonText(out)
	case "nexus_providers":
		cfg, _ := config.Load(config.DefaultPath())
		out := []map[string]string{}
		for _, pc := range cfg.Providers {
			out = append(out, map[string]string{"name": pc.Name, "tier": pc.Tier})
		}
		text = jsonText(out)
	case "nexus_cost_breakdown":
		b, err := db.GetProviderBreakdown()
		text = jsonTextOrErr(b, err)
	default:
		return map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": "unknown tool: " + p.Name}},
			"isError": true,
		}
	}
	return map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": text}}}
}

func jsonText(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func jsonTextOrErr(v interface{}, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return jsonText(v)
}
