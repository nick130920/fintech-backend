package configs

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	validJWTSecret = "0123456789abcdef0123456789abcdef"
	fixtureURL     = "postgres://fixture_user:fixture_password@db.example.test:5432/fixture_db?sslmode=require"
)

func TestConfigValidateReleaseRejectsUnsafeJWTSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{name: "empty", secret: ""},
		{name: "whitespace", secret: " \t "},
		{name: "built-in placeholder", secret: " default-secret-key-change-in-production "},
		{name: "too short", secret: "too-short-fixture-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validReleaseConfig()
			cfg.JWT.SecretKey = tt.secret

			assertSafeValidationError(t, cfg.Validate(), "JWT_SECRET_KEY", tt.secret)
		})
	}
}

func TestConfigValidateReleaseAcceptsRailwayDatabaseURL(t *testing.T) {
	cfg := loadReleaseConfig(t, fixtureURL)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}
}

func TestConfigValidateReleaseRejectsUnsafeDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "unsupported scheme", url: "mysql://fixture_user:fixture_password@db.example.test/fixture_db"},
		{name: "malformed", url: "postgres://fixture_user:%zz@db.example.test/fixture_db"},
		{name: "missing host", url: "postgres://fixture_user:fixture_password@/fixture_db"},
		{name: "missing user", url: "postgres://:fixture_password@db.example.test/fixture_db"},
		{name: "whitespace-only user", url: "postgres://%20:fixture_password@db.example.test/fixture_db"},
		{name: "missing password", url: "postgres://fixture_user@db.example.test/fixture_db"},
		{name: "whitespace-only password", url: "postgres://fixture_user:%20@db.example.test/fixture_db"},
		{name: "default password", url: "postgres://fixture_user:postgres@db.example.test/fixture_db"},
		{name: "whitespace-only database name", url: "postgres://fixture_user:fixture_password@db.example.test/%20"},
		{name: "missing database name", url: "postgres://fixture_user:fixture_password@db.example.test/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadReleaseConfig(t, tt.url)

			assertSafeValidationError(t, cfg.Validate(), "DATABASE_URL", tt.url)
		})
	}
}

func TestLoadReleaseDatabaseURLWhitespaceHandling(t *testing.T) {
	t.Run("whitespace-only URL is authoritative and invalid", func(t *testing.T) {
		cfg := loadReleaseConfig(t, "   ")

		err := cfg.Validate()
		assertSafeValidationError(t, err, "DATABASE_URL", "fallback.example.test")
		if strings.Contains(err.Error(), "DB_HOST") {
			t.Errorf("Validate() error = %q; DATABASE_URL must not fall back to DB_HOST", err)
		}
		if cfg.Database.Host != "" || cfg.Database.User != "" || cfg.Database.Password != "" || cfg.Database.DBName != "" {
			t.Errorf("Load() fell back to DB_* values: %+v", cfg.Database)
		}
	})

	t.Run("surrounding whitespace is normalized", func(t *testing.T) {
		cfg := loadReleaseConfig(t, " \t"+fixtureURL+"\n ")

		if cfg.Database.URL != fixtureURL {
			t.Errorf("Database.URL = %q; want normalized %q", cfg.Database.URL, fixtureURL)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v; want nil", err)
		}
	})
}

func TestLoadReleaseMalformedDatabaseURLDoesNotLogOrFallBack(t *testing.T) {
	previousOutput := log.Writer()
	var output bytes.Buffer
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	malformedURL := "postgres://fixture_user:%zz@db.example.test/fixture_db"
	cfg := loadReleaseConfig(t, malformedURL)

	assertSafeValidationError(t, cfg.Validate(), "DATABASE_URL", malformedURL)
	if output.Len() != 0 {
		t.Errorf("Load() wrote %q; want no raw DATABASE_URL or parser log", output.String())
	}
	if cfg.Database.Host != "" || cfg.Database.User != "" || cfg.Database.Password != "" || cfg.Database.DBName != "" {
		t.Errorf("Load() fell back to DB_* values: %+v", cfg.Database)
	}
}

func TestConfigValidateReleaseRejectsUnsafeDiscreteDatabaseConfig(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DatabaseConfig)
		want string
	}{
		{name: "missing host", edit: func(db *DatabaseConfig) { db.Host = " " }, want: "DB_HOST"},
		{name: "missing port", edit: func(db *DatabaseConfig) { db.Port = "" }, want: "DB_PORT"},
		{name: "missing user", edit: func(db *DatabaseConfig) { db.User = "" }, want: "DB_USER"},
		{name: "missing password", edit: func(db *DatabaseConfig) { db.Password = " \t" }, want: "DB_PASSWORD"},
		{name: "default password", edit: func(db *DatabaseConfig) { db.Password = " postgres " }, want: "DB_PASSWORD"},
		{name: "missing database name", edit: func(db *DatabaseConfig) { db.DBName = "" }, want: "DB_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validReleaseConfig()
			tt.edit(&cfg.Database)

			assertSafeValidationError(t, cfg.Validate(), tt.want, "fixture_password")
		})
	}
}

func TestLoadReleaseRequiresDiscreteDatabaseVariables(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "host", key: "DB_HOST"},
		{name: "port", key: "DB_PORT"},
		{name: "user", key: "DB_USER"},
		{name: "password", key: "DB_PASSWORD"},
		{name: "database name", key: "DB_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadReleaseConfigWithoutDatabaseURL(t, tt.key)

			assertSafeValidationError(t, cfg.Validate(), tt.key, "fixture_password")
		})
	}
}

func TestLoadDebugPreservesDevelopmentDatabaseDefaults(t *testing.T) {
	cfg := loadDebugConfig(t)

	if cfg.Database.Host != "localhost" || cfg.Database.Port != "5432" || cfg.Database.User != "postgres" || cfg.Database.Password != "postgres" || cfg.Database.DBName != "fintech_db" {
		t.Errorf("Load() database = %+v; want documented debug defaults", cfg.Database)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}
}

func TestConfigValidateDebugAllowsDevelopmentDefaults(t *testing.T) {
	cfg := Config{
		Server: validHTTPServerConfig("debug"),
		JWT:    JWTConfig{SecretKey: "default-secret-key-change-in-production"},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			DBName:   "fintech_db",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}
}

func TestConfigValidatePreservesGmailValidationOutsideRelease(t *testing.T) {
	cfg := Config{
		Server: validHTTPServerConfig("debug"),
		External: ExternalConfig{Gmail: GmailOAuthConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "https://example.test/callback",
		}},
	}

	assertSafeValidationError(t, cfg.Validate(), "TOKEN_ENCRYPTION_KEY", "client-secret")
}

func loadReleaseConfig(t *testing.T, databaseURL string) *Config {
	t.Helper()
	previous := globalConfig
	globalConfig = nil
	t.Cleanup(func() { globalConfig = previous })

	t.Setenv("GIN_MODE", "release")
	t.Setenv("JWT_SECRET_KEY", validJWTSecret)
	t.Setenv("DATABASE_URL", databaseURL)
	t.Setenv("DB_HOST", "fallback.example.test")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "fallback_user")
	t.Setenv("DB_PASSWORD", "fallback_password")
	t.Setenv("DB_NAME", "fallback_db")

	return Load()
}

func loadReleaseConfigWithoutDatabaseURL(t *testing.T, missingKey string) *Config {
	t.Helper()
	previous := globalConfig
	globalConfig = nil
	t.Cleanup(func() { globalConfig = previous })

	t.Setenv("GIN_MODE", "release")
	t.Setenv("JWT_SECRET_KEY", validJWTSecret)
	unsetEnv(t, "DATABASE_URL")
	t.Setenv("DB_HOST", "db.example.test")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "fixture_user")
	t.Setenv("DB_PASSWORD", "fixture_password")
	t.Setenv("DB_NAME", "fixture_db")
	t.Setenv(missingKey, "")

	return Load()
}

func loadDebugConfig(t *testing.T) *Config {
	t.Helper()
	previous := globalConfig
	globalConfig = nil
	t.Cleanup(func() { globalConfig = previous })

	t.Setenv("GIN_MODE", "debug")
	t.Setenv("JWT_SECRET_KEY", "default-secret-key-change-in-production")
	unsetEnv(t, "DATABASE_URL")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")

	return Load()
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, wasSet := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q): %v", key, err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func validReleaseConfig() Config {
	return Config{
		Server: validHTTPServerConfig("release"),
		JWT:    JWTConfig{SecretKey: validJWTSecret},
		Database: DatabaseConfig{
			Host:     "db.example.test",
			Port:     "5432",
			User:     "fixture_user",
			Password: "fixture_password",
			DBName:   "fixture_db",
		},
	}
}

func validHTTPServerConfig(mode string) ServerConfig {
	return ServerConfig{
		Mode:              mode,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func assertSafeValidationError(t *testing.T, err error, wantVariable, fixtureMarker string) {
	t.Helper()
	if err == nil {
		t.Fatal("Validate() error = nil; want error")
	}
	if !strings.Contains(err.Error(), wantVariable) {
		t.Errorf("Validate() error = %q; want it to name %s", err, wantVariable)
	}
	if fixtureMarker != "" && strings.Contains(err.Error(), fixtureMarker) {
		t.Errorf("Validate() error = %q; must not include fixture secret or URL", err)
	}
}
