package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite",
		path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	wrapper := &DB{DB: db}
	if err := wrapper.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := wrapper.seed(); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	return wrapper, nil
}

func (d *DB) migrate() error {
	paths := []string{
		filepath.Join("migrations", "0001_init.sql"),
		filepath.Join(filepath.Dir(os.Args[0]), "migrations", "0001_init.sql"),
		"/migrations/0001_init.sql",
	}

	var sqlBytes []byte
	var err error
	for _, p := range paths {
		sqlBytes, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	if _, err := d.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

// seed inserts default demo projects so the API is usable immediately.
// ponytail: fixed keys for local dev; real deploys should rotate via SQL.
func (d *DB) seed() error {
	nowMs := time.Now().UnixMilli()
	defaults := []struct {
		name   string
		apiKey string
	}{
		{"shop-a", "shop-a-srga-key-2026"},
		{"demo", "demo-srga-key-2026"},
		{"test", "test-srga-key-2026"},
	}
	for _, p := range defaults {
		_, err := d.Exec(`INSERT INTO projects
			(name, api_key, default_change_freq, default_priority,
			 max_urls_per_sitemap, validate_urls_default, is_active,
			 created_at, updated_at)
			VALUES (?, ?, 'weekly', 0.5, 50000, 0, 1, ?, ?)
			ON CONFLICT(api_key) DO NOTHING`,
			p.name, p.apiKey, nowMs, nowMs)
		if err != nil {
			return err
		}
	}
	return nil
}
