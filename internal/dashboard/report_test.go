package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

func TestReportEndpoint(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "rep.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, _ = db.LogRequest(&storage.Request{
		CreatedAt: time.Now(), ModelAsked: "claude-sonnet-4-6", Provider: "groq",
		InputTokens: 1_000_000, OutputTokens: 0, CostUSD: 0, Redacted: 7, Status: 200,
	})

	s := &Server{db: db}
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, httptest.NewRequest("GET", "/api/report?period=today", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["redacted_total"].(float64) != 7 {
		t.Errorf("redacted_total = %v, want 7", m["redacted_total"])
	}
	if m["saved_usd"].(float64) < 2.9 {
		t.Errorf("saved_usd = %v, want ~3", m["saved_usd"])
	}
	if _, ok := m["leaked"]; !ok {
		t.Error("report should include a leaked count")
	}
}
