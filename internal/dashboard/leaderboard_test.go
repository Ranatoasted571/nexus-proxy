package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

func TestLeaderboardEndpoint(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "lb.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, _ = db.LogRequest(&storage.Request{
		CreatedAt: time.Now(), User: "alice", ModelAsked: "claude-sonnet-4-6", Provider: "groq",
		InputTokens: 1_000_000, OutputTokens: 0, CostUSD: 0, Status: 200,
	})

	s := &Server{db: db}
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, httptest.NewRequest("GET", "/api/leaderboard?period=today", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "alice") {
		t.Errorf("leaderboard should include alice: %s", rec.Body.String())
	}
	var m map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if _, ok := m["leaderboard"]; !ok {
		t.Errorf("response missing leaderboard key: %v", m)
	}
}
