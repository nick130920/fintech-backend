package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/auth"
	"github.com/nick130920/fintech-backend/pkg/exchange"
)

// NewRouter inicializa todas las rutas de la API v1
func NewRouter(
	router *gin.Engine,
	userUC *usecase.UserUseCase,
	accountUC *usecase.AccountUseCase,
	transactionUC *usecase.TransactionUseCase,
	budgetUC *usecase.BudgetUseCase,
	expenseUC *usecase.ExpenseUseCase,
	incomeUC *usecase.IncomeUseCase,
	bankAccountUC *usecase.BankAccountUseCase,
	bankNotificationPatternUC *usecase.BankNotificationPatternUseCase,
	emailGmailUC *usecase.EmailGmailUseCase,
	categoryRepo repo.CategoryRepo,
	exchangeProvider exchange.Provider,
	jwtManager *auth.JWTManager,
	logger zerolog.Logger,
) {
	// Configurar middlewares globales de seguridad
	setupGlobalMiddlewares(router)

	// Rutas públicas (sin autenticación)
	setupPublicRoutes(router)

	// Grupo principal de API v1
	v1 := router.Group("/api/v1")

	// Configurar middlewares específicos de API
	setupAPIMiddlewares(v1)

	// Inicializar handlers
	userHandler := NewUserHandler(userUC)
	accountHandler := NewAccountHandler(accountUC)
	transactionHandler := NewTransactionHandler(transactionUC, logger)
	budgetHandler := NewBudgetHandler(budgetUC, logger)
	expenseHandler := NewExpenseHandler(expenseUC, logger)
	incomeHandler := NewIncomeHandler(incomeUC, logger)
	bankAccountHandler := NewBankAccountHandler(bankAccountUC, logger)
	bankNotificationPatternHandler := NewBankNotificationPatternHandler(bankNotificationPatternUC, logger)
	emailConnectionHandler := NewEmailConnectionHandler(emailGmailUC, logger)
	webhookHandler := NewWebhookHandler(bankNotificationPatternUC, transactionUC, logger)
	categoryHandler := NewCategoryHandler(categoryRepo, logger)
	currencyHandler := NewCurrencyHandler(exchangeProvider)

	// Public currency endpoints (no auth required, cacheable)
	v1.GET("/currencies", currencyHandler.GetCurrencies)
	v1.GET("/exchange-rates", currencyHandler.GetExchangeRates)

	// Geo: detecta país y moneda sugerida por IP del cliente (sin auth, sin permisos)
	v1.GET("/geo/country", GetCountryFromIP)

	// OAuth Gmail: callback público (sin JWT)
	v1.GET("/email-connections/gmail/callback", emailConnectionHandler.GmailOAuthCallback)

	// Middleware de autenticación
	authMiddleware := NewAuthMiddleware(jwtManager)

	// Rutas de autenticación (públicas)
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/refresh", userHandler.RefreshToken)
		authGroup.POST("/forgot-password", userHandler.ForgotPassword)
		authGroup.POST("/reset-password", userHandler.ResetPassword)

		// Rutas de auth que requieren token válido
		authProtected := authGroup.Group("/")
		authProtected.Use(authMiddleware.RequireAuth())
		{
			authProtected.POST("/logout", userHandler.Logout)
			authProtected.GET("/validate", userHandler.ValidateToken)
		}
	}

	// Rutas protegidas (requieren autenticación)
	protectedGroup := v1.Group("/")
	protectedGroup.Use(authMiddleware.RequireAuth())
	{
		// Rutas de usuarios
		usersGroup := protectedGroup.Group("/users")
		{
			usersGroup.GET("/profile", userHandler.GetProfile)
			usersGroup.PUT("/profile", userHandler.UpdateProfile)
			usersGroup.PUT("/preferences", userHandler.UpdatePreferences)
		}

		// Rutas de cuentas
		accountsGroup := protectedGroup.Group("/accounts")
		{
			accountsGroup.GET("/", accountHandler.GetAccounts)
			accountsGroup.POST("/", accountHandler.CreateAccount)
			accountsGroup.GET("/:id", accountHandler.GetAccount)
			accountsGroup.PUT("/:id", accountHandler.UpdateAccount)
			accountsGroup.DELETE("/:id", accountHandler.DeleteAccount)
			accountsGroup.GET("/summaries", accountHandler.GetAccountSummaries)
			accountsGroup.GET("/balance/total", accountHandler.GetTotalBalance)
		}

		// Rutas de transacciones
		transactionsGroup := protectedGroup.Group("/transactions")
		{
			transactionsGroup.GET("/", transactionHandler.GetTransactions)
			transactionsGroup.GET("", transactionHandler.GetTransactions)
			transactionsGroup.POST("/", transactionHandler.CreateTransaction)
			transactionsGroup.POST("", transactionHandler.CreateTransaction)
			transactionsGroup.GET("/:id", transactionHandler.GetTransaction)
			transactionsGroup.PUT("/:id", transactionHandler.UpdateTransaction)
			transactionsGroup.DELETE("/:id", transactionHandler.DeleteTransaction)
			transactionsGroup.POST("/:id/cancel", transactionHandler.CancelTransaction)
			transactionsGroup.GET("/recent", transactionHandler.GetRecentTransactions)
			transactionsGroup.GET("/totals", transactionHandler.GetTotalsByType)
		}

		// Rutas de presupuestos
		budgetsGroup := protectedGroup.Group("/budgets")
		{
			budgetsGroup.POST("/", budgetHandler.CreateBudget)
			budgetsGroup.GET("/current", budgetHandler.GetCurrentBudget)
			budgetsGroup.GET("/month", budgetHandler.GetBudgetByMonth)
			budgetsGroup.PUT("/:id", budgetHandler.UpdateBudget)
			budgetsGroup.GET("/dashboard", budgetHandler.GetDashboard)
			budgetsGroup.POST("/rollover", budgetHandler.ProcessDailyRollover)

			// Rutas de asignaciones de presupuesto
			budgetsGroup.PUT("/allocations/:id", budgetHandler.UpdateAllocation)
		}

		// Rutas de categorías
		categoriesGroup := protectedGroup.Group("/categories")
		{
			categoriesGroup.GET("/", categoryHandler.GetCategories)
			categoriesGroup.GET("", categoryHandler.GetCategories)
			categoriesGroup.POST("/", categoryHandler.CreateCategory)
			categoriesGroup.POST("", categoryHandler.CreateCategory)
			categoriesGroup.PUT("/:id", categoryHandler.UpdateCategory)
			categoriesGroup.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		// Rutas de gastos
		expensesGroup := protectedGroup.Group("/expenses")
		{
			expensesGroup.POST("/", expenseHandler.CreateExpense)
			expensesGroup.GET("/", expenseHandler.GetExpenses)
			expensesGroup.GET("/recent", expenseHandler.GetRecentExpenses)
			expensesGroup.GET("/automatic", expenseHandler.GetAutomaticExpenses)
			expensesGroup.GET("/by-category", expenseHandler.GetExpensesByCategory)
			expensesGroup.PUT("/:id", expenseHandler.UpdateExpense)
			expensesGroup.DELETE("/:id", expenseHandler.DeleteExpense)
			expensesGroup.POST("/:id/confirm", expenseHandler.ConfirmExpense)
			expensesGroup.POST("/:id/reject", expenseHandler.RejectExpense)
		}

		// Rutas de ingresos
		incomesGroup := protectedGroup.Group("/incomes")
		{
			incomesGroup.POST("/", incomeHandler.CreateIncome)
			incomesGroup.GET("/", incomeHandler.GetIncomes)
			incomesGroup.GET("/recent", incomeHandler.GetRecentIncomes)
			incomesGroup.GET("/stats", incomeHandler.GetIncomeStats)
			incomesGroup.POST("/process-recurring", incomeHandler.ProcessRecurringIncomes)
			incomesGroup.GET("/:id", incomeHandler.GetIncome)
			incomesGroup.PUT("/:id", incomeHandler.UpdateIncome)
			incomesGroup.DELETE("/:id", incomeHandler.DeleteIncome)
		}

		// Rutas de cuentas bancarias
		bankAccountsGroup := protectedGroup.Group("/bank-accounts")
		{
			// Rutas principales (con y sin trailing slash para evitar redirects)
			bankAccountsGroup.GET("/", bankAccountHandler.GetUserBankAccounts)
			bankAccountsGroup.GET("", bankAccountHandler.GetUserBankAccounts)
			bankAccountsGroup.POST("/", bankAccountHandler.CreateBankAccount)
			bankAccountsGroup.POST("", bankAccountHandler.CreateBankAccount)

			// Otras rutas
			bankAccountsGroup.GET("/summary", bankAccountHandler.GetBankAccountSummary)
			bankAccountsGroup.GET("/type/:type", bankAccountHandler.GetBankAccountsByType)
			bankAccountsGroup.GET("/:id", bankAccountHandler.GetBankAccount)
			bankAccountsGroup.PUT("/:id", bankAccountHandler.UpdateBankAccount)
			bankAccountsGroup.DELETE("/:id", bankAccountHandler.DeleteBankAccount)
			bankAccountsGroup.PATCH("/:id/active", bankAccountHandler.SetBankAccountActive)
			bankAccountsGroup.PATCH("/:id/balance", bankAccountHandler.UpdateBankAccountBalance)
		}

		// Rutas de patrones de notificación
		notificationPatternsGroup := protectedGroup.Group("/notification-patterns")
		{
			// Rutas principales (con y sin trailing slash para evitar redirects)
			notificationPatternsGroup.GET("/", bankNotificationPatternHandler.GetUserPatterns)
			notificationPatternsGroup.GET("", bankNotificationPatternHandler.GetUserPatterns)
			notificationPatternsGroup.POST("/", bankNotificationPatternHandler.CreatePattern)
			notificationPatternsGroup.POST("", bankNotificationPatternHandler.CreatePattern)

			// Otras rutas
			notificationPatternsGroup.GET("/statistics", bankNotificationPatternHandler.GetPatternStatistics)
			notificationPatternsGroup.POST("/process", bankNotificationPatternHandler.ProcessNotification)
			notificationPatternsGroup.POST("/process-sms", bankNotificationPatternHandler.ProcessSMSWithAI)
			notificationPatternsGroup.POST("/process-sms-batch", bankNotificationPatternHandler.ProcessSMSBatchWithAI)
			notificationPatternsGroup.POST("/analyze-sms-batch/jobs", bankNotificationPatternHandler.StartAnalyzeSMSBatchJob)
			notificationPatternsGroup.GET("/analyze-sms-batch/jobs/:jobId", bankNotificationPatternHandler.GetAnalyzeSMSBatchJobStatus)
			notificationPatternsGroup.POST("/analyze-sms-batch", bankNotificationPatternHandler.AnalyzeSMSBatch)
			notificationPatternsGroup.POST("/analyze-statement", bankNotificationPatternHandler.AnalyzeStatement)
			notificationPatternsGroup.GET("/:id", bankNotificationPatternHandler.GetPattern)
			notificationPatternsGroup.PUT("/:id", bankNotificationPatternHandler.UpdatePattern)
			notificationPatternsGroup.DELETE("/:id", bankNotificationPatternHandler.DeletePattern)
			notificationPatternsGroup.PATCH("/:id/status", bankNotificationPatternHandler.SetPatternStatus)

			// Rutas de patrones por cuenta bancaria (usando ruta alternativa)
			notificationPatternsGroup.GET("/bank-account/:bank_account_id", bankNotificationPatternHandler.GetBankAccountPatterns)
		}

		// Correo OAuth (Gmail)
		emailConnGroup := protectedGroup.Group("/email-connections")
		{
			emailConnGroup.GET("/gmail/authorize", emailConnectionHandler.GmailAuthorize)
			emailConnGroup.GET("", emailConnectionHandler.GetEmailConnectionStatus)
			emailConnGroup.DELETE("/gmail", emailConnectionHandler.GmailDisconnect)
			emailConnGroup.POST("/gmail/sync", emailConnectionHandler.GmailSync)
		}
	}

	// Rutas de webhooks (públicas - sin autenticación)
	{
		webhooksGroup := v1.Group("/webhooks")
		cfg := configs.Get()
		webhooksGroup.Use(WebhookAuthMiddleware(cfg.External.WebhookSecret))
		webhookRateLimiter := NewRateLimiter(10, time.Minute)
		webhooksGroup.Use(webhookRateLimiter.RateLimitMiddleware())

		// Webhook principal para notificaciones bancarias
		webhooksGroup.POST("/bank-notification", webhookHandler.ReceiveBankNotification)

		// Webhook específico para SMS
		webhooksGroup.POST("/sms", webhookHandler.ReceiveSMSNotification)

		// Procesamiento de notificaciones pendientes
		webhooksGroup.POST("/process-pending", webhookHandler.ProcessPendingNotifications)

		// Estadísticas de notificaciones
		webhooksGroup.GET("/stats", webhookHandler.GetNotificationStats)
	}
}

// setupGlobalMiddlewares configura middlewares globales
func setupGlobalMiddlewares(router *gin.Engine) {
	// Recovery optimizado según entorno
	if gin.Mode() == gin.ReleaseMode {
		// Para producción (Railway): recovery simple
		router.Use(SimpleRecoveryMiddleware())
	} else {
		// Para desarrollo: recovery detallado
		router.Use(RecoveryMiddleware())
	}

	// Headers de seguridad
	router.Use(SecurityHeadersMiddleware())

	// CORS se configura en app.go - no duplicar aquí
}

// setupAPIMiddlewares configura middlewares específicos de la API
func setupAPIMiddlewares(group *gin.RouterGroup) {
	// Logging unificado
	group.Use(LoggerMiddleware())

	// Manejo de errores centralizado
	group.Use(ErrorHandlerMiddleware())

	// Rate limiting: 100 requests por minuto por IP
	rateLimiter := NewRateLimiter(100, time.Minute)
	group.Use(rateLimiter.RateLimitMiddleware())

	// Validación de content-type
	group.Use(ValidateContentTypeMiddleware())

	// Límite de tamaño de request: 10MB
	group.Use(RequestSizeLimitMiddleware(10 * 1024 * 1024))

	// Timeout de request: 30 segundos
	group.Use(TimeoutMiddleware(30 * time.Second))

	// Detección de actividad sospechosa
	group.Use(SuspiciousActivityMiddleware())

	// Validador personalizado
	customValidator := NewCustomValidator()
	group.Use(ValidationMiddleware(customValidator))
}

// setupPublicRoutes configura rutas públicas que no requieren autenticación
func setupPublicRoutes(router *gin.Engine) {
	privacyHandler := NewPrivacyHandler()

	// Rutas de políticas y términos
	router.GET("/privacy", privacyHandler.GetPrivacyPolicy)
	router.GET("/privacy-policy", privacyHandler.GetPrivacyPolicy)
	router.GET("/terms", privacyHandler.GetTermsOfService)
	router.GET("/terms-of-service", privacyHandler.GetTermsOfService)

	// Ruta de eliminación de cuenta
	router.GET("/delete-account", privacyHandler.GetAccountDeletion)
	router.GET("/account-deletion", privacyHandler.GetAccountDeletion)
}
