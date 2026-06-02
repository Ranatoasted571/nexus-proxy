package proxy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// cacheEntry is a cached upstream response plus the metadata needed to log a hit.
type cacheEntry struct {
	status  int
	ctype   string
	body    []byte
	model   string
	in, out int
	expires time.Time
}

// responseCache is an in-memory, TTL + FIFO-capped response cache. Identical
// requests are served instantly and for free. It only caches successful (200)
// responses and keys on a normalized hash of the request body, so stream vs
// non-stream and different models never collide.
type responseCache struct {
	mu     sync.Mutex
	ttl    time.Duration
	max    int
	m      map[string]cacheEntry
	order  []string
	Hits   int64
	Misses int64
}

func newResponseCache(ttl time.Duration, max int) *responseCache {
	return &responseCache{ttl: ttl, max: max, m: make(map[string]cacheEntry)}
}

func (c *responseCache) get(key string) (cacheEntry, bool) {
	if c == nil {
		return cacheEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expires) {
		if ok {
			delete(c.m, key)
		}
		c.Misses++
		return cacheEntry{}, false
	}
	c.Hits++
	return e, true
}

func (c *responseCache) set(key string, e cacheEntry) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e.expires = time.Now().Add(c.ttl)
	if _, exists := c.m[key]; !exists {
		c.order = append(c.order, key)
		for len(c.order) > c.max {
			delete(c.m, c.order[0])
			c.order = c.order[1:]
		}
	}
	c.m[key] = e
}

// cacheKey normalizes the request body to canonical JSON before hashing, so
// whitespace / key-order differences still hit the same entry.
func cacheKey(prefix string, body []byte) string {
	norm := body
	var v interface{}
	if json.Unmarshal(body, &v) == nil {
		if b, err := json.Marshal(v); err == nil {
			norm = b
		}
	}
	sum := sha256.Sum256(norm)
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func quickModel(body []byte) string {
	var r struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &r)
	return r.Model
}

// bestEffortUsage tries every known usage format (Anthropic JSON, OpenAI JSON,
// Anthropic SSE) and returns the first non-zero token counts.
func bestEffortUsage(body []byte) (int, int) {
	if in, out := parseAnthropicUsage(body); in+out > 0 {
		return in, out
	}
	if in, out := parseOpenAIUsage(body); in+out > 0 {
		return in, out
	}
	return parseStreamUsage(body)
}

// cachingWriter tees the response into a buffer so a 200 response can be cached.
// It transparently forwards Flush so streaming still works.
type cachingWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
	tooBig bool
}

func newCachingWriter(w http.ResponseWriter) *cachingWriter {
	return &cachingWriter{ResponseWriter: w, status: http.StatusOK}
}

func (c *cachingWriter) WriteHeader(s int) {
	c.status = s
	c.ResponseWriter.WriteHeader(s)
}

func (c *cachingWriter) Write(b []byte) (int, error) {
	if !c.tooBig {
		if c.buf.Len()+len(b) > 2<<20 { // don't cache responses over ~2MB
			c.tooBig = true
			c.buf.Reset()
		} else {
			c.buf.Write(b)
		}
	}
	return c.ResponseWriter.Write(b)
}

func (c *cachingWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *cachingWriter) cacheable() bool {
	return c.status == http.StatusOK && !c.tooBig && c.buf.Len() > 0
}

func (c *cachingWriter) entry() cacheEntry {
	in, out := bestEffortUsage(c.buf.Bytes())
	return cacheEntry{
		status: c.status,
		ctype:  c.Header().Get("Content-Type"),
		body:   append([]byte(nil), c.buf.Bytes()...),
		in:     in,
		out:    out,
	}
}
