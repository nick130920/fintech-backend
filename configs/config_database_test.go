package configs

import "testing"

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
