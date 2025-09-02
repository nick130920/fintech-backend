package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/nick130920/fintech-backend/internal/app"
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
	if err := godotenv.Load("configs/.env"); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Ejecutar aplicación
	app.Run()
}
