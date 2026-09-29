package configs

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseDatabaseURL_WithSSLMode(t *testing.T) {
	dbURL := "postgres://user1:pass1@localhost:5433/fintech_db?sslmode=disable"
	cfg, err := parseDatabaseURL(dbURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" || cfg.Port != "5433" || cfg.DBName != "fintech_db" {
		t.Fatalf("unexpected host/port/dbname: %+v", cfg)
	}
	if cfg.User != "user1" || cfg.Password != "pass1" {
		t.Fatalf("unexpected credentials: %+v", cfg)
	}
	if cfg.SSLMode != "disable" {
		t.Fatalf("expected sslmode disable, got %s", cfg.SSLMode)
	}
}

func TestParseDatabaseURL_DefaultPortAndSSLRequire(t *testing.T) {
	dbURL := "postgres://user2:pass2@db.internal/fintech_db"
	cfg, err := parseDatabaseURL(dbURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "5432" {
		t.Fatalf("expected default port 5432, got %s", cfg.Port)
	}
	if cfg.SSLMode != "require" {
		t.Fatalf("expected default sslmode require, got %s", cfg.SSLMode)
	}
}

func TestLoadHTTPServerSettings(t *testing.T) {
	const (
		readHeaderTimeout = "HTTP_READ_HEADER_TIMEOUT_SECONDS"
		readTimeout       = "HTTP_READ_TIMEOUT_SECONDS"
		writeTimeout      = "HTTP_WRITE_TIMEOUT_SECONDS"
		idleTimeout       = "HTTP_IDLE_TIMEOUT_SECONDS"
		maxHeaderBytes    = "HTTP_MAX_HEADER_BYTES"
	)

	tests := []struct {
		name string
		env  map[string]string
		want ServerConfig
	}{
		{
			name: "defaults",
			want: ServerConfig{
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      60 * time.Second,
				IdleTimeout:       60 * time.Second,
				MaxHeaderBytes:    1 << 20,
			},
		},
		{
			name: "valid overrides",
			env: map[string]string{
				readHeaderTimeout: "6",
				readTimeout:       "31",
				writeTimeout:      "61",
				idleTimeout:       "62",
				maxHeaderBytes:    "2097152",
			},
			want: ServerConfig{
				ReadHeaderTimeout: 6 * time.Second,
				ReadTimeout:       31 * time.Second,
				WriteTimeout:      61 * time.Second,
				IdleTimeout:       62 * time.Second,
				MaxHeaderBytes:    2 << 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalConfig(t)
			setHTTPServerEnv(t, tt.env)
			t.Setenv("GIN_MODE", "debug")

			got := Load().Server
			if got.ReadHeaderTimeout != tt.want.ReadHeaderTimeout {
				t.Errorf("ReadHeaderTimeout = %s; want %s", got.ReadHeaderTimeout, tt.want.ReadHeaderTimeout)
			}
			if got.ReadTimeout != tt.want.ReadTimeout {
				t.Errorf("ReadTimeout = %s; want %s", got.ReadTimeout, tt.want.ReadTimeout)
			}
			if got.WriteTimeout != tt.want.WriteTimeout {
				t.Errorf("WriteTimeout = %s; want %s", got.WriteTimeout, tt.want.WriteTimeout)
			}
			if got.IdleTimeout != tt.want.IdleTimeout {
				t.Errorf("IdleTimeout = %s; want %s", got.IdleTimeout, tt.want.IdleTimeout)
			}
			if got.MaxHeaderBytes != tt.want.MaxHeaderBytes {
				t.Errorf("MaxHeaderBytes = %d; want %d", got.MaxHeaderBytes, tt.want.MaxHeaderBytes)
			}
		})
	}
}

func TestLoadRejectsOverflowingHTTPDurationSeconds(t *testing.T) {
	const key = "HTTP_READ_HEADER_TIMEOUT_SECONDS"
	seconds := (int64(^uint64(0)>>1)/int64(time.Second))*2 + 2

	for _, mode := range []string{"debug", "release"} {
		t.Run(mode, func(t *testing.T) {
			resetGlobalConfig(t)
			setHTTPServerEnv(t, map[string]string{key: strconv.FormatInt(seconds, 10)})
			t.Setenv("GIN_MODE", mode)

			err := Load().Validate()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("Validate() error = %v; want overflowing %s to be rejected", err, key)
			}
		})
	}
}

func TestLoadRejectsOverflowingHTTPMaxHeaderBytes(t *testing.T) {
	const key = "HTTP_MAX_HEADER_BYTES"

	for _, mode := range []string{"debug", "release"} {
		t.Run(mode, func(t *testing.T) {
			resetGlobalConfig(t)
			setHTTPServerEnv(t, map[string]string{key: "999999999999999999999999999999999999999"})
			t.Setenv("GIN_MODE", mode)

			err := Load().Validate()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("Validate() error = %v; want overflowing %s to be rejected", err, key)
			}
		})
	}
}

func TestLoadRejectsInvalidHTTPServerSettings(t *testing.T) {
	settings := []string{
		"HTTP_READ_HEADER_TIMEOUT_SECONDS",
		"HTTP_READ_TIMEOUT_SECONDS",
		"HTTP_WRITE_TIMEOUT_SECONDS",
		"HTTP_IDLE_TIMEOUT_SECONDS",
		"HTTP_MAX_HEADER_BYTES",
	}
	invalidValues := []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "malformed", value: "invalid"},
		{name: "zero", value: "0"},
		{name: "negative", value: "-1"},
	}

	for _, mode := range []string{"debug", "release"} {
		for _, key := range settings {
			for _, invalid := range invalidValues {
				t.Run(mode+"/"+key+"/"+invalid.name, func(t *testing.T) {
					resetGlobalConfig(t)
					setHTTPServerEnv(t, map[string]string{key: invalid.value})
					t.Setenv("GIN_MODE", mode)

					err := Load().Validate()
					if err == nil {
						t.Fatal("Validate() error = nil; want invalid HTTP server setting to be rejected")
					}
					if !strings.Contains(err.Error(), key) {
						t.Errorf("Validate() error = %q; want it to identify %s", err, key)
					}
				})
			}
		}
	}
}

func resetGlobalConfig(t *testing.T) {
	t.Helper()
	previous := globalConfig
	globalConfig = nil
	t.Cleanup(func() {
		globalConfig = previous
	})
}

func setHTTPServerEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range []string{
		"HTTP_READ_HEADER_TIMEOUT_SECONDS",
		"HTTP_READ_TIMEOUT_SECONDS",
		"HTTP_WRITE_TIMEOUT_SECONDS",
		"HTTP_IDLE_TIMEOUT_SECONDS",
		"HTTP_MAX_HEADER_BYTES",
	} {
		previous, wasSet := os.LookupEnv(key)
		if value, ok := values[key]; ok {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("set %s: %v", key, err)
			}
		} else if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv(key, previous)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
