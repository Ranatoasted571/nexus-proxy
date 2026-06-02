package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

func TestMCPInitialize(t *testing.T) {
	resp := mcpHandle(nil, &mcpRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"})
	if resp == nil || resp.Result == nil {
		t.Fatal("initialize returned no result")
	}
	m := resp.Result.(map[string]interface{})
	if m["protocolVersion"] == nil || m["serverInfo"] == nil || m["capabilities"] == nil {
		t.Errorf("incomplete initialize result: %v", m)
	}
}

func TestMCPToolsList(t *testing.T) {
	resp := mcpHandle(nil, &mcpRequest{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/list"})
	tools := resp.Result.(map[string]interface{})["tools"].([]map[string]interface{})
	if len(tools) < 5 {
		t.Errorf("expected at least 5 tools, got %d", len(tools))
	}
	for _, tl := range tools {
		if tl["name"] == "" || tl["inputSchema"] == nil {
			t.Errorf("tool missing name/schema: %v", tl)
		}
	}
}

func TestMCPNotificationNoReply(t *testing.T) {
	if mcpHandle(nil, &mcpRequest{JSONRPC: "2.0", Method: "notifications/initialized"}) != nil {
		t.Error("notifications (no id) must not get a reply")
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	resp := mcpHandle(nil, &mcpRequest{JSONRPC: "2.0", ID: json.RawMessage(`9`), Method: "bogus/method"})
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Errorf("expected -32601 method-not-found, got %+v", resp.Error)
	}
}

func TestMCPToolsCallStats(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, _ = db.LogRequest(&storage.Request{
		CreatedAt: time.Now(), Provider: "groq", ModelAsked: "claude-haiku-4-5", ModelUsed: "llama",
		Complexity: "simple", InputTokens: 100, OutputTokens: 50, CostUSD: 0, Status: 200,
	})

	resp := mcpHandle(db, &mcpRequest{
		JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: "tools/call",
		Params: json.RawMessage(`{"name":"nexus_stats","arguments":{"period":"today"}}`),
	})
	content := resp.Result.(map[string]interface{})["content"].([]map[string]interface{})
	text := content[0]["text"].(string)
	if !strings.Contains(text, "total_requests") {
		t.Errorf("nexus_stats output should include total_requests, got: %s", text)
	}
}

func TestMCPToolsCallUnknownTool(t *testing.T) {
	resp := mcpHandle(nil, &mcpRequest{
		JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: "tools/call",
		Params: json.RawMessage(`{"name":"nope","arguments":{}}`),
	})
	res := resp.Result.(map[string]interface{})
	if res["isError"] != true {
		t.Errorf("unknown tool should set isError, got %v", res)
	}
}
