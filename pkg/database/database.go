package database

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/pkg/logger"
)

// Database representa una conexión a la base de datos
type Database struct {
	*gorm.DB
}

// Initialize inicializa la conexión a la base de datos
func Initialize() (*gorm.DB, error) {
	// Obtener configuración centralizada
	cfg := configs.Load()
	dbConfig := cfg.Database
	log := logger.Get()

	// Construir DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbConfig.Host, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode, dbConfig.TimeZone)

	// Configurar logger
	logLevel := gormlogger.Silent
	switch dbConfig.LogLevel {
	case "info":
		logLevel = gormlogger.Info
	case "warn":
		logLevel = gormlogger.Warn
	case "error":
		logLevel = gormlogger.Error
	}

	gormLogger := gormlogger.Default.LogMode(logLevel)

	// Abrir conexión
	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("error al conectar con la base de datos: %v", err)
	}

	// Configurar pool de conexiones
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error al obtener instancia SQL DB: %v", err)
	}

	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetime) * time.Minute)

	// Ejecutar migraciones
	if dbConfig.AutoMigrate {
		// Transición gradual: habilita golang-migrate versionado sin romper compatibilidad.
		if strings.EqualFold(dbConfig.MigrationEngine, "golang-migrate") {
			if err := runVersionedMigrations(db, dbConfig.MigrationPath); err != nil {
				return nil, fmt.Errorf("error al ejecutar migraciones versionadas: %v", err)
			}
		}
		if err := runMigrations(db); err != nil {
			return nil, fmt.Errorf("error al ejecutar migraciones: %v", err)
		}
		log.Info().Msg("Migrations executed successfully")
	}

	log.Info().
		Str("user", dbConfig.User).
		Str("host", dbConfig.Host).
		Str("port", dbConfig.Port).
		Str("dbname", dbConfig.DBName).
		Msg("Connected to database")
	return db, nil
}

func runVersionedMigrations(db *gorm.DB, migrationPath string) error {
	if migrationPath == "" {
		migrationPath = "file://migrations"
	}
	if !strings.Contains(migrationPath, "://") {
		migrationPath = "file://" + migrationPath
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("error al obtener SQL DB para migraciones: %w", err)
	}

	driver, err := migratepostgres.WithInstance(sqlDB, &migratepostgres.Config{})
	if err != nil {
		return fmt.Errorf("error al inicializar driver postgres para migraciones: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("error al inicializar golang-migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("error ejecutando golang-migrate up: %w", err)
	}

	return nil
}

// runMigrations ejecuta las migraciones de la base de datos
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.RevokedToken{},
		&entity.User{},
		&entity.Category{},
		&entity.Budget{},
		&entity.BudgetAllocation{},
		&entity.Expense{},
		&entity.Income{},
		// Opcional: mantener Account y Transaction para compatibilidad
		&entity.Account{},
		&entity.Transaction{},
		// Nuevas entidades para notificaciones bancarias
		&entity.BankAccount{},
		&entity.BankNotificationPattern{},
		&entity.BudgetSuggestionSlugStat{},
		&entity.BudgetSuggestionJob{},
		&entity.PendingNotification{},
		&entity.UserEmailConnection{},
		&entity.ProcessedEmailMessage{},
		// Modulo de viajes (turismo)
		&entity.Trip{},
		&entity.TripMember{},
		&entity.TripInvitation{},
		&entity.TripBudgetAllocation{},
		&entity.ExpenseSplit{},
		&entity.Settlement{},
		&entity.TripItineraryItem{},
	)
}

// CreateTables crea las tablas manualmente (alternativa a AutoMigrate)
func CreateTables(db *gorm.DB) error {
	log := logger.Get()
	// Crear extensiones de PostgreSQL si es necesario
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Warn().Err(err).Msg("Could not create uuid-ossp extension")
	}

	// Ejecutar migraciones
	return runMigrations(db)
}

// DropTables elimina todas las tablas (usar con cuidado)
func DropTables(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&entity.TripItineraryItem{},
		&entity.Settlement{},
		&entity.ExpenseSplit{},
		&entity.TripBudgetAllocation{},
		&entity.TripInvitation{},
		&entity.TripMember{},
		&entity.Trip{},
		&entity.BudgetSuggestionJob{},
		&entity.BudgetSuggestionSlugStat{},
		&entity.PendingNotification{},
		&entity.ProcessedEmailMessage{},
		&entity.UserEmailConnection{},
		// Eliminar en orden inverso por dependencias
		&entity.BankNotificationPattern{}, // Depende de BankAccount
		&entity.Expense{},
		&entity.BudgetAllocation{},
		&entity.Budget{},
		&entity.Transaction{}, // Actualizada con referencia a BankAccount
		&entity.BankAccount{}, // Depende de User
		&entity.Account{},
		&entity.Category{},
		&entity.User{},
	)
}

// Seed llena la base de datos con datos de prueba
func Seed(db *gorm.DB) error {
	log := logger.Get()
	// Crear categorías por defecto siempre
	if err := createDefaultCategories(db); err != nil {
		log.Warn().Err(err).Msg("Failed to create default categories")
	}

	// Solo para desarrollo - crear usuario de prueba
	cfg := configs.Load()
	if cfg.IsDevelopment() {
		var count int64
		db.Model(&entity.User{}).Count(&count)

		if count == 0 {
			testUser := &entity.User{
				FirstName:  "Usuario",
				LastName:   "Prueba",
				Email:      "test@fintech.com",
				Password:   "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
				IsActive:   true,
				IsVerified: true,
			}

			if err := db.Create(testUser).Error; err != nil {
				return err
			}

			// Crear cuenta de prueba
			testAccount := &entity.Account{
				UserID:         testUser.ID,
				Name:           "Cuenta Principal",
				Type:           entity.AccountTypeChecking,
				InitialBalance: 10000.00,
				Balance:        10000.00,
				Currency:       "USD",
				Color:          "#007bff",
				IsActive:       true,
			}

			if err := db.Create(testAccount).Error; err != nil {
				return err
			}

			log.Info().Msg("Test data created successfully")
		}
	}

	return nil
}

// createDefaultCategories crea las categorías predefinidas del sistema
func createDefaultCategories(db *gorm.DB) error {
	log := logger.Get()
	// Verificar si ya existen categorías por defecto
	var count int64
	db.Model(&entity.Category{}).Where("is_default = ?", true).Count(&count)

	if count > 0 {
		return nil // Ya existen
	}

	// Crear categorías por defecto
	defaultCategories := entity.DefaultCategories()
	for _, category := range defaultCategories {
		if err := db.Create(&category).Error; err != nil {
			log.Warn().Err(err).Str("category_name", category.Name).Msg("Failed to create category")
		}
	}

	log.Info().Int("count", len(defaultCategories)).Msg("Created default categories")
	return nil
}
