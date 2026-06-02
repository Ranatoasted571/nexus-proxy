package providers

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestSignV4GetVanilla verifies the signer against the canonical AWS
// "get-vanilla" SigV4 test vector.
func TestSignV4GetVanilla(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.amazonaws.com/", nil)
	req.Host = "example.amazonaws.com"
	tm := time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)

	signV4(req, nil, "AKIDEXAMPLE", "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "", "us-east-1", "service", tm)

	want := "Signature=5fa00fa31553b73ebf1942676e86291e8372ff2a2260956d9b8aae1d763fbf31"
	if got := req.Header.Get("Authorization"); !strings.Contains(got, want) {
		t.Errorf("SigV4 signature mismatch\n got:  %s\n want substring: %s", got, want)
	}
	if req.Header.Get("X-Amz-Date") != "20150830T123600Z" {
		t.Errorf("X-Amz-Date = %q", req.Header.Get("X-Amz-Date"))
	}
}
