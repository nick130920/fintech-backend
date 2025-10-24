package main

import (
	"fmt"
	"log"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/pkg/database"
	"github.com/nick130920/fintech-backend/pkg/repository"
)

// Patrones de notificación por defecto para bancos colombianos
var defaultPatterns = []entity.BankNotificationPattern{
	{
		Name:                "Bancolombia - Compra",
		Description:         "Patrón para notificaciones de compras de Bancolombia",
		Channel:             "sms",
		Status:              "active",
		MessagePattern:      "Compra por \\$([0-9,]+) en (.+) el ([0-9/]+)",
		ExampleMessage:      "Compra por $50,000 en SUPERMERCADO XYZ el 15/01/2024",
		AmountRegex:         "\\$([0-9,]+)",
		DateRegex:           "el ([0-9/]+)",
		MerchantRegex:       "en (.+) el",
		RequiresValidation:  false,
		ConfidenceThreshold: 0.8,
		AutoApprove:         true,
		Priority:            1,
		IsDefault:           true,
	},
	{
		Name:                "Bancolombia - Retiro",
		Description:         "Patrón para notificaciones de retiros de Bancolombia",
		Channel:             "sms",
		Status:              "active",
		MessagePattern:      "Retiro por \\$([0-9,]+) en (.+) el ([0-9/]+)",
		ExampleMessage:      "Retiro por $100,000 en CAJERO AUTOMATICO el 15/01/2024",
		AmountRegex:         "\\$([0-9,]+)",
		DateRegex:           "el ([0-9/]+)",
		MerchantRegex:       "en (.+) el",
		RequiresValidation:  false,
		ConfidenceThreshold: 0.8,
		AutoApprove:         true,
		Priority:            1,
		IsDefault:           true,
	},
	{
		Name:                "Davivienda - Compra",
		Description:         "Patrón para notificaciones de compras de Davivienda",
		Channel:             "sms",
		Status:              "active",
		MessagePattern:      "Transaccion aprobada por \\$([0-9,]+) en (.+) ([0-9/]+)",
		ExampleMessage:      "Transaccion aprobada por $75,500 en TIENDA ABC 15/01/2024",
		AmountRegex:         "\\$([0-9,]+)",
		DateRegex:           "([0-9/]+)$",
		MerchantRegex:       "en (.+) [0-9/]+",
		RequiresValidation:  false,
		ConfidenceThreshold: 0.8,
		AutoApprove:         true,
		Priority:            1,
		IsDefault:           true,
	},
	{
		Name:                "Nequi - Pago",
		Description:         "Patrón para notificaciones de pagos de Nequi",
		Channel:             "push",
		Status:              "active",
		MessagePattern:      "Pagaste \\$([0-9,]+) a (.+)",
		ExampleMessage:      "Pagaste $25,000 a UBER",
		AmountRegex:         "\\$([0-9,]+)",
		MerchantRegex:       "a (.+)$",
		RequiresValidation:  false,
		ConfidenceThreshold: 0.9,
		AutoApprove:         true,
		Priority:            1,
		IsDefault:           true,
	},
	{
		Name:                "Banco de Bogotá - Compra",
		Description:         "Patrón para notificaciones de compras del Banco de Bogotá",
		Channel:             "sms",
		Status:              "active",
		MessagePattern:      "Compra: \\$([0-9,]+) (.+) ([0-9/]+)",
		ExampleMessage:      "Compra: $45,000 RESTAURANTE LA PLAZA 15/01/2024",
		AmountRegex:         "\\$([0-9,]+)",
		DateRegex:           "([0-9/]+)$",
		MerchantRegex:       "\\$[0-9,]+ (.+) [0-9/]+",
		RequiresValidation:  false,
		ConfidenceThreshold: 0.8,
		AutoApprove:         true,
		Priority:            1,
		IsDefault:           true,
	},
}

func main() {
	// Cargar configuración
	cfg := configs.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Conectar a la base de datos
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Crear repositorio
	patternRepo := repository.NewBankNotificationPatternPostgres(db)

	fmt.Println("🚀 Creando patrones de notificación por defecto...")

	for i, pattern := range defaultPatterns {
		// Verificar si ya existe un patrón similar (por ahora saltamos esta verificación)
		// TODO: Implementar método GetByName en el repositorio
		// existing, err := patternRepo.GetByName(pattern.Name)
		// if err == nil && existing != nil {
		//	fmt.Printf("⚠️  Patrón '%s' ya existe, saltando...\n", pattern.Name)
		//	continue
		// }

		// Crear el patrón
		if err := patternRepo.Create(&pattern); err != nil {
			log.Printf("❌ Error creando patrón '%s': %v", pattern.Name, err)
			continue
		}

		fmt.Printf("✅ Patrón %d/%d creado: %s\n", i+1, len(defaultPatterns), pattern.Name)
	}

	fmt.Println("🎉 ¡Patrones por defecto creados exitosamente!")
	fmt.Println("\n📋 Patrones disponibles:")
	fmt.Println("- Bancolombia (Compras y Retiros)")
	fmt.Println("- Davivienda (Compras)")
	fmt.Println("- Nequi (Pagos)")
	fmt.Println("- Banco de Bogotá (Compras)")
	fmt.Println("\n💡 Ahora puedes recibir notificaciones automáticamente y crear transacciones!")
}
