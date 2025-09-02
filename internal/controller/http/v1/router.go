package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/nick130920/proyecto-fintech/internal/usecase"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
	"github.com/nick130920/proyecto-fintech/pkg/auth"
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
	categoryRepo repo.CategoryRepo,
	jwtManager *auth.JWTManager,
) {
	// Grupo principal de API v1
	v1 := router.Group("/api/v1")

	// Inicializar handlers
	userHandler := NewUserHandler(userUC)
	accountHandler := NewAccountHandler(accountUC)
	transactionHandler := NewTransactionHandler(transactionUC)
	budgetHandler := NewBudgetHandler(budgetUC)
	expenseHandler := NewExpenseHandler(expenseUC)
	incomeHandler := NewIncomeHandler(incomeUC)
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
	}
}
