package main

import (
	"github.com/joho/godotenv"
	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/app"
	"github.com/nick130920/fintech-backend/pkg/logger"
)

// @title API Fintech
// @version 1.0
// @description API para aplicación de finanzas personales
// @termsOfService http://swagger.io/terms/

// @contact.name Soporte API
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host fintech-production-5841.up.railway.app
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Bearer token para autenticación

func main() {
	// Cargar variables de entorno si existe el archivo .env
	err := godotenv.Load("configs/.env")

	// Es importante inicializar el logger DESPUÉS de cargar .env
	// para que pueda leer la variable LOG_LEVEL correctamente.
	cfg := configs.Load()
	logger.InitLogger(cfg.Logger.Level, cfg.Server.Mode)

	if err != nil {
		log := logger.Get()
		log.Info().Msg("No .env file found, using system environment variables")
	}

	// Ejecutar aplicación
	app.Run()
}
