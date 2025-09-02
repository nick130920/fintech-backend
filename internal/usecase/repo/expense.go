package repo

import (
	"time"

	"github.com/nick130920/proyecto-fintech/internal/entity"
)

// ExpenseRepo define la interfaz para operaciones de gasto en la base de datos
type ExpenseRepo interface {
	// Operaciones básicas CRUD
	Create(expense *entity.Expense) error
	GetByID(id uint) (*entity.Expense, error)
	GetByUserID(userID uint) ([]*entity.Expense, error)
	Update(expense *entity.Expense) error
	Delete(id uint) error

	// Operaciones con filtros avanzados
	GetByUserIDWithFilter(userID uint, filter *entity.ExpenseFilter) ([]*entity.ExpenseSummary, error)
	GetByUserAndDateRange(userID uint, fromDate, toDate *time.Time) ([]*entity.Expense, error)
	GetByUserCategoryAndDate(userID, categoryID uint, date *time.Time) ([]*entity.Expense, error)
	GetByBudgetID(budgetID uint) ([]*entity.Expense, error)
	GetByAllocationID(allocationID uint) ([]*entity.Expense, error)

	// Operaciones de estado
	GetPendingExpenses(userID uint) ([]*entity.Expense, error)
	ConfirmExpense(id uint) error
	CancelExpense(id uint) error
	BatchConfirmExpenses(expenseIDs []uint) error

	// Búsquedas específicas por tiempo
	GetTodayExpenses(userID uint) ([]*entity.Expense, error)
	GetWeekExpenses(userID uint) ([]*entity.Expense, error)
	GetMonthExpenses(userID uint, year, month int) ([]*entity.Expense, error)
	GetRecentExpenses(userID uint, limit int) ([]*entity.Expense, error)

	// Búsquedas por categoría
	GetByCategoryAndDateRange(userID, categoryID uint, fromDate, toDate *time.Time) ([]*entity.Expense, error)
	GetExpensesByCategories(userID uint, categoryIDs []uint, fromDate, toDate *time.Time) ([]*entity.Expense, error)

	// Operaciones de cálculo
	CalculateTotalByUser(userID uint, fromDate, toDate *time.Time) (float64, error)
	CalculateTotalByCategory(userID, categoryID uint, fromDate, toDate *time.Time) (float64, error)
	CalculateTotalByAllocation(allocationID uint) (float64, error)
	CalculateDailyAverage(userID uint, fromDate, toDate *time.Time) (float64, error)

	// Estadísticas y reportes
	GetExpensesBySource(userID uint, fromDate, toDate *time.Time) (map[entity.ExpenseSource]float64, error)
	GetTopMerchants(userID uint, limit int, fromDate, toDate *time.Time) ([]map[string]interface{}, error)
	GetCategoryTotals(userID uint, fromDate, toDate *time.Time) (map[uint]float64, error)
	GetDailyTotals(userID uint, fromDate, toDate *time.Time) (map[string]float64, error)

	// Operaciones para procesamiento automático
	CreateFromSMS(smsData map[string]interface{}) (*entity.Expense, error)
	GetDuplicateCandidates(amount float64, description string, date time.Time, tolerance time.Duration) ([]*entity.Expense, error)

	// Búsquedas avanzadas
	SearchByDescription(userID uint, searchTerm string, limit int) ([]*entity.Expense, error)
	GetByMerchant(userID uint, merchant string, fromDate, toDate *time.Time) ([]*entity.Expense, error)
	GetWithReceipt(userID uint) ([]*entity.Expense, error)

	// Operaciones de mantenimiento
	CleanupCancelledExpenses(olderThan time.Duration) error
	UpdateCurrency(fromCurrency, toCurrency string, exchangeRate float64) error
}
