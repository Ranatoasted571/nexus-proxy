package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBudgetAlertsFireOncePerThreshold(t *testing.T) {
	got := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- string(b)
	}))
	defer srv.Close()

	b := newBudgetTracker(10.0, 0, srv.URL, 0.8)
	b.Add(7.0) // 70% — below threshold, no alert
	b.Add(1.5) // 85% — warn
	b.Add(2.0) // 105% — over
	b.Add(5.0) // still over — must NOT fire again

	recv := func() string {
		select {
		case m := <-got:
			return m
		case <-time.After(2 * time.Second):
			t.Fatal("expected an alert webhook call")
			return ""
		}
	}
	// The two alerts fire in separate goroutines, so order isn't guaranteed.
	both := recv() + "\n" + recv()
	if !strings.Contains(both, "budget used") {
		t.Errorf("expected a warning alert, got: %s", both)
	}
	if !strings.Contains(both, "exceeded") {
		t.Errorf("expected an exceeded alert, got: %s", both)
	}
	select {
	case extra := <-got:
		t.Errorf("alerts must fire once per threshold; got an extra: %s", extra)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestBudgetNoAlertWithoutWebhook(t *testing.T) {
	b := newBudgetTracker(1.0, 0, "", 0) // no webhook configured
	b.Add(2.0)                           // over budget, but nothing to post to
	if !b.Over() {
		t.Error("should be over budget")
	}
}
