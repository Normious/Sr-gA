-- ─────────────────────────────────────────────
-- 1. Projects (Tenants)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    api_key TEXT UNIQUE NOT NULL,
    default_change_freq TEXT DEFAULT 'weekly',
    default_priority REAL DEFAULT 0.5,
    max_urls_per_sitemap INTEGER DEFAULT 50000,
    validate_urls_default INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_api_key ON projects(api_key);

-- ─────────────────────────────────────────────
-- 2. Generation History
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS generation_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    operation TEXT NOT NULL,
    url_count INTEGER DEFAULT 0,
    valid_count INTEGER DEFAULT 0,
    invalid_count INTEGER DEFAULT 0,
    output_size_bytes INTEGER,
    compressed INTEGER DEFAULT 0,
    cache_hit TEXT,
    duration_ms INTEGER,
    status TEXT NOT NULL,
    error_message TEXT,
    client_ip TEXT,
    user_agent TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_history_project ON generation_history(project_id);
CREATE INDEX IF NOT EXISTS idx_history_operation ON generation_history(project_id, operation);
CREATE INDEX IF NOT EXISTS idx_history_created_at ON generation_history(created_at);

-- ─────────────────────────────────────────────
-- 3. Sitemap Content Cache (by hash)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sitemap_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content_hash TEXT UNIQUE NOT NULL,
    operation TEXT NOT NULL,
    output_xml TEXT NOT NULL,
    compressed INTEGER DEFAULT 0,
    url_count INTEGER,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cache_hash ON sitemap_cache(content_hash);
CREATE INDEX IF NOT EXISTS idx_cache_expires ON sitemap_cache(expires_at);

-- ─────────────────────────────────────────────
-- 4. Validation Results (short-lived)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS url_validations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    url TEXT NOT NULL,
    url_hash TEXT NOT NULL,
    final_url TEXT,
    status_code INTEGER,
    redirect_count INTEGER DEFAULT 0,
    content_type TEXT,
    response_time_ms INTEGER,
    is_valid INTEGER DEFAULT 0,
    error_message TEXT,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_validations_hash ON url_validations(url_hash);
CREATE INDEX IF NOT EXISTS idx_validations_expires ON url_validations(expires_at);

-- ─────────────────────────────────────────────
-- 5. Daily Summary
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS daily_summary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    date TEXT NOT NULL,
    sitemaps_generated INTEGER DEFAULT 0,
    total_urls INTEGER DEFAULT 0,
    urls_validated INTEGER DEFAULT 0,
    cache_hits INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    UNIQUE(project_id, date)
);

CREATE INDEX IF NOT EXISTS idx_summary_project ON daily_summary(project_id);
CREATE INDEX IF NOT EXISTS idx_summary_date ON daily_summary(date);
