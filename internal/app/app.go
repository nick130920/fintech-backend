package app

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	logrus "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	docs "github.com/nick130920/fintech-backend/api/swagger"
	"github.com/nick130920/fintech-backend/configs"
	v1 "github.com/nick130920/fintech-backend/internal/controller/http/v1"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/auth"
	"github.com/nick130920/fintech-backend/pkg/database"
	"github.com/nick130920/fintech-backend/pkg/repository"
)

// Run inicializa y ejecuta la aplicación
func Run() {
	// Cargar configuración
	cfg := configs.Load()

	// Configurar logging para Railway
	setupLogging()

	// Validar configuración
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Configurar modo de Gin
	gin.SetMode(cfg.Server.Mode)

	// Inicializar base de datos
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Crear datos de prueba si es necesario
	if cfg.IsDevelopment() {
		if err := database.Seed(db); err != nil {
			log.Printf("Warning: Failed to seed database: %v", err)
		}
	}

	// Inicializar JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.ExpiresIn)

	// Inicializar dependencias
	deps := initDependencies(db, jwtManager)

	// Inicializar servidor HTTP
	httpServer := initHTTPServer(cfg, deps)

	// Ejecutar servidor
	runServer(httpServer, cfg.Server.Port)
}

// Dependencies contiene todas las dependencias de la aplicación
type Dependencies struct {
	// Use cases
	UserUC                    *usecase.UserUseCase
	AccountUC                 *usecase.AccountUseCase
	TransactionUC             *usecase.TransactionUseCase
	BudgetUC                  *usecase.BudgetUseCase
	ExpenseUC                 *usecase.ExpenseUseCase
	IncomeUC                  *usecase.IncomeUseCase
	BankAccountUC             *usecase.BankAccountUseCase
	BankNotificationPatternUC *usecase.BankNotificationPatternUseCase

	// Repositories (necesarios para algunos handlers)
	CategoryRepo repo.CategoryRepo

	// JWT Manager
	JWTManager *auth.JWTManager
}

// initDependencies inicializa todas las dependencias usando inyección de dependencias
func initDependencies(db *gorm.DB, jwtManager *auth.JWTManager) *Dependencies {
	// Inicializar repositorios
	userRepo := repository.NewUserPostgres(db)
	accountRepo := repository.NewAccountPostgres(db)
	transactionRepo := repository.NewTransactionPostgres(db)
	budgetRepo := repository.NewBudgetPostgres(db)
	categoryRepo := repository.NewCategoryPostgres(db)
	bankAccountRepo := repository.NewBankAccountPostgres(db)
	bankNotificationPatternRepo := repository.NewBankNotificationPatternPostgres(db)

	// Asegurar que existan las categorías por defecto
	if err := categoryRepo.EnsureDefaultCategoriesExist(); err != nil {
		log.Printf("Warning: Failed to ensure default categories exist: %v", err)
	}

	// Crear interfaces necesarias para el caso de uso de presupuesto
	expenseRepo := repository.NewExpensePostgres(db)
	incomeRepo := repository.NewIncomePostgres(db)

	// Inicializar casos de uso
	userUC := usecase.NewUserUseCase(userRepo, jwtManager)
	accountUC := usecase.NewAccountUseCase(accountRepo, userRepo)
	transactionUC := usecase.NewTransactionUseCase(transactionRepo, accountRepo, userRepo)
	budgetUC := usecase.NewBudgetUseCase(budgetRepo, categoryRepo, expenseRepo, userRepo)
	expenseUC := usecase.NewExpenseUseCase(expenseRepo, budgetRepo, categoryRepo, userRepo)
	incomeUC := usecase.NewIncomeUseCase(incomeRepo, userRepo)
	bankAccountUC := usecase.NewBankAccountUseCase(bankAccountRepo, userRepo)
	bankNotificationPatternUC := usecase.NewBankNotificationPatternUseCase(bankNotificationPatternRepo, bankAccountRepo, userRepo)

	return &Dependencies{
		UserUC:                    userUC,
		AccountUC:                 accountUC,
		TransactionUC:             transactionUC,
		BudgetUC:                  budgetUC,
		ExpenseUC:                 expenseUC,
		IncomeUC:                  incomeUC,
		BankAccountUC:             bankAccountUC,
		BankNotificationPatternUC: bankNotificationPatternUC,
		CategoryRepo:              categoryRepo,
		JWTManager:                jwtManager,
	}
}

// initHTTPServer inicializa el servidor HTTP con todas las rutas
func initHTTPServer(cfg *configs.Config, deps *Dependencies) *gin.Engine {
	// Crear router
	router := gin.New()

	// Los middlewares avanzados están integrados en el router v1
	// No necesitamos configurarlos aquí

	// Middleware CORS
	router.Use(corsMiddleware(cfg.CORS))

	// Rutas básicas
	router.GET("/health", healthCheckHandler)
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "fintech-api",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// Inicializar rutas API v1
	v1.NewRouter(router, deps.UserUC, deps.AccountUC, deps.TransactionUC, deps.BudgetUC, deps.ExpenseUC, deps.IncomeUC, deps.BankAccountUC, deps.BankNotificationPatternUC, deps.CategoryRepo, deps.JWTManager)

	// Documentación Swagger (solo en desarrollo)
	if cfg.Features.EnableSwagger {
		setupSwagger(router)
	}

	return router
}

// runServer ejecuta el servidor con graceful shutdown
func runServer(router *gin.Engine, port string) {
	server := router

	// Canal para señales del sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ejecutar servidor en goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Environment: %s", gin.Mode())

		if err := server.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Esperar señal de shutdown
	<-quit
	log.Println("Shutting down server...")

	// TODO: Implementar graceful shutdown cuando sea necesario
	log.Println("Server stopped")
}

// corsMiddleware configura CORS
func corsMiddleware(corsConfig configs.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Verificar si el origen está permitido
		allowed := false
		for _, allowedOrigin := range corsConfig.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// healthCheckHandler maneja el endpoint de health check
func healthCheckHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "OK",
		"timestamp": time.Now().Unix(),
		"service":   "fintech-api",
		"version":   "1.0.0",
	})
}

// setupSwagger configura la documentación Swagger
func setupSwagger(router *gin.Engine) {
	// Configurar host dinámicamente
	cfg := configs.Load()
	if cfg.Server.Host != "localhost" && cfg.Server.Host != "" {
		// Si no es localhost, usar el host de producción
		docs.SwaggerInfo.Host = cfg.Server.Host
	} else {
		// Para desarrollo local
		docs.SwaggerInfo.Host = "localhost:" + cfg.Server.Port
	}

	docs.SwaggerInfo.Title = "API Fintech"
	docs.SwaggerInfo.Description = "API para aplicación de finanzas personales"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Configurar ruta de Swagger
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Swagger UI available at: http://%s/swagger/index.html", docs.SwaggerInfo.Host)
}

// setupLogging configura logrus para diferentes entornos
func setupLogging() {
	// Configurar formato según entorno
	if gin.Mode() == gin.ReleaseMode {
		// Producción (Railway): formato JSON compacto
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z",
			DisableHTMLEscape: true,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "time",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "msg",
			},
		})
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		// Desarrollo: formato texto con colores
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors:     true,
			TimestampFormat: "15:04:05",
			FullTimestamp:   true,
		})
		logrus.SetLevel(logrus.DebugLevel)
	}

	// Siempre usar stdout
	logrus.SetOutput(os.Stdout)
	
	// Log de configuración
	if gin.Mode() == gin.ReleaseMode {
		logrus.Info("🚀 Logging configured for Railway (JSON format)")
	} else {
		logrus.Info("🛠️  Logging configured for development (Text format)")
	}
}
