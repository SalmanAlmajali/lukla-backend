package app

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
// The application must fail to start if required fields are missing or invalid.
type Config struct {
	AppEnv  string
	AppName string

	HTTP HTTPConfig
	DB   DBConfig
	Auth AuthConfig
	Log  LogConfig
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DBConfig holds PostgreSQL connection pool settings.
// Uses int to match the database/sql API (SetMaxOpenConns, SetMaxIdleConns).
type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// AuthConfig holds session/token settings.
type AuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	TokenSecret     string
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string
	Format string
}

// LoadConfig reads configuration from environment variables and validates required fields.
func LoadConfig() (Config, error) {
	// Attempt to load .env from current directory, parent (for cmd/api), or workspace root
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	cfg := Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppName: getEnv("APP_NAME", "lukla-operations-engine"),
		HTTP: HTTPConfig{
			Addr:            getEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		DB: DBConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Auth: AuthConfig{
			AccessTokenTTL:  getDuration("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDuration("AUTH_REFRESH_TOKEN_TTL", 168*time.Hour),
			TokenSecret:     getEnv("AUTH_TOKEN_SECRET", ""),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.DB.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Auth.TokenSecret == "" {
		return fmt.Errorf("AUTH_TOKEN_SECRET is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
