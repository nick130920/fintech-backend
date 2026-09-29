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

func TestVersionedMigrationsFromEmptyDatabase(t *testing.T) {
	db := openIntegrationDatabase(t)
	resetDisposablePublicSchema(t, db)

	if err := runVersionedMigrations(db, integrationMigrationPath(t)); err != nil {
		t.Fatalf("run versioned migrations: %v", err)
	}

	assertVersionedMigrationState(t, db)
	assertVersionedOnlySchema(t, db)
}

func TestVersionedMigrationsAfterGORMBootstrap(t *testing.T) {
	db := openIntegrationDatabase(t)
	resetDisposablePublicSchema(t, db)

	if err := runMigrations(db); err != nil {
		t.Fatalf("run GORM bootstrap migrations: %v", err)
	}
	if err := runVersionedMigrations(db, integrationMigrationPath(t)); err != nil {
		t.Fatalf("run versioned migrations after GORM bootstrap: %v", err)
	}

	assertVersionedMigrationState(t, db)
}

func openIntegrationDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("CI_DATABASE_URL")
	if dsn == "" {
		t.Skip("CI_DATABASE_URL is required for the PostgreSQL integration test")
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open PostgreSQL connection: %v", err)
	}
	return db
}

func resetDisposablePublicSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("reset disposable public schema: %v", err)
	}
}

func assertVersionedMigrationState(t *testing.T, db *gorm.DB) {
	t.Helper()

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

func assertVersionedOnlySchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	expectedTables := []string{
		"budget_allocations",
		"budgets",
		"categories",
		"expense_splits",
		"expenses",
		"revoked_tokens",
		"settlements",
		"trip_budget_allocations",
		"trip_invitations",
		"trip_itinerary_items",
		"trip_members",
		"trips",
		"users",
	}

	var tableCount int
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_type = 'BASE TABLE'
			AND table_name <> 'schema_migrations'
	`).Row().Scan(&tableCount); err != nil {
		t.Fatalf("count versioned tables: %v", err)
	}
	if tableCount != len(expectedTables) {
		t.Errorf("versioned table count = %d, want %d", tableCount, len(expectedTables))
	}

	for _, table := range expectedTables {
		var exists bool
		if err := db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = ?
			)
		`, table).Row().Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("missing versioned table %s", table)
		}
	}

	assertColumn(t, db, "categories", "is_trip_category", false)
	assertColumn(t, db, "expenses", "trip_id", true)
	assertColumn(t, db, "expenses", "paid_by_member_id", true)
	assertColumn(t, db, "expenses", "budget_id", true)
	assertColumn(t, db, "expenses", "allocation_id", true)

	var ownerForeignKey bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = 'fk_trips_owner'
				AND conrelid = 'trips'::regclass
				AND confrelid = 'users'::regclass
		)
	`).Row().Scan(&ownerForeignKey); err != nil {
		t.Fatalf("check trips owner foreign key: %v", err)
	}
	if !ownerForeignKey {
		t.Error("trips owner foreign key is missing")
	}
}

func assertColumn(t *testing.T, db *gorm.DB, table, column string, nullable bool) {
	t.Helper()

	var isNullable string
	if err := db.Raw(`
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ? AND column_name = ?
	`, table, column).Row().Scan(&isNullable); err != nil {
		t.Fatalf("read %s.%s: %v", table, column, err)
	}
	want := "NO"
	if nullable {
		want = "YES"
	}
	if isNullable != want {
		t.Errorf("%s.%s nullable = %s, want %s", table, column, isNullable, want)
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
