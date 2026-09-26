package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type Project struct {
	ID                  int64
	Name                string
	APIKey              string
	DefaultChangeFreq   string
	DefaultPriority     float64
	MaxURLsPerSitemap   int
	ValidateURLsDefault bool
	IsActive            bool
	CreatedAt           int64
	UpdatedAt           int64
}

var ErrNotFound = errors.New("not found")

func (d *DB) GetProjectByAPIKey(ctx context.Context, apiKey string) (*Project, error) {
	row := d.QueryRowContext(ctx,
		`SELECT id, name, api_key, default_change_freq, default_priority,
				max_urls_per_sitemap, validate_urls_default, is_active,
				created_at, updated_at
		 FROM projects WHERE api_key = ? AND is_active = 1`, apiKey)

	var p Project
	var validate, active int
	err := row.Scan(&p.ID, &p.Name, &p.APIKey, &p.DefaultChangeFreq,
		&p.DefaultPriority, &p.MaxURLsPerSitemap, &validate, &active,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	p.ValidateURLsDefault = validate == 1
	p.IsActive = active == 1
	return &p, nil
}

// Cache -----------------------------------------------------------------

func (d *DB) GetCache(ctx context.Context, contentHash, operation string) (string, int, error) {
	nowMs := time.Now().UnixMilli()
	row := d.QueryRowContext(ctx,
		`SELECT output_xml, compressed FROM sitemap_cache
		 WHERE content_hash = ? AND operation = ? AND expires_at > ?`,
		contentHash, operation, nowMs)

	var xml string
	var compressed int
	if err := row.Scan(&xml, &compressed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, ErrNotFound
		}
		return "", 0, err
	}
	return xml, compressed, nil
}

func (d *DB) PutCache(ctx context.Context, contentHash, operation, xml string, compressed int, urlCount, ttlSeconds int) error {
	nowMs := time.Now().UnixMilli()
	expiresAt := nowMs + int64(ttlSeconds)*1000
	_, err := d.ExecContext(ctx,
		`INSERT INTO sitemap_cache (content_hash, operation, output_xml, compressed, url_count, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(content_hash) DO UPDATE SET
			 output_xml = excluded.output_xml,
			 compressed = excluded.compressed,
			 url_count = excluded.url_count,
			 expires_at = excluded.expires_at,
			 created_at = excluded.created_at`,
		contentHash, operation, xml, compressed, urlCount, expiresAt, nowMs)
	return err
}

func (d *DB) PurgeExpiredCache(ctx context.Context) (int64, error) {
	res, err := d.ExecContext(ctx, `DELETE FROM sitemap_cache WHERE expires_at < ?`, time.Now().UnixMilli())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// History ---------------------------------------------------------------

type HistoryEntry struct {
	ID              int64  `json:"id"`
	Operation       string `json:"operation"`
	URLCount        int    `json:"url_count"`
	ValidCount      int    `json:"valid_count"`
	InvalidCount    int    `json:"invalid_count"`
	OutputSizeBytes int    `json:"output_size_bytes"`
	Compressed      bool   `json:"compressed"`
	CacheHit        string `json:"cache_hit,omitempty"`
	DurationMs      int    `json:"duration_ms"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message,omitempty"`
	ClientIP        string `json:"client_ip,omitempty"`
	CreatedAt       int64  `json:"created_at"`
}

func (d *DB) LogHistory(ctx context.Context, projectID int64, operation string, urlCount, validCount, invalidCount, outputSize, compressed, durationMs int, cacheHit, status, errMsg, clientIP, userAgent string) error {
	nowMs := time.Now().UnixMilli()

	_, err := d.ExecContext(ctx,
		`INSERT INTO generation_history
		 (project_id, operation, url_count, valid_count, invalid_count,
		  output_size_bytes, compressed, cache_hit, duration_ms, status,
		  error_message, client_ip, user_agent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, operation, urlCount, validCount, invalidCount,
		outputSize, compressed, cacheHit, durationMs, status,
		errMsg, clientIP, userAgent, nowMs)
	if err != nil {
		return err
	}

	date := time.Now().UTC().Format("2006-01-02")
	isHit := 0
	if cacheHit == "memory" || cacheHit == "sqlite" {
		isHit = 1
	}
	isFail := 0
	if status != "success" {
		isFail = 1
	}

	_, err = d.ExecContext(ctx,
		`INSERT INTO daily_summary
		 (project_id, date, sitemaps_generated, total_urls, urls_validated,
		  cache_hits, failed_count, updated_at)
		 VALUES (?, ?, 1, ?, ?, ?, ?, ?)
		 ON CONFLICT(project_id, date) DO UPDATE SET
			 sitemaps_generated = sitemaps_generated + 1,
			 total_urls = total_urls + excluded.total_urls,
			 urls_validated = urls_validated + excluded.urls_validated,
			 cache_hits = cache_hits + excluded.cache_hits,
			 failed_count = failed_count + excluded.failed_count,
			 updated_at = excluded.updated_at`,
		projectID, date, urlCount, validCount+invalidCount, isHit, isFail, nowMs)
	return err
}

func (d *DB) ListHistory(ctx context.Context, projectID int64, operation, status, search string, limit, offset int) ([]HistoryEntry, int, error) {
	where := "project_id = ?"
	args := []any{projectID}

	if operation != "" {
		where += " AND operation = ?"
		args = append(args, operation)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if search != "" {
		where += " AND (error_message LIKE ? OR operation LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	var total int
	err := d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM generation_history WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, operation, url_count, valid_count, invalid_count,
					 output_size_bytes, compressed, COALESCE(cache_hit, ''),
					 duration_ms, status, COALESCE(error_message, ''),
					 COALESCE(client_ip, ''), created_at
			  FROM generation_history WHERE ` + where + `
			  ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := []HistoryEntry{}
	for rows.Next() {
		var e HistoryEntry
		var compressed int
		if err := rows.Scan(&e.ID, &e.Operation, &e.URLCount, &e.ValidCount,
			&e.InvalidCount, &e.OutputSizeBytes, &compressed, &e.CacheHit,
			&e.DurationMs, &e.Status, &e.ErrorMessage, &e.ClientIP, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Compressed = compressed == 1
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// Stats -----------------------------------------------------------------

type DayStat struct {
	Date              string `json:"date"`
	SitemapsGenerated int    `json:"sitemaps_generated"`
	TotalURLs         int    `json:"total_urls"`
	URLsValidated     int    `json:"urls_validated"`
	CacheHits         int    `json:"cache_hits"`
	FailedCount       int    `json:"failed_count"`
}

type Stats struct {
	Days        int            `json:"days"`
	Daily       []DayStat      `json:"daily"`
	Totals      map[string]any `json:"totals"`
	ByOperation map[string]int `json:"by_operation"`
}

func (d *DB) GetStats(ctx context.Context, projectID int64, days int) (*Stats, error) {
	fromDate := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := d.QueryContext(ctx,
		`SELECT date, sitemaps_generated, total_urls, urls_validated, cache_hits, failed_count
		 FROM daily_summary WHERE project_id = ? AND date >= ?
		 ORDER BY date ASC`, projectID, fromDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	daily := []DayStat{}
	var totalSitemaps, totalURLs, totalValidated, totalCacheHits, totalFailed int
	for rows.Next() {
		var ds DayStat
		if err := rows.Scan(&ds.Date, &ds.SitemapsGenerated, &ds.TotalURLs,
			&ds.URLsValidated, &ds.CacheHits, &ds.FailedCount); err != nil {
			return nil, err
		}
		daily = append(daily, ds)
		totalSitemaps += ds.SitemapsGenerated
		totalURLs += ds.TotalURLs
		totalValidated += ds.URLsValidated
		totalCacheHits += ds.CacheHits
		totalFailed += ds.FailedCount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	opRows, err := d.QueryContext(ctx,
		`SELECT operation, COUNT(*) FROM generation_history
		 WHERE project_id = ? AND status = 'success'
		 GROUP BY operation ORDER BY COUNT(*) DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer opRows.Close()

	byOperation := make(map[string]int)
	for opRows.Next() {
		var op string
		var count int
		if err := opRows.Scan(&op, &count); err != nil {
			return nil, err
		}
		byOperation[op] = count
	}

	hitRate := 0.0
	if totalSitemaps > 0 {
		hitRate = float64(totalCacheHits) / float64(totalSitemaps) * 100
	}

	return &Stats{
		Days:  days,
		Daily: daily,
		Totals: map[string]any{
			"sitemaps_generated": totalSitemaps,
			"total_urls":         totalURLs,
			"urls_validated":     totalValidated,
			"cache_hits":         totalCacheHits,
			"failed_count":       totalFailed,
			"cache_hit_rate":     float64(int(hitRate*100)) / 100,
		},
		ByOperation: byOperation,
	}, nil
}

// Hash helpers ----------------------------------------------------------

func HashContent(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashJSON(v any) string {
	data, _ := json.Marshal(v)
	return HashContent(data)
}
