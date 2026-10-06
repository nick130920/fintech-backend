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
	assertUsersDefaultAccountCatalog(t, db)
	assertVersionedForeignKeys(t, db)
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
}

func assertUsersDefaultAccountCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()

	assertColumn(t, db, "users", "default_account_id", true)
	assertIndexColumns(t, db, "users", "idx_users_default_account_id", "default_account_id")
	assertForeignKeySourceCount(t, db, "users", "default_account_id", 0)
}

func assertVersionedForeignKeys(t *testing.T, db *gorm.DB) {
	t.Helper()

	assertForeignKey(t, db, "fk_trips_owner", "trips", "owner_user_id", "users", "id", "NO ACTION", "CASCADE")
	assertForeignKey(t, db, "fk_expenses_trip", "expenses", "trip_id", "trips", "id", "NO ACTION", "SET NULL")
	assertForeignKey(t, db, "fk_expenses_paid_by", "expenses", "paid_by_member_id", "trip_members", "id", "NO ACTION", "SET NULL")
	assertForeignKeyMappingCount(t, db, "expenses", "trip_id", "trips", "id", 1)
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

type catalogColumn struct {
	name         string
	dataType     string
	nullable     bool
	defaultValue *string
	maxLength    *int64
	precision    *int64
	scale        *int64
}

type catalogIndex struct {
	name    string
	unique  bool
	columns string
}

type catalogTable struct {
	name    string
	columns []catalogColumn
	indexes []catalogIndex
}

func assertIncomesCatalog(t *testing.T, db *gorm.DB) {
	t.Helper()

	assertTableCatalog(t, db, catalogTable{
		name: "incomes",
		columns: []catalogColumn{
			{name: "id", dataType: "bigint", nullable: false, defaultValue: stringPointer("nextval('incomes_id_seq'::regclass)"), precision: int64Pointer(64), scale: int64Pointer(0)},
			{name: "created_at", dataType: "timestamp with time zone", nullable: true},
			{name: "updated_at", dataType: "timestamp with time zone", nullable: true},
			{name: "deleted_at", dataType: "timestamp with time zone", nullable: true},
			{name: "user_id", dataType: "bigint", nullable: false, precision: int64Pointer(64), scale: int64Pointer(0)},
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
		},
		indexes: []catalogIndex{
			{name: "incomes_pkey", unique: true, columns: "id"},
			{name: "idx_incomes_deleted_at", unique: false, columns: "deleted_at"},
			{name: "idx_incomes_user_id", unique: false, columns: "user_id"},
		},
	})
	assertIncomesForeignKey(t, db)
}

func assertTableCatalog(t *testing.T, db *gorm.DB, table catalogTable) {
	t.Helper()

	var columnCount int
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ?
	`, table.name).Row().Scan(&columnCount); err != nil {
		t.Fatalf("count %s columns: %v", table.name, err)
	}
	if columnCount != len(table.columns) {
		t.Errorf("%s column count = %d, want %d", table.name, columnCount, len(table.columns))
	}
	for ordinal, want := range table.columns {
		assertCatalogColumn(t, db, table.name, ordinal+1, want)
	}
	assertCatalogIndexes(t, db, table)
}

func assertCatalogColumn(t *testing.T, db *gorm.DB, table string, ordinal int, want catalogColumn) {
	t.Helper()

	var gotName, gotType, gotNullable string
	var gotDefault sql.NullString
	var gotLength, gotPrecision, gotScale sql.NullInt64
	if err := db.Raw(`
		SELECT column_name, data_type, is_nullable, column_default,
			character_maximum_length, numeric_precision, numeric_scale
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ? AND ordinal_position = ?
	`, table, ordinal).Row().Scan(&gotName, &gotType, &gotNullable, &gotDefault, &gotLength, &gotPrecision, &gotScale); err != nil {
		t.Errorf("read %s column %d: %v", table, ordinal, err)
		return
	}
	if gotName != want.name || gotType != want.dataType {
		t.Errorf("%s column %d = %s %s, want %s %s", table, ordinal, gotName, gotType, want.name, want.dataType)
	}
	wantNullable := "NO"
	if want.nullable {
		wantNullable = "YES"
	}
	if gotNullable != wantNullable {
		t.Errorf("%s.%s nullable = %s, want %s", table, want.name, gotNullable, wantNullable)
	}
	assertCatalogString(t, table+"."+want.name+" default", gotDefault, want.defaultValue)
	assertCatalogInt64(t, table+"."+want.name+" maximum length", gotLength, want.maxLength)
	assertCatalogInt64(t, table+"."+want.name+" numeric precision", gotPrecision, want.precision)
	assertCatalogInt64(t, table+"."+want.name+" numeric scale", gotScale, want.scale)
}

func assertCatalogString(t *testing.T, description string, got sql.NullString, want *string) {
	t.Helper()

	if want == nil {
		if got.Valid {
			t.Errorf("%s = %q, want NULL", description, got.String)
		}
		return
	}
	if !got.Valid || got.String != *want {
		t.Errorf("%s = %q, want %q", description, got.String, *want)
	}
}

func assertCatalogInt64(t *testing.T, description string, got sql.NullInt64, want *int64) {
	t.Helper()

	if want == nil {
		if got.Valid {
			t.Errorf("%s = %d, want NULL", description, got.Int64)
		}
		return
	}
	if !got.Valid || got.Int64 != *want {
		t.Errorf("%s = %d, want %d", description, got.Int64, *want)
	}
}

func assertCatalogIndexes(t *testing.T, db *gorm.DB, table catalogTable) {
	t.Helper()

	rows, err := db.Raw(`
		SELECT index_class.relname, i.indisunique,
			string_agg(pg_get_indexdef(i.indexrelid, key.ordinality, true), ',' ORDER BY key.ordinality)
		FROM pg_index i
		JOIN pg_class index_class ON index_class.oid = i.indexrelid
		JOIN LATERAL generate_series(1, i.indnkeyatts) AS key(ordinality) ON true
		WHERE i.indrelid = (?::text)::regclass
		GROUP BY index_class.relname, i.indisunique
		ORDER BY index_class.relname
	`, "public."+table.name).Rows()
	if err != nil {
		t.Fatalf("read %s indexes: %v", table.name, err)
	}
	defer rows.Close()

	var actual []catalogIndex
	for rows.Next() {
		var index catalogIndex
		if err := rows.Scan(&index.name, &index.unique, &index.columns); err != nil {
			t.Fatalf("scan %s index: %v", table.name, err)
		}
		actual = append(actual, index)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s indexes: %v", table.name, err)
	}
	if len(actual) != len(table.indexes) {
		t.Errorf("%s index count = %d, want %d", table.name, len(actual), len(table.indexes))
	}
	for _, want := range table.indexes {
		found := false
		for _, got := range actual {
			if got.name != want.name {
				continue
			}
			found = true
			if got.unique != want.unique || got.columns != want.columns {
				t.Errorf("%s index %s = unique:%t columns:%s, want unique:%t columns:%s", table.name, got.name, got.unique, got.columns, want.unique, want.columns)
			}
			break
		}
		if !found {
			t.Errorf("%s index %s is missing", table.name, want.name)
		}
	}
}

func assertIncomesForeignKey(t *testing.T, db *gorm.DB) {
	t.Helper()
	assertForeignKey(t, db, "fk_incomes_user", "incomes", "user_id", "users", "id", "NO ACTION", "NO ACTION")
}

func assertIndexColumns(t *testing.T, db *gorm.DB, table, indexName, wantColumns string) {
	t.Helper()

	var keyColumns, indexColumns, matchingIndexes int
	var unique, valid, ready, hasPredicate, hasExpressions bool
	var gotColumns string
	if err := db.Raw(`
		SELECT i.indnkeyatts, i.indnatts, i.indisunique, i.indisvalid, i.indisready,
			i.indpred IS NOT NULL, i.indexprs IS NOT NULL, COUNT(*) OVER (),
			string_agg(pg_get_indexdef(i.indexrelid, key.ordinality, true), ',' ORDER BY key.ordinality)
		FROM pg_index i
		JOIN pg_class table_class ON table_class.oid = i.indrelid
		JOIN pg_namespace table_schema ON table_schema.oid = table_class.relnamespace
		JOIN pg_class index_class ON index_class.oid = i.indexrelid
		JOIN pg_namespace index_schema ON index_schema.oid = index_class.relnamespace
		JOIN LATERAL generate_series(1, i.indnkeyatts) AS key(ordinality) ON true
		WHERE table_schema.nspname = 'public'
			AND index_schema.nspname = 'public'
			AND table_class.relname = ?
			AND index_class.relname = ?
		GROUP BY i.indexrelid
	`, table, indexName).Row().Scan(&keyColumns, &indexColumns, &unique, &valid, &ready, &hasPredicate, &hasExpressions, &matchingIndexes, &gotColumns); err != nil {
		t.Fatalf("read index %s on %s: %v", indexName, table, err)
	}
	if matchingIndexes != 1 {
		t.Errorf("index %s on %s has %d matching definitions, want 1", indexName, table, matchingIndexes)
	}
	if keyColumns != 1 || indexColumns != 1 || gotColumns != wantColumns {
		t.Errorf("index %s on %s has %d key columns, %d total columns, columns %s; want exactly %s", indexName, table, keyColumns, indexColumns, gotColumns, wantColumns)
	}
	if unique || !valid || !ready || hasPredicate || hasExpressions {
		t.Errorf("index %s on %s = unique:%t valid:%t ready:%t partial:%t expressions:%t; want unique:false valid:true ready:true partial:false expressions:false", indexName, table, unique, valid, ready, hasPredicate, hasExpressions)
	}
}

func assertForeignKey(t *testing.T, db *gorm.DB, name, sourceTable, sourceColumn, targetTable, targetColumn, wantUpdateRule, wantDeleteRule string) {
	t.Helper()

	var gotSourceColumn, gotTargetTable, gotTargetColumn, gotUpdateRule, gotDeleteRule string
	var matchingConstraints int
	if err := db.Raw(`
		SELECT source_column.attname, target_table.relname, target_column.attname,
			CASE foreign_key.confupdtype WHEN 'a' THEN 'NO ACTION' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'r' THEN 'RESTRICT' WHEN 'd' THEN 'SET DEFAULT' END,
			CASE foreign_key.confdeltype WHEN 'a' THEN 'NO ACTION' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'r' THEN 'RESTRICT' WHEN 'd' THEN 'SET DEFAULT' END,
			COUNT(*) OVER ()
		FROM pg_constraint foreign_key
		JOIN pg_class source_table ON source_table.oid = foreign_key.conrelid
		JOIN pg_namespace source_schema ON source_schema.oid = source_table.relnamespace
		JOIN pg_class target_table ON target_table.oid = foreign_key.confrelid
		JOIN pg_namespace target_schema ON target_schema.oid = target_table.relnamespace
		JOIN LATERAL unnest(foreign_key.conkey) WITH ORDINALITY AS source_key(attnum, ordinality) ON true
		JOIN LATERAL unnest(foreign_key.confkey) WITH ORDINALITY AS target_key(attnum, ordinality)
			ON target_key.ordinality = source_key.ordinality
		JOIN pg_attribute source_column ON source_column.attrelid = foreign_key.conrelid AND source_column.attnum = source_key.attnum
		JOIN pg_attribute target_column ON target_column.attrelid = foreign_key.confrelid AND target_column.attnum = target_key.attnum
		WHERE foreign_key.conname = ?
			AND source_schema.nspname = 'public'
			AND target_schema.nspname = 'public'
			AND source_table.relname = ?
			AND target_table.relname = ?
			AND array_length(foreign_key.conkey, 1) = 1
			AND array_length(foreign_key.confkey, 1) = 1
	`, name, sourceTable, targetTable).Row().Scan(&gotSourceColumn, &gotTargetTable, &gotTargetColumn, &gotUpdateRule, &gotDeleteRule, &matchingConstraints); err != nil {
		t.Fatalf("read foreign key %s: %v", name, err)
	}
	if matchingConstraints != 1 {
		t.Errorf("foreign key %s has %d matching constraints, want 1", name, matchingConstraints)
	}
	if gotSourceColumn != sourceColumn || gotTargetTable != targetTable || gotTargetColumn != targetColumn || gotUpdateRule != wantUpdateRule || gotDeleteRule != wantDeleteRule {
		t.Errorf("foreign key %s = %s -> %s(%s), update/delete = %s/%s; want %s -> %s(%s), %s/%s", name, gotSourceColumn, gotTargetTable, gotTargetColumn, gotUpdateRule, gotDeleteRule, sourceColumn, targetTable, targetColumn, wantUpdateRule, wantDeleteRule)
	}
}

func assertForeignKeySourceCount(t *testing.T, db *gorm.DB, table, column string, want int) {
	t.Helper()

	assertForeignKeyCount(t, db, table, column, "", "", want)
}

func assertForeignKeyMappingCount(t *testing.T, db *gorm.DB, sourceTable, sourceColumn, targetTable, targetColumn string, want int) {
	t.Helper()

	assertForeignKeyCount(t, db, sourceTable, sourceColumn, targetTable, targetColumn, want)
}

func assertForeignKeyCount(t *testing.T, db *gorm.DB, sourceTable, sourceColumn, targetTable, targetColumn string, want int) {
	t.Helper()

	var got int
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM pg_constraint foreign_key
		JOIN pg_class source_table_class ON source_table_class.oid = foreign_key.conrelid
		JOIN pg_namespace source_schema ON source_schema.oid = source_table_class.relnamespace
		JOIN pg_class target_table_class ON target_table_class.oid = foreign_key.confrelid
		JOIN pg_namespace target_schema ON target_schema.oid = target_table_class.relnamespace
		JOIN pg_attribute source_column_attribute
			ON source_column_attribute.attrelid = foreign_key.conrelid
			AND source_column_attribute.attname = ?
		LEFT JOIN pg_attribute target_column_attribute
			ON target_column_attribute.attrelid = foreign_key.confrelid
			AND target_column_attribute.attname = ?
		WHERE foreign_key.contype = 'f'
			AND source_schema.nspname = 'public'
			AND source_table_class.relname = ?
			AND source_column_attribute.attnum = ANY(foreign_key.conkey)
			AND (? = '' OR (target_schema.nspname = 'public' AND target_table_class.relname = ?))
			AND (? = '' OR foreign_key.confkey[array_position(foreign_key.conkey, source_column_attribute.attnum)] = target_column_attribute.attnum)
	`, sourceColumn, targetColumn, sourceTable, targetTable, targetTable, targetColumn).Row().Scan(&got); err != nil {
		t.Fatalf("count foreign keys from %s.%s: %v", sourceTable, sourceColumn, err)
	}
	if got != want {
		t.Errorf("foreign key count from %s.%s to %s.%s = %d, want %d", sourceTable, sourceColumn, targetTable, targetColumn, got, want)
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
