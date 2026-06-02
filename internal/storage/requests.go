package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Request represents a logged proxy request
type Request struct {
	ID           int64
	CreatedAt    time.Time
	RequestID    string
	ModelAsked   string
	ModelUsed    string
	Provider     string
	Complexity   string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	LatencyMS    int64
	Status       int
	Error        string
	Stream       bool
}

// LogRequest saves a request to the database and returns the new row ID.
func (db *DB) LogRequest(req *Request) (int64, error) {
	res, err := db.conn.Exec(`
		INSERT INTO requests (
			created_at, request_id, model_asked, model_used,
			provider, complexity, input_tokens, output_tokens,
			cost_usd, latency_ms, status, error, stream
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		// Store as SQLite-native UTC datetime text so date()/strftime() work.
		req.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		req.RequestID,
		req.ModelAsked,
		req.ModelUsed,
		req.Provider,
		req.Complexity,
		req.InputTokens,
		req.OutputTokens,
		req.CostUSD,
		req.LatencyMS,
		req.Status,
		req.Error,
		req.Stream,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// GetRecentRequests returns the N most recent requests
func (db *DB) GetRecentRequests(limit int) ([]*Request, error) {
	rows, err := db.conn.Query(`
		SELECT id, created_at, request_id, model_asked, model_used,
			   provider, complexity, input_tokens, output_tokens,
			   cost_usd, latency_ms, status, error, stream
		FROM requests
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRequests(rows)
}

// GetStats returns aggregated statistics
func (db *DB) GetStats(period string) (*Stats, error) {
	var since string
	switch period {
	case "today":
		since = "date('now')"
	case "week":
		since = "date('now', '-7 days')"
	case "month":
		since = "date('now', '-30 days')"
	default:
		since = "date('now')"
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM requests
		WHERE date(created_at) >= %s`, since)

	var stats Stats
	err := db.conn.QueryRow(query).Scan(
		&stats.TotalRequests,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCostUSD,
		&stats.AvgLatencyMS,
	)
	if err != nil {
		return nil, err
	}
	stats.Period = period
	return &stats, nil
}

// GetProviderBreakdown returns stats grouped by provider
func (db *DB) GetProviderBreakdown() ([]*ProviderStats, error) {
	rows, err := db.conn.Query(`
		SELECT
			provider,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM requests
		GROUP BY provider
		ORDER BY requests DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ProviderStats
	for rows.Next() {
		var ps ProviderStats
		if err := rows.Scan(&ps.Provider, &ps.Requests, &ps.TotalTokens, &ps.TotalCostUSD, &ps.AvgLatencyMS); err != nil {
			continue
		}
		results = append(results, &ps)
	}
	return results, nil
}

// GetCostForecast estimates monthly cost based on current usage rate
func (db *DB) GetCostForecast() (float64, error) {
	var costToday float64
	err := db.conn.QueryRow(`
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM requests
		WHERE date(created_at) = date('now')`).Scan(&costToday)
	if err != nil {
		return 0, err
	}
	// Extrapolate to 30 days
	return costToday * 30, nil
}

// TimeBucket is one point in a time series.
type TimeBucket struct {
	Bucket   string  `json:"bucket"`
	Requests int     `json:"requests"`
	CostUSD  float64 `json:"cost_usd"`
}

// GetHourlySeries returns per-hour request counts and cost over the last 24h.
func (db *DB) GetHourlySeries() ([]TimeBucket, error) {
	rows, err := db.conn.Query(`
		SELECT strftime('%m-%d %H:00', created_at) AS bucket,
		       COUNT(*),
		       COALESCE(SUM(cost_usd), 0)
		FROM requests
		WHERE created_at >= datetime('now', '-24 hours')
		GROUP BY bucket
		ORDER BY bucket`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TimeBucket
	for rows.Next() {
		var b TimeBucket
		if err := rows.Scan(&b.Bucket, &b.Requests, &b.CostUSD); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetComplexityBreakdown returns request counts grouped by complexity.
func (db *DB) GetComplexityBreakdown() (map[string]int, error) {
	rows, err := db.conn.Query(`SELECT complexity, COUNT(*) FROM requests GROUP BY complexity`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := map[string]int{}
	for rows.Next() {
		var c string
		var n int
		if err := rows.Scan(&c, &n); err == nil {
			m[c] = n
		}
	}
	return m, rows.Err()
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func scanRequests(rows *sql.Rows) ([]*Request, error) {
	var results []*Request
	for rows.Next() {
		r := &Request{}
		err := rows.Scan(
			&r.ID, &r.CreatedAt, &r.RequestID, &r.ModelAsked, &r.ModelUsed,
			&r.Provider, &r.Complexity, &r.InputTokens, &r.OutputTokens,
			&r.CostUSD, &r.LatencyMS, &r.Status, &r.Error, &r.Stream,
		)
		if err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ─── Types ─────────────────────────────────────────────────────────────────

type Stats struct {
	Period            string  `json:"period"`
	TotalRequests     int     `json:"total_requests"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	AvgLatencyMS      float64 `json:"avg_latency_ms"`
}

type ProviderStats struct {
	Provider     string  `json:"provider"`
	Requests     int     `json:"requests"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
}
