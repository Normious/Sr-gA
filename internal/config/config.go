package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         int
	Env          string
	LogLevel     string
	DatabasePath string

	MemoryCacheTTLSeconds int
	SQLiteCacheTTLSeconds int
	MemoryCacheMaxItems   int

	MaxURLsPerRequest             int
	MaxURLsForValidation          int
	MaxBatchValidationConcurrency int
	DefaultValidationTimeoutSecs  int

	ValidationUserAgent     string
	ValidationMaxRedirects int

	TempDir string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:         getInt("PORT", 4017),
		Env:          getString("ENV", "production"),
		LogLevel:     getString("LOG_LEVEL", "info"),
		DatabasePath: getString("DATABASE_PATH", "./data/srga.db"),

		MemoryCacheTTLSeconds: getInt("MEMORY_CACHE_TTL_SECONDS", 3600),
		SQLiteCacheTTLSeconds: getInt("SQLITE_CACHE_TTL_SECONDS", 86400),
		MemoryCacheMaxItems:   getInt("MEMORY_CACHE_MAX_ITEMS", 500),

		MaxURLsPerRequest:             getInt("MAX_URLS_PER_REQUEST", 50000),
		MaxURLsForValidation:          getInt("MAX_URLS_FOR_VALIDATION", 500),
		MaxBatchValidationConcurrency: getInt("MAX_BATCH_VALIDATION_CONCURRENCY", 50),
		DefaultValidationTimeoutSecs:  getInt("DEFAULT_VALIDATION_TIMEOUT_SECONDS", 10),

		ValidationUserAgent:     getString("VALIDATION_USER_AGENT", "Sr-gA-Validator/1.0 (+https://github.com/Normious/Sr-gA)"),
		ValidationMaxRedirects: getInt("VALIDATION_MAX_REDIRECTS", 5),

		TempDir: getString("TEMP_DIR", "./data/tmp"),
	}
}

func (c *Config) SetupLogger() {
	var level slog.Level
	switch c.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func getString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
