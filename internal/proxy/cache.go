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
	status   int
	ctype    string
	body     []byte
	model    string
	in, out  int
	vec      sparseVec // semantic fingerprint (only set when semantic cache is on)
	hasTools bool      // requests with tools are never served as a semantic match
	expires  time.Time
}

// responseCache is an in-memory, TTL + FIFO-capped response cache. Identical
// requests are served instantly and for free. It only caches successful (200)
// responses and keys on a normalized hash of the request body, so stream vs
// non-stream and different models never collide.
//
// With semantic mode enabled, a miss on the exact key falls back to a cosine
// similarity search over recent tool-less entries of the same model, so
// near-identical prompts are also served from cache.
type responseCache struct {
	mu        sync.Mutex
	ttl       time.Duration
	max       int
	m         map[string]cacheEntry
	order     []string
	semantic  bool
	threshold float64
	Hits      int64
	Misses    int64
}

func newResponseCache(ttl time.Duration, max int, semantic bool, threshold float64) *responseCache {
	if threshold <= 0 {
		threshold = 0.95
	}
	return &responseCache{ttl: ttl, max: max, m: make(map[string]cacheEntry), semantic: semantic, threshold: threshold}
}

// getSemantic returns the best near-match for vec among non-tool entries of the
// same model whose cosine similarity meets the threshold.
func (c *responseCache) getSemantic(model string, vec sparseVec) (cacheEntry, bool) {
	if c == nil || !c.semantic || len(vec) == 0 {
		return cacheEntry{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	best := c.threshold
	var hit cacheEntry
	found := false
	for _, e := range c.m {
		if e.hasTools || e.model != model || len(e.vec) == 0 || now.After(e.expires) {
			continue
		}
		if s := cosine(vec, e.vec); s >= best {
			best, hit, found = s, e, true
		}
	}
	if found {
		c.Hits++
	}
	return hit, found
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
// whitespace / key-order differences — and volatile, response-irrelevant fields
// like metadata and cache_control markers — still hit the same entry.
func cacheKey(prefix string, body []byte) string {
	norm := body
	var v interface{}
	if json.Unmarshal(body, &v) == nil {
		stripVolatile(v)
		if b, err := json.Marshal(v); err == nil {
			norm = b
		}
	}
	sum := sha256.Sum256(norm)
	return prefix + ":" + hex.EncodeToString(sum[:])
}

// stripVolatile removes fields that never change the model's output, so requests
// that differ only in billing/telemetry metadata share a cache entry:
//   - top-level "metadata" / "request_id"
//   - "cache_control" markers anywhere (they affect provider billing, not output)
func stripVolatile(v interface{}) {
	if m, ok := v.(map[string]interface{}); ok {
		delete(m, "metadata")
		delete(m, "request_id")
	}
	stripCacheControl(v)
}

func stripCacheControl(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		delete(t, "cache_control")
		for _, val := range t {
			stripCacheControl(val)
		}
	case []interface{}:
		for _, val := range t {
			stripCacheControl(val)
		}
	}
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
