//go:build integration

package database

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVersionedMigrationsAfterLegacyBootstrap(t *testing.T) {
	dsn := os.Getenv("CI_DATABASE_URL")
	if dsn == "" {
		t.Skip("CI_DATABASE_URL is required for the PostgreSQL integration test")
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open PostgreSQL connection: %v", err)
	}

	if err := runMigrations(db); err != nil {
		t.Fatalf("run legacy migrations: %v", err)
	}
	if err := runVersionedMigrations(db, integrationMigrationPath(t)); err != nil {
		t.Fatalf("run versioned migrations: %v", err)
	}

	var version uint
	var dirty bool
	if err := db.Raw("SELECT version, dirty FROM schema_migrations LIMIT 1").Row().Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration state: %v", err)
	}
	if version != 3 {
		t.Errorf("schema_migrations version = %d, want 3", version)
	}
	if dirty {
		t.Error("schema_migrations is dirty")
	}
}

func integrationMigrationPath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}
