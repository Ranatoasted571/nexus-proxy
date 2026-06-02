package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/storage"
)

func TestRequestDetailEndpoint(t *testing.T) {
	db, err := storage.New(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id, err := db.LogRequest(&storage.Request{
		CreatedAt: time.Now(), Provider: "deepseek", ModelAsked: "claude-sonnet-4-6",
		ModelUsed: "deepseek-chat", Complexity: "standard", Status: 200,
		Prompt:   `{"messages":[{"role":"user","content":"hi"}]}`,
		Response: `{"content":[{"type":"text","text":"yo"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	s := &Server{db: db}
	rec := httptest.NewRecorder()
	s.Routes().ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/requests/%d", id), nil))
	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var m map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if resp, _ := m["response"].(string); !strings.Contains(resp, "yo") {
		t.Errorf("detail missing response: %v", m)
	}
	if m["inspected"] != true {
		t.Errorf("inspected flag should be true: %v", m["inspected"])
	}
}
