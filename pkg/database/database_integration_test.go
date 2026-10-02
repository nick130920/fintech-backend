//go:build integration

package database

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVersionedMigrationsFromEmptyDatabase(t *testing.T) {
	db := openIntegrationDatabase(t)
	resetDisposablePublicSchema(t, db)

	if err := runVersionedMigrations(db, integrationMigrationPath(t)); err != nil {
		t.Fatalf("run versioned migrations: %v", err)
	}

	assertVersionedMigrationState(t, db, 4)
	assertVersionedOnlySchema(t, db)
	assertIncomesCatalog(t, db)
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

	assertVersionedMigrationState(t, db, 4)
	assertIncomesCatalog(t, db)
}

func TestVersionedMigrationsStepDownFromIncomes(t *testing.T) {
	db := openIntegrationDatabase(t)
	resetDisposablePublicSchema(t, db)

	migrationPath := integrationMigrationPath(t)
	if err := runVersionedMigrations(db, migrationPath); err != nil {
		t.Fatalf("run versioned migrations: %v", err)
	}
	stepDownVersionedMigration(t, db, migrationPath)

	assertVersionedMigrationState(t, db, 3)
	assertVersionThreeSchema(t, db)
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

func assertVersionedMigrationState(t *testing.T, db *gorm.DB, wantVersion uint) {
	t.Helper()

	var version uint
	var dirty bool
	if err := db.Raw("SELECT version, dirty FROM schema_migrations LIMIT 1").Row().Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration state: %v", err)
	}
	if version != wantVersion {
		t.Errorf("schema_migrations version = %d, want %d", version, wantVersion)
	}
	if dirty {
		t.Error("schema_migrations is dirty")
	}
}

func assertVersionedOnlySchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	assertVersionedSchema(t, db, append(versionThreeTables(), "incomes"))
}

func assertVersionThreeSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	assertVersionedSchema(t, db, versionThreeTables())

	var incomesExists bool
	if err := db.Raw("SELECT to_regclass('public.incomes') IS NOT NULL").Row().Scan(&incomesExists); err != nil {
		t.Fatalf("check incomes removal: %v", err)
	}
	if incomesExists {
		t.Error("incomes remains after stepping down to version 3")
	}
}

func versionThreeTables() []string {
	return []string{
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
}

func assertVersionedSchema(t *testing.T, db *gorm.DB, expectedTables []string) {
	t.Helper()

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

func stepDownVersionedMigration(t *testing.T, db *gorm.DB, migrationPath string) {
	t.Helper()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		t.Fatalf("create migration driver: %v", err)
	}
	migration, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationPath,
		"postgres",
		driver,
	)
	if err != nil {
		t.Fatalf("open versioned migrations: %v", err)
	}

	if err := migration.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("step down versioned migration: %v", err)
	}
}

type incomeColumn struct {
	name         string
	dataType     string
	nullable     bool
	defaultValue *string
	maxLength    *int64
}

func assertIncomesCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()

	expectedColumns := []incomeColumn{
		{name: "id", dataType: "bigint", nullable: false, defaultValue: stringPointer("nextval('incomes_id_seq'::regclass)")},
		{name: "created_at", dataType: "timestamp with time zone", nullable: true},
		{name: "updated_at", dataType: "timestamp with time zone", nullable: true},
		{name: "deleted_at", dataType: "timestamp with time zone", nullable: true},
		{name: "user_id", dataType: "bigint", nullable: false},
		{name: "amount", dataType: "numeric", nullable: false},
		{name: "description", dataType: "text", nullable: false},
		{name: "source", dataType: "text", nullable: false},
		{name: "date", dataType: "timestamp with time zone", nullable: false},
		{name: "notes", dataType: "text", nullable: true},
		{name: "currency", dataType: "character varying", nullable: true, defaultValue: stringPointer("'USD'::character varying"), maxLength: int64Pointer(3)},
		{name: "is_recurring", dataType: "boolean", nullable: true, defaultValue: stringPointer("false")},
		{name: "frequency", dataType: "character varying", nullable: true, maxLength: int64Pointer(20)},
		{name: "next_date", dataType: "timestamp with time zone", nullable: true},
		{name: "end_date", dataType: "timestamp with time zone", nullable: true},
		{name: "recurring_until", dataType: "timestamp with time zone", nullable: true},
		{name: "tax_deducted", dataType: "numeric", nullable: true, defaultValue: stringPointer("0")},
		{name: "net_amount", dataType: "numeric", nullable: true, defaultValue: stringPointer("0")},
	}

	var columnCount int
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'incomes'
	`).Row().Scan(&columnCount); err != nil {
		t.Fatalf("count incomes columns: %v", err)
	}
	if columnCount != len(expectedColumns) {
		t.Errorf("incomes column count = %d, want %d", columnCount, len(expectedColumns))
	}

	for ordinal, want := range expectedColumns {
		var gotName, gotType, gotNullable string
		var gotDefault sql.NullString
		var gotLength sql.NullInt64
		var numericPrecision, numericScale sql.NullInt64
		if err := db.Raw(`
			SELECT column_name, data_type, is_nullable, column_default,
				character_maximum_length, numeric_precision, numeric_scale
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'incomes' AND ordinal_position = ?
		`, ordinal+1).Row().Scan(&gotName, &gotType, &gotNullable, &gotDefault, &gotLength, &numericPrecision, &numericScale); err != nil {
			t.Errorf("read incomes column %d: %v", ordinal+1, err)
			continue
		}
		if gotName != want.name || gotType != want.dataType {
			t.Errorf("incomes column %d = %s %s, want %s %s", ordinal+1, gotName, gotType, want.name, want.dataType)
		}
		wantNullable := "NO"
		if want.nullable {
			wantNullable = "YES"
		}
		if gotNullable != wantNullable {
			t.Errorf("incomes.%s nullable = %s, want %s", want.name, gotNullable, wantNullable)
		}
		if want.defaultValue == nil {
			if gotDefault.Valid {
				t.Errorf("incomes.%s default = %q, want NULL", want.name, gotDefault.String)
			}
		} else if !gotDefault.Valid || gotDefault.String != *want.defaultValue {
			t.Errorf("incomes.%s default = %q, want %q", want.name, gotDefault.String, *want.defaultValue)
		}
		if want.maxLength == nil {
			if gotLength.Valid {
				t.Errorf("incomes.%s maximum length = %d, want NULL", want.name, gotLength.Int64)
			}
		} else if !gotLength.Valid || gotLength.Int64 != *want.maxLength {
			t.Errorf("incomes.%s maximum length = %d, want %d", want.name, gotLength.Int64, *want.maxLength)
		}
		if want.dataType == "numeric" && (numericPrecision.Valid || numericScale.Valid) {
			t.Errorf("incomes.%s numeric precision/scale = %v/%v, want unqualified NUMERIC", want.name, numericPrecision, numericScale)
		}
	}

	assertIncomesIndexes(t, db)
	assertIncomesForeignKey(t, db)
}

func assertIncomesIndexes(t *testing.T, db *gorm.DB) {
	t.Helper()

	type incomeIndex struct {
		name    string
		unique  bool
		columns string
	}
	expected := []incomeIndex{
		{name: "incomes_pkey", unique: true, columns: "id"},
		{name: "idx_incomes_deleted_at", unique: false, columns: "deleted_at"},
		{name: "idx_incomes_user_id", unique: false, columns: "user_id"},
	}

	rows, err := db.Raw(`
		SELECT index_class.relname, i.indisunique,
			string_agg(pg_get_indexdef(i.indexrelid, key.ordinality, true), ',' ORDER BY key.ordinality)
		FROM pg_index i
		JOIN pg_class index_class ON index_class.oid = i.indexrelid
		JOIN LATERAL generate_series(1, i.indnkeyatts) AS key(ordinality) ON true
		WHERE i.indrelid = 'incomes'::regclass
		GROUP BY index_class.relname, i.indisunique
		ORDER BY index_class.relname
	`).Rows()
	if err != nil {
		t.Fatalf("read incomes indexes: %v", err)
	}
	defer rows.Close()

	var actual []incomeIndex
	for rows.Next() {
		var index incomeIndex
		if err := rows.Scan(&index.name, &index.unique, &index.columns); err != nil {
			t.Fatalf("scan incomes index: %v", err)
		}
		actual = append(actual, index)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate incomes indexes: %v", err)
	}
	if len(actual) != len(expected) {
		t.Errorf("incomes index count = %d, want %d", len(actual), len(expected))
	}
	for _, want := range expected {
		found := false
		for _, got := range actual {
			if got.name != want.name {
				continue
			}
			found = true
			if got.unique != want.unique || got.columns != want.columns {
				t.Errorf("incomes index %s = unique:%t columns:%s, want unique:%t columns:%s", got.name, got.unique, got.columns, want.unique, want.columns)
			}
			break
		}
		if !found {
			t.Errorf("incomes index %s is missing", want.name)
		}
	}
}

func assertIncomesForeignKey(t *testing.T, db *gorm.DB) {
	t.Helper()

	var sourceColumn, targetTable, targetColumn, updateRule, deleteRule string
	if err := db.Raw(`
		SELECT kcu.column_name, kcu2.table_name, kcu2.column_name, rc.update_rule, rc.delete_rule
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_catalog = rc.constraint_catalog
			AND kcu.constraint_schema = rc.constraint_schema
			AND kcu.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage kcu2
			ON kcu2.constraint_catalog = rc.unique_constraint_catalog
			AND kcu2.constraint_schema = rc.unique_constraint_schema
			AND kcu2.constraint_name = rc.unique_constraint_name
		WHERE rc.constraint_schema = 'public' AND rc.constraint_name = 'fk_incomes_user'
	`).Row().Scan(&sourceColumn, &targetTable, &targetColumn, &updateRule, &deleteRule); err != nil {
		t.Fatalf("read incomes foreign key: %v", err)
	}
	if sourceColumn != "user_id" || targetTable != "users" || targetColumn != "id" || updateRule != "NO ACTION" || deleteRule != "NO ACTION" {
		t.Errorf("fk_incomes_user = %s -> %s(%s), update/delete = %s/%s, want user_id -> users(id), NO ACTION/NO ACTION", sourceColumn, targetTable, targetColumn, updateRule, deleteRule)
	}
}

func stringPointer(value string) *string { return &value }

func int64Pointer(value int64) *int64 { return &value }

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
