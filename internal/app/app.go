package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	docs "github.com/nick130920/fintech-backend/api/swagger"
	"github.com/nick130920/fintech-backend/configs"
	v1 "github.com/nick130920/fintech-backend/internal/controller/http/v1"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/nick130920/fintech-backend/pkg/auth"
	"github.com/nick130920/fintech-backend/pkg/database"
	"github.com/nick130920/fintech-backend/pkg/exchange"
	"github.com/nick130920/fintech-backend/pkg/logger"
	"github.com/nick130920/fintech-backend/pkg/observability"
	"github.com/nick130920/fintech-backend/pkg/repository"
)

var appDB *gorm.DB

func startRevokedTokenCleanupWorker(repo interface{ DeleteExpired() error }) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := repo.DeleteExpired(); err != nil {
				// No es fatal, se reintenta en el siguiente ciclo
				continue
			}
		}
	}()
}

func startGmailSyncWorker(uc *usecase.EmailGmailUseCase) {
	if uc == nil || !uc.IsGmailConfigured() {
		return
	}
	go func() {
		ticker := time.NewTicker(20 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
			uc.SyncAllGmailConnections(ctx)
			cancel()
		}
	}()
}

// Run inicializa y ejecuta la aplicación
func Run() {
	// Cargar configuración
	cfg := configs.Load()

	// Configurar logging
	logger.InitLogger(cfg.Logger.Level, cfg.Server.Mode)
	log := logger.Get()

	// Validar configuración
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Configuration validation failed")
	}

	if err := observability.InitSentry(cfg.External.SentryDSN, cfg.Server.Mode); err != nil {
		log.Warn().Err(err).Msg("GlitchTip/Sentry no inicializado (revisa SENTRY_DSN)")
	}
	defer observability.FlushSentry()

	// Configurar modo de Gin
	gin.SetMode(cfg.Server.Mode)

	// Inicializar base de datos
	db, err := database.Initialize()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	appDB = db

	// Crear datos de prueba si es necesario
	if cfg.IsDevelopment() {
		if err := database.Seed(db); err != nil {
			log.Warn().Err(err).Msg("Failed to seed database")
		}
	}

	// Inicializar JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.ExpiresIn)

	// Inicializar dependencias
	deps := initDependencies(cfg, db, jwtManager)

	// Inicializar servidor HTTP
	httpServer := initHTTPServer(cfg, deps, logger.Get())

	// Ejecutar servidor
	runServer(httpServer, db, cfg.Server.Port)
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
	EmailGmailUC              *usecase.EmailGmailUseCase

	// Servicios externos
	AIService        *webapi.AIServiceWithFallback
	ExchangeProvider exchange.Provider

	// Repositories (necesarios para algunos handlers)
	CategoryRepo     repo.CategoryRepo
	RevokedTokenRepo repo.RevokedTokenRepo

	// JWT Manager
	JWTManager *auth.JWTManager
}

// initDependencies inicializa todas las dependencias usando inyección de dependencias
func initDependencies(cfg *configs.Config, db *gorm.DB, jwtManager *auth.JWTManager) *Dependencies {
	log := logger.Get()

	// Inicializar repositorios
	revokedTokenRepo := repository.NewRevokedTokenPostgres(db)
	userRepo := repository.NewUserPostgres(db)
	accountRepo := repository.NewAccountPostgres(db)
	transactionRepo := repository.NewTransactionPostgres(db)
	budgetRepo := repository.NewBudgetPostgres(db)
	categoryRepo := repository.NewCategoryPostgres(db)
	bankAccountRepo := repository.NewBankAccountPostgres(db)
	bankNotificationPatternRepo := repository.NewBankNotificationPatternPostgres(db)
	slugStatsRepo := repository.NewBudgetSuggestionSlugStatsPostgres(db)
	budgetSuggestionJobRepo := repository.NewBudgetSuggestionJobPostgres(db)

	// Inicializar servicios externos de IA (con fallback OpenRouter -> Gemini)
	aiService, err := webapi.NewAIServiceWithFallback()
	if err != nil {
		log.Fatal().Err(err).Msg("Error al inicializar servicios de IA (configure OPENROUTER_API_KEY o GEMINI_API_KEY)")
	}
	if aiService.HasFallback() {
		log.Info().Msg("Servicio de IA configurado con fallback")
	}

	ocrService, err := webapi.NewOCRServiceFromEnv()
	if err != nil {
		log.Warn().Err(err).Msg("No se pudo inicializar OCR service")
	}
	if ocrService == nil {
		log.Warn().Msg("OCR service no configurado (OCR_API_KEY vacío)")
	}

	// Asegurar que existan las categorías por defecto
	if err := categoryRepo.EnsureDefaultCategoriesExist(); err != nil {
		log.Warn().Err(err).Msg("Failed to ensure default categories exist")
	}

	// Crear interfaces necesarias para el caso de uso de presupuesto
	expenseRepo := repository.NewExpensePostgres(db)
	incomeRepo := repository.NewIncomePostgres(db)

	// Inicializar casos de uso
	userUC := usecase.NewUserUseCase(userRepo, revokedTokenRepo, jwtManager)
	accountUC := usecase.NewAccountUseCase(accountRepo, userRepo)
	transactionUC := usecase.NewTransactionUseCase(transactionRepo, accountRepo, userRepo)
	budgetUC := usecase.NewBudgetUseCase(budgetRepo, categoryRepo, expenseRepo, userRepo)
	expenseUC := usecase.NewExpenseUseCase(expenseRepo, budgetRepo, categoryRepo, userRepo)
	incomeUC := usecase.NewIncomeUseCase(incomeRepo, userRepo)
	bankAccountUC := usecase.NewBankAccountUseCase(bankAccountRepo, userRepo)
	bankNotificationPatternUC := usecase.NewBankNotificationPatternUseCase(
		bankNotificationPatternRepo,
		accountRepo,
		bankAccountRepo,
		userRepo,
		transactionRepo,
		expenseRepo,
		incomeRepo,
		budgetRepo,
		categoryRepo,
		slugStatsRepo,
		budgetSuggestionJobRepo,
		aiService,
		ocrService,
	)

	fxProvider := exchange.NewCachedProvider(
		exchange.NewFrankfurterProvider(),
		24*time.Hour,
	)

	emailConnRepo := repository.NewUserEmailConnectionPostgres(db)
	procEmailRepo := repository.NewProcessedEmailMessagePostgres(db)
	emailGmailUC, err := usecase.NewEmailGmailUseCase(cfg, emailConnRepo, procEmailRepo, bankNotificationPatternUC)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init EmailGmailUseCase")
	}

	return &Dependencies{
		UserUC:                    userUC,
		AccountUC:                 accountUC,
		TransactionUC:             transactionUC,
		BudgetUC:                  budgetUC,
		ExpenseUC:                 expenseUC,
		IncomeUC:                  incomeUC,
		BankAccountUC:             bankAccountUC,
		BankNotificationPatternUC: bankNotificationPatternUC,
		EmailGmailUC:              emailGmailUC,
		AIService:                 aiService,
		ExchangeProvider:          fxProvider,
		CategoryRepo:              categoryRepo,
		RevokedTokenRepo:          revokedTokenRepo,
		JWTManager:                jwtManager,
	}
}

// initHTTPServer inicializa el servidor HTTP con todas las rutas
func initHTTPServer(cfg *configs.Config, deps *Dependencies, log zerolog.Logger) *gin.Engine {
	// Crear router
	router := gin.New()

	if strings.TrimSpace(cfg.External.SentryDSN) != "" {
		router.Use(observability.SentryGinMiddleware())
	}

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

	// Prueba manual de GlitchTip/Sentry (solo desarrollo; requiere SENTRY_TEST_SECRET).
	if cfg.Features.EnableDebugRoutes && cfg.IsDevelopment() {
		dsn := strings.TrimSpace(cfg.External.SentryDSN)
		secret := strings.TrimSpace(cfg.External.SentryTestSecret)
		if dsn != "" && secret != "" {
			router.POST("/debug/sentry-test", func(c *gin.Context) {
				if c.GetHeader("X-Sentry-Test-Secret") != secret {
					c.Status(404)
					return
				}
				observability.CaptureGlitchTipTestEvent()
				c.JSON(200, gin.H{
					"ok":      true,
					"message": "Evento de prueba enviado; revisa Issues en GlitchTip en unos segundos.",
				})
			})
		}
	}

	// Inicializar rutas API v1
	v1.NewRouter(router, deps.UserUC, deps.AccountUC, deps.TransactionUC, deps.BudgetUC, deps.ExpenseUC, deps.IncomeUC, deps.BankAccountUC, deps.BankNotificationPatternUC, deps.EmailGmailUC, deps.CategoryRepo, deps.ExchangeProvider, deps.JWTManager, log)

	startGmailSyncWorker(deps.EmailGmailUC)
	startRevokedTokenCleanupWorker(deps.RevokedTokenRepo)

	// Documentación Swagger (solo en desarrollo)
	if cfg.Features.EnableSwagger {
		setupSwagger(router)
	}

	return router
}

// runServer ejecuta el servidor con graceful shutdown
func runServer(router *gin.Engine, db *gorm.DB, port string) {
	log := logger.Get()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Canal para señales del sistema
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ejecutar servidor en goroutine
	go func() {
		log.Info().Msgf("Server starting on port %s", port)
		log.Info().Msgf("Environment: %s", gin.Mode())

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Esperar señal de shutdown
	<-quit
	log.Info().Msg("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Graceful shutdown failed")
	} else {
		log.Info().Msg("HTTP server shut down gracefully")
	}

	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			log.Error().Err(err).Msg("Failed to access SQL DB during shutdown")
		} else if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close DB connection")
		} else {
			log.Info().Msg("Database connection closed")
		}
	}

	log.Info().Msg("Server stopped")
}

// corsMiddleware configura CORS
func corsMiddleware(corsConfig configs.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Verificar si el origen está permitido
		allowed := false
		wildcardAllowed := false

		for _, allowedOrigin := range corsConfig.AllowedOrigins {
			if allowedOrigin == "*" {
				allowed = true
				wildcardAllowed = true
				break
			}
			if allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			// Si es wildcard (*), establecer el origen específico del request
			// porque no se puede usar "*" con credentials=true
			if wildcardAllowed || origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			}
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
	dbStatus := "disconnected"
	status := "degraded"
	pingMs := int64(0)

	if appDB != nil {
		sqlDB, err := appDB.DB()
		if err == nil {
			start := time.Now()
			if err := sqlDB.Ping(); err == nil {
				dbStatus = "connected"
				status = "healthy"
			}
			pingMs = time.Since(start).Milliseconds()
		}
	}

	code := 200
	if status != "healthy" {
		code = 503
	}

	c.JSON(code, gin.H{
		"status":       status,
		"timestamp":    time.Now().Unix(),
		"service":      "fintech-api",
		"version":      "1.0.0",
		"database":     dbStatus,
		"db_ping_ms":   pingMs,
	})
}

// setupSwagger configura la documentación Swagger
func setupSwagger(router *gin.Engine) {
	log := logger.Get()
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

	log.Info().Msgf("Swagger UI available at: http://%s/swagger/index.html", docs.SwaggerInfo.Host)
}

// setupLogging configura logrus para diferentes entornos
func setupLogging() {
	// Configurar formato según entorno
	if gin.Mode() == gin.ReleaseMode {
		// Producción (Railway): formato JSON compacto
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat:   "2006-01-02T15:04:05Z",
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
