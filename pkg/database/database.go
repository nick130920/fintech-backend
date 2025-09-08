package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/entity"
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

	// Construir DSN (Data Source Name)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbConfig.Host, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode, dbConfig.TimeZone)

	// Configurar logger
	logLevel := logger.Silent
	switch dbConfig.LogLevel {
	case "info":
		logLevel = logger.Info
	case "warn":
		logLevel = logger.Warn
	case "error":
		logLevel = logger.Error
	}

	gormLogger := logger.Default.LogMode(logLevel)

	// Abrir conexión
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
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
		if err := runMigrations(db); err != nil {
			return nil, fmt.Errorf("error al ejecutar migraciones: %v", err)
		}
		log.Println("Migraciones ejecutadas exitosamente")
	}

	log.Printf("Conectado a la base de datos: %s@%s:%s/%s", dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.DBName)
	return db, nil
}

// runMigrations ejecuta las migraciones de la base de datos
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
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
	)
}

// CreateTables crea las tablas manualmente (alternativa a AutoMigrate)
func CreateTables(db *gorm.DB) error {
	// Crear extensiones de PostgreSQL si es necesario
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("Advertencia: No se pudo crear extensión uuid-ossp: %v", err)
	}

	// Ejecutar migraciones
	return runMigrations(db)
}

// DropTables elimina todas las tablas (usar con cuidado)
func DropTables(db *gorm.DB) error {
	return db.Migrator().DropTable(
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
	// Crear categorías por defecto siempre
	if err := createDefaultCategories(db); err != nil {
		log.Printf("Warning: Failed to create default categories: %v", err)
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
				Currency:       "MXN",
				Color:          "#007bff",
				IsActive:       true,
			}

			if err := db.Create(testAccount).Error; err != nil {
				return err
			}

			log.Println("Datos de prueba creados exitosamente")
		}
	}

	return nil
}

// createDefaultCategories crea las categorías predefinidas del sistema
func createDefaultCategories(db *gorm.DB) error {
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
			log.Printf("Warning: Failed to create category %s: %v", category.Name, err)
		}
	}

	log.Printf("Created %d default categories", len(defaultCategories))
	return nil
}
