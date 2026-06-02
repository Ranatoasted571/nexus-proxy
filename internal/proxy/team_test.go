package proxy

import (
	"net/http"
	"testing"
)

func TestDeriveUser(t *testing.T) {
	cases := []struct {
		set  map[string]string
		want string
	}{
		{map[string]string{"X-Nexus-User": "carol"}, "carol"},
		{map[string]string{"x-api-key": "nexus-alice"}, "alice"},
		{map[string]string{"x-api-key": "nexus-local"}, ""}, // placeholder, not a user
		{map[string]string{"x-api-key": "sk-ant-realkey"}, ""},
		{map[string]string{}, ""},
		// header wins over key
		{map[string]string{"X-Nexus-User": "dave", "x-api-key": "nexus-alice"}, "dave"},
	}
	for _, c := range cases {
		h := http.Header{}
		for k, v := range c.set {
			h.Set(k, v)
		}
		if got := deriveUser(h); got != c.want {
			t.Errorf("deriveUser(%v) = %q, want %q", c.set, got, c.want)
		}
	}
}
