package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"github.com/nick130920/fintech-backend/pkg/auth"
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
	categoryRepo repo.CategoryRepo,
	jwtManager *auth.JWTManager,
) {
	// Configurar middlewares globales de seguridad
	setupGlobalMiddlewares(router)

	// Grupo principal de API v1
	v1 := router.Group("/api/v1")

	// Configurar middlewares específicos de API
	setupAPIMiddlewares(v1)

	// Inicializar handlers
	userHandler := NewUserHandler(userUC)
	accountHandler := NewAccountHandler(accountUC)
	transactionHandler := NewTransactionHandler(transactionUC)
	budgetHandler := NewBudgetHandler(budgetUC)
	expenseHandler := NewExpenseHandler(expenseUC)
	incomeHandler := NewIncomeHandler(incomeUC)
	bankAccountHandler := NewBankAccountHandler(bankAccountUC)
	bankNotificationPatternHandler := NewBankNotificationPatternHandler(bankNotificationPatternUC)
	categoryHandler := NewCategoryHandler(categoryRepo)

	// Middleware de autenticación
	authMiddleware := NewAuthMiddleware(jwtManager)

	// Rutas de autenticación (públicas)
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", userHandler.Register)
		authGroup.POST("/login", userHandler.Login)
		authGroup.POST("/refresh", userHandler.RefreshToken)

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
			transactionsGroup.POST("/", transactionHandler.CreateTransaction)
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
			categoriesGroup.POST("/", categoryHandler.CreateCategory)
			categoriesGroup.PUT("/:id", categoryHandler.UpdateCategory)
			categoriesGroup.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		// Rutas de gastos
		expensesGroup := protectedGroup.Group("/expenses")
		{
			expensesGroup.POST("/", expenseHandler.CreateExpense)
			expensesGroup.GET("/", expenseHandler.GetExpenses)
			expensesGroup.GET("/recent", expenseHandler.GetRecentExpenses)
			expensesGroup.GET("/by-category", expenseHandler.GetExpensesByCategory)
			expensesGroup.PUT("/:id", expenseHandler.UpdateExpense)
			expensesGroup.DELETE("/:id", expenseHandler.DeleteExpense)
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
			notificationPatternsGroup.GET("/:id", bankNotificationPatternHandler.GetPattern)
			notificationPatternsGroup.PUT("/:id", bankNotificationPatternHandler.UpdatePattern)
			notificationPatternsGroup.DELETE("/:id", bankNotificationPatternHandler.DeletePattern)
			notificationPatternsGroup.PATCH("/:id/status", bankNotificationPatternHandler.SetPatternStatus)

			// Rutas de patrones por cuenta bancaria (usando ruta alternativa)
			notificationPatternsGroup.GET("/bank-account/:bank_account_id", bankNotificationPatternHandler.GetBankAccountPatterns)
		}
	}
}

// setupGlobalMiddlewares configura middlewares globales
func setupGlobalMiddlewares(router *gin.Engine) {
	// Recovery mejorado (debe ir primero)
	router.Use(RecoveryMiddleware())

	// Headers de seguridad
	router.Use(SecurityHeadersMiddleware())

	// CORS (si es necesario)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

// setupAPIMiddlewares configura middlewares específicos de la API
func setupAPIMiddlewares(group *gin.RouterGroup) {
	// Logging avanzado con métricas
	group.Use(EnhancedLoggerMiddleware())

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
