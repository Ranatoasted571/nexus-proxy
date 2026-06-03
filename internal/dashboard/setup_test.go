// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// isolateNexusHome redirects ~/.nexus to a TempDir so tests never write to the
// user's real config. We set both HOME (Unix) and USERPROFILE (Windows) so the
// resolver picks the temp dir on every platform.
func isolateNexusHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return filepath.Join(dir, ".nexus")
}

func TestSetupStatusFirstRun(t *testing.T) {
	isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)

	rec := httptest.NewRecorder()
	s.handleSetupStatus(rec, httptest.NewRequest("GET", "/api/setup/status", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["first_run"] != true {
		t.Errorf("first_run = %v, want true on a fresh home", got["first_run"])
	}
	if got["platform"] == nil || got["platform"] == "" {
		t.Error("platform should be reported (linux/darwin/windows)")
	}
	if rec, ok := got["recommended"].([]interface{}); !ok || len(rec) == 0 {
		t.Errorf("recommended providers missing: %#v", got["recommended"])
	}
}

func TestSetupSaveSkipMarksDone(t *testing.T) {
	nx := isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)

	body := bytes.NewBufferString(`{"skip": true}`)
	rec := httptest.NewRecorder()
	s.handleSetupSave(rec, httptest.NewRequest("POST", "/api/setup/save", body))
	if rec.Code != 200 {
		t.Fatalf("save = %d, want 200", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(nx, setupMarkerName)); err != nil {
		t.Errorf("expected setup marker at %s: %v", nx, err)
	}

	// Status should no longer be first_run after the skip.
	rec2 := httptest.NewRecorder()
	s.handleSetupStatus(rec2, httptest.NewRequest("GET", "/api/setup/status", nil))
	var got map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &got)
	if got["first_run"] != false {
		t.Errorf("first_run after skip = %v, want false", got["first_run"])
	}
}

func TestSetupSavePersistsProviders(t *testing.T) {
	nx := isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)

	body := bytes.NewBufferString(`{"providers":[
		{"name":"groq","api_key":"gsk_test_xxx"},
		{"name":"deepseek","api_key":"sk_test_yyy"},
		{"name":"bogus-provider","api_key":"k"}
	]}`)
	rec := httptest.NewRecorder()
	s.handleSetupSave(rec, httptest.NewRequest("POST", "/api/setup/save", body))
	if rec.Code != 200 {
		t.Fatalf("save = %d, want 200", rec.Code)
	}

	var got map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	// Bogus provider should be rejected silently — only valid ones counted.
	if got["saved"] != float64(2) {
		t.Errorf("saved = %v, want 2", got["saved"])
	}

	// Marker exists.
	if _, err := os.Stat(filepath.Join(nx, setupMarkerName)); err != nil {
		t.Errorf("expected setup marker: %v", err)
	}

	// Status now sees both providers configured.
	rec2 := httptest.NewRecorder()
	s.handleSetupStatus(rec2, httptest.NewRequest("GET", "/api/setup/status", nil))
	var status map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &status)
	cfgd, _ := status["configured"].([]interface{})
	if len(cfgd) != 2 {
		t.Errorf("configured = %v, want 2 entries", cfgd)
	}
	if status["first_run"] != false {
		t.Errorf("first_run after save = %v, want false", status["first_run"])
	}
}

func TestSetupTestRejectsBadJSON(t *testing.T) {
	isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)
	rec := httptest.NewRecorder()
	s.handleSetupTest(rec, httptest.NewRequest("POST", "/api/setup/test", bytes.NewBufferString("not json")))
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200 (errors come in body)", rec.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["ok"] != false {
		t.Errorf("ok = %v, want false", got["ok"])
	}
}

func TestSetupRoutesRegistered(t *testing.T) {
	isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)
	r := s.Routes()
	for _, path := range []string{"/api/setup/status", "/api/setup/test", "/api/setup/save"} {
		method := "GET"
		var body bytes.Buffer
		if path != "/api/setup/status" {
			method = "POST"
			body.WriteString("{}")
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, &body)
		r.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s not routed", path)
		}
	}
}
