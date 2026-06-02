package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultDBPathShape verifies the default path is built with OS-native
// separators and lands at <home>/.nexus/nexus.db on every platform.
func TestDefaultDBPathShape(t *testing.T) {
	p := DefaultDBPath()
	if got := filepath.Base(p); got != "nexus.db" {
		t.Errorf("base = %q, want nexus.db", got)
	}
	if got := filepath.Base(filepath.Dir(p)); got != ".nexus" {
		t.Errorf("parent dir = %q, want .nexus", got)
	}
}

// TestOpenDBPathWithSpaces exercises DB open + I/O at a path containing a space.
// The default install path can contain spaces (e.g. "…\nexus proxy\…"), which is
// a classic Windows DSN-handling footgun, so guard it explicitly.
func TestOpenDBPathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := New(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("open db at spaced path: %v", err)
	}
	defer db.Close() // release the handle so TempDir cleanup can delete it (Windows)

	if _, err := db.LogRequest(&Request{
		CreatedAt: time.Now().UTC(), Provider: "groq",
		ModelAsked: "m", ModelUsed: "m", Complexity: "simple", Status: 200,
	}); err != nil {
		t.Fatalf("log request: %v", err)
	}
	st, err := db.GetStats("today")
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if st.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", st.TotalRequests)
	}
}
