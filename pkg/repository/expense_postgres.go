package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
)

// ExpensePostgres implementa ExpenseRepo usando PostgreSQL
type ExpensePostgres struct {
	db *gorm.DB
}

// NewExpensePostgres crea una nueva instancia de ExpensePostgres
func NewExpensePostgres(db *gorm.DB) repo.ExpenseRepo {
	return &ExpensePostgres{db: db}
}

// Create crea un nuevo gasto
func (r *ExpensePostgres) Create(expense *entity.Expense) error {
	return r.db.Create(expense).Error
}

// GetByID obtiene un gasto por ID
func (r *ExpensePostgres) GetByID(id uint) (*entity.Expense, error) {
	var expense entity.Expense
	err := r.db.Preload("Category").First(&expense, id).Error
	if err != nil {
		return nil, err
	}
	return &expense, nil
}

// GetByUserID obtiene todos los gastos de un usuario
func (r *ExpensePostgres) GetByUserID(userID uint) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("user_id = ?", userID).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error
	return expenses, err
}

// GetByUserAndDateRange obtiene gastos de un usuario en un rango de fechas
func (r *ExpensePostgres) GetByUserAndDateRange(userID uint, startDate, endDate *time.Time) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	query := r.db.Preload("Category").Where("user_id = ?", userID)

	if startDate != nil {
		query = query.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("date <= ?", *endDate)
	}

	err := query.Order("date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

// GetByUserCategoryAndDate obtiene gastos de un usuario en una categoría y fecha específica
func (r *ExpensePostgres) GetByUserCategoryAndDate(userID, categoryID uint, date *time.Time) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	query := r.db.Preload("Category").
		Where("user_id = ? AND category_id = ?", userID, categoryID)

	if date != nil {
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
		query = query.Where("date BETWEEN ? AND ?", startOfDay, endOfDay)
	}

	err := query.Order("date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

// GetByUserAndCategory obtiene gastos de un usuario por categoría
func (r *ExpensePostgres) GetByUserAndCategory(userID, categoryID uint) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("user_id = ? AND category_id = ?", userID, categoryID).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error
	return expenses, err
}

// GetByBudgetID obtiene gastos asociados a un presupuesto específico
func (r *ExpensePostgres) GetByBudgetID(budgetID uint) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("budget_id = ?", budgetID).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error
	return expenses, err
}

// GetByAllocationID obtiene gastos asociados a una asignación de presupuesto específica
func (r *ExpensePostgres) GetByAllocationID(allocationID uint) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("allocation_id = ?", allocationID).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error
	return expenses, err
}

// Update actualiza un gasto
func (r *ExpensePostgres) Update(expense *entity.Expense) error {
	return r.db.Save(expense).Error
}

// Delete elimina un gasto (soft delete)
func (r *ExpensePostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Expense{}, id).Error
}

// GetExpensesByStatus obtiene gastos por estado
func (r *ExpensePostgres) GetExpensesByStatus(userID uint, status entity.ExpenseStatus) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("user_id = ? AND status = ?", userID, status).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error
	return expenses, err
}

// GetRecentExpenses obtiene los gastos más recientes de un usuario
func (r *ExpensePostgres) GetRecentExpenses(userID uint, limit int) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	err := r.db.Preload("Category").
		Where("user_id = ?", userID).
		Order("date DESC, created_at DESC").
		Limit(limit).
		Find(&expenses).Error
	return expenses, err
}

// GetExpensesSummaryByCategory obtiene un resumen de gastos agrupados por categoría
func (r *ExpensePostgres) GetExpensesSummaryByCategory(userID uint, startDate, endDate *time.Time) ([]ExpenseCategorySummary, error) {
	var summaries []ExpenseCategorySummary

	query := r.db.Table("expenses").
		Select("category_id, categories.name as category_name, categories.icon as category_icon, categories.color as category_color, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_amount").
		Joins("JOIN categories ON expenses.category_id = categories.id").
		Where("expenses.user_id = ? AND expenses.status IN (?)", userID, []string{"confirmed", "pending"}).
		Group("category_id, categories.name, categories.icon, categories.color").
		Order("total_amount DESC")

	if startDate != nil {
		query = query.Where("expenses.date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("expenses.date <= ?", *endDate)
	}

	err := query.Scan(&summaries).Error
	return summaries, err
}

// GetMonthlyExpensesSummary obtiene un resumen de gastos mensuales
func (r *ExpensePostgres) GetMonthlyExpensesSummary(userID uint, year int) ([]MonthlyExpenseSummary, error) {
	var summaries []MonthlyExpenseSummary

	err := r.db.Table("expenses").
		Select("EXTRACT(month FROM date) as month, EXTRACT(year FROM date) as year, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_amount").
		Where("user_id = ? AND EXTRACT(year FROM date) = ? AND status IN (?)", userID, year, []string{"confirmed", "pending"}).
		Group("EXTRACT(year FROM date), EXTRACT(month FROM date)").
		Order("year, month").
		Scan(&summaries).Error

	return summaries, err
}

// Search busca gastos por criterios múltiples
func (r *ExpensePostgres) Search(userID uint, params SearchParams) ([]*entity.Expense, int64, error) {
	var expenses []*entity.Expense
	var total int64

	// Construir query base
	query := r.db.Model(&entity.Expense{}).Where("user_id = ?", userID)

	// Aplicar filtros
	if params.CategoryID != nil {
		query = query.Where("category_id = ?", *params.CategoryID)
	}

	if params.StartDate != nil {
		query = query.Where("date >= ?", *params.StartDate)
	}

	if params.EndDate != nil {
		query = query.Where("date <= ?", *params.EndDate)
	}

	if params.MinAmount != nil {
		query = query.Where("amount >= ?", *params.MinAmount)
	}

	if params.MaxAmount != nil {
		query = query.Where("amount <= ?", *params.MaxAmount)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if params.SearchTerm != "" {
		searchPattern := "%" + params.SearchTerm + "%"
		query = query.Where("description ILIKE ? OR merchant ILIKE ? OR location ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	// Contar total
	countQuery := query
	err := countQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Aplicar paginación
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	// Ordenar y obtener resultados
	orderBy := "date DESC, created_at DESC"
	if params.OrderBy != "" {
		orderBy = params.OrderBy
	}

	err = query.Preload("Category").Order(orderBy).Find(&expenses).Error
	return expenses, total, err
}

// UpdateStatus actualiza el estado de un gasto
func (r *ExpensePostgres) UpdateStatus(id uint, status entity.ExpenseStatus) error {
	return r.db.Model(&entity.Expense{}).Where("id = ?", id).Update("status", status).Error
}

// BulkCreate crea múltiples gastos en una transacción
func (r *ExpensePostgres) BulkCreate(expenses []*entity.Expense) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&expenses).Error
	})
}

// GetTotalSpentByUser obtiene el total gastado por un usuario en un período
func (r *ExpensePostgres) GetTotalSpentByUser(userID uint, startDate, endDate *time.Time) (float64, error) {
	var total float64

	query := r.db.Model(&entity.Expense{}).
		Where("user_id = ? AND status IN (?)", userID, []string{"confirmed", "pending"}).
		Select("COALESCE(SUM(amount), 0)")

	if startDate != nil {
		query = query.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("date <= ?", *endDate)
	}

	err := query.Scan(&total).Error
	return total, err
}

// GetTotalSpentByCategory obtiene el total gastado por categoría
func (r *ExpensePostgres) GetTotalSpentByCategory(userID, categoryID uint, startDate, endDate *time.Time) (float64, error) {
	var total float64

	query := r.db.Model(&entity.Expense{}).
		Where("user_id = ? AND category_id = ? AND status IN (?)", userID, categoryID, []string{"confirmed", "pending"}).
		Select("COALESCE(SUM(amount), 0)")

	if startDate != nil {
		query = query.Where("date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("date <= ?", *endDate)
	}

	err := query.Scan(&total).Error
	return total, err
}

// Estructuras auxiliares para resúmenes

// ExpenseCategorySummary representa un resumen de gastos por categoría
type ExpenseCategorySummary struct {
	CategoryID    uint    `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	CategoryIcon  string  `json:"category_icon"`
	CategoryColor string  `json:"category_color"`
	Count         int64   `json:"count"`
	TotalAmount   float64 `json:"total_amount"`
}

// MonthlyExpenseSummary representa un resumen de gastos mensuales
type MonthlyExpenseSummary struct {
	Year        int     `json:"year"`
	Month       int     `json:"month"`
	Count       int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// SearchParams define los parámetros de búsqueda para gastos
type SearchParams struct {
	CategoryID *uint                 `json:"category_id"`
	StartDate  *time.Time            `json:"start_date"`
	EndDate    *time.Time            `json:"end_date"`
	MinAmount  *float64              `json:"min_amount"`
	MaxAmount  *float64              `json:"max_amount"`
	Status     *entity.ExpenseStatus `json:"status"`
	SearchTerm string                `json:"search_term"`
	Offset     int                   `json:"offset"`
	Limit      int                   `json:"limit"`
	OrderBy    string                `json:"order_by"`
}

// Métodos adicionales requeridos por la interfaz ExpenseRepo

// GetByUserIDWithFilter obtiene gastos filtrados (implementación básica)
func (r *ExpensePostgres) GetByUserIDWithFilter(userID uint, filter *entity.ExpenseFilter) ([]*entity.ExpenseSummary, error) {
	// Implementación básica - retornar error de no implementado por ahora
	return nil, nil
}

// GetPendingExpenses obtiene gastos pendientes
func (r *ExpensePostgres) GetPendingExpenses(userID uint) ([]*entity.Expense, error) {
	return r.GetExpensesByStatus(userID, entity.ExpenseStatusPending)
}

// ConfirmExpense confirma un gasto
func (r *ExpensePostgres) ConfirmExpense(id uint) error {
	return r.UpdateStatus(id, entity.ExpenseStatusConfirmed)
}

// CancelExpense cancela un gasto
func (r *ExpensePostgres) CancelExpense(id uint) error {
	return r.UpdateStatus(id, entity.ExpenseStatusCancelled)
}

// BatchConfirmExpenses confirma múltiples gastos
func (r *ExpensePostgres) BatchConfirmExpenses(expenseIDs []uint) error {
	return r.db.Model(&entity.Expense{}).
		Where("id IN ?", expenseIDs).
		Update("status", entity.ExpenseStatusConfirmed).Error
}

// GetTodayExpenses obtiene gastos de hoy
func (r *ExpensePostgres) GetTodayExpenses(userID uint) ([]*entity.Expense, error) {
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	return r.GetByUserAndDateRange(userID, &startOfDay, &endOfDay)
}

// GetWeekExpenses obtiene gastos de la semana
func (r *ExpensePostgres) GetWeekExpenses(userID uint) ([]*entity.Expense, error) {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))

	return r.GetByUserAndDateRange(userID, &weekStart, &now)
}

// GetMonthExpenses obtiene gastos del mes
func (r *ExpensePostgres) GetMonthExpenses(userID uint, year, month int) ([]*entity.Expense, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	return r.GetByUserAndDateRange(userID, &startOfMonth, &endOfMonth)
}

// GetByCategoryAndDateRange obtiene gastos por categoría y rango de fechas
func (r *ExpensePostgres) GetByCategoryAndDateRange(userID, categoryID uint, fromDate, toDate *time.Time) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	query := r.db.Preload("Category").
		Where("user_id = ? AND category_id = ?", userID, categoryID)

	if fromDate != nil {
		query = query.Where("date >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("date <= ?", *toDate)
	}

	err := query.Order("date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

// GetExpensesByCategories obtiene gastos de múltiples categorías
func (r *ExpensePostgres) GetExpensesByCategories(userID uint, categoryIDs []uint, fromDate, toDate *time.Time) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	query := r.db.Preload("Category").
		Where("user_id = ? AND category_id IN ?", userID, categoryIDs)

	if fromDate != nil {
		query = query.Where("date >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("date <= ?", *toDate)
	}

	err := query.Order("date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

// CalculateTotalByUser calcula el total gastado por usuario
func (r *ExpensePostgres) CalculateTotalByUser(userID uint, fromDate, toDate *time.Time) (float64, error) {
	return r.GetTotalSpentByUser(userID, fromDate, toDate)
}

// CalculateTotalByCategory calcula el total gastado por categoría
func (r *ExpensePostgres) CalculateTotalByCategory(userID, categoryID uint, fromDate, toDate *time.Time) (float64, error) {
	return r.GetTotalSpentByCategory(userID, categoryID, fromDate, toDate)
}

// CalculateTotalByAllocation calcula el total gastado por asignación
func (r *ExpensePostgres) CalculateTotalByAllocation(allocationID uint) (float64, error) {
	var total float64

	err := r.db.Model(&entity.Expense{}).
		Where("allocation_id = ? AND status IN (?)", allocationID, []string{"confirmed", "pending"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

// CalculateDailyAverage calcula el promedio diario de gastos
func (r *ExpensePostgres) CalculateDailyAverage(userID uint, fromDate, toDate *time.Time) (float64, error) {
	total, err := r.GetTotalSpentByUser(userID, fromDate, toDate)
	if err != nil {
		return 0, err
	}

	if fromDate == nil || toDate == nil {
		return total, nil
	}

	days := toDate.Sub(*fromDate).Hours() / 24
	if days <= 0 {
		return total, nil
	}

	return total / days, nil
}

// Métodos básicos para cumplir con la interfaz (implementaciones mínimas)

func (r *ExpensePostgres) GetExpensesBySource(userID uint, fromDate, toDate *time.Time) (map[entity.ExpenseSource]float64, error) {
	return make(map[entity.ExpenseSource]float64), nil
}

func (r *ExpensePostgres) GetTopMerchants(userID uint, limit int, fromDate, toDate *time.Time) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (r *ExpensePostgres) GetCategoryTotals(userID uint, fromDate, toDate *time.Time) (map[uint]float64, error) {
	return make(map[uint]float64), nil
}

func (r *ExpensePostgres) GetDailyTotals(userID uint, fromDate, toDate *time.Time) (map[string]float64, error) {
	return make(map[string]float64), nil
}

func (r *ExpensePostgres) CreateFromSMS(smsData map[string]interface{}) (*entity.Expense, error) {
	return nil, nil
}

func (r *ExpensePostgres) GetDuplicateCandidates(amount float64, description string, date time.Time, tolerance time.Duration) ([]*entity.Expense, error) {
	return []*entity.Expense{}, nil
}

func (r *ExpensePostgres) SearchByDescription(userID uint, searchTerm string, limit int) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	searchPattern := "%" + searchTerm + "%"

	err := r.db.Preload("Category").
		Where("user_id = ? AND description ILIKE ?", userID, searchPattern).
		Order("date DESC, created_at DESC").
		Limit(limit).
		Find(&expenses).Error

	return expenses, err
}

func (r *ExpensePostgres) GetByMerchant(userID uint, merchant string, fromDate, toDate *time.Time) ([]*entity.Expense, error) {
	var expenses []*entity.Expense
	query := r.db.Preload("Category").
		Where("user_id = ? AND merchant = ?", userID, merchant)

	if fromDate != nil {
		query = query.Where("date >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("date <= ?", *toDate)
	}

	err := query.Order("date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

func (r *ExpensePostgres) GetWithReceipt(userID uint) ([]*entity.Expense, error) {
	var expenses []*entity.Expense

	err := r.db.Preload("Category").
		Where("user_id = ? AND receipt_url IS NOT NULL AND receipt_url != ''", userID).
		Order("date DESC, created_at DESC").
		Find(&expenses).Error

	return expenses, err
}

func (r *ExpensePostgres) CleanupCancelledExpenses(olderThan time.Duration) error {
	cutoffDate := time.Now().Add(-olderThan)

	return r.db.Unscoped().Where("status = ? AND deleted_at < ?",
		entity.ExpenseStatusCancelled, cutoffDate).Delete(&entity.Expense{}).Error
}

func (r *ExpensePostgres) UpdateCurrency(fromCurrency, toCurrency string, exchangeRate float64) error {
	return r.db.Model(&entity.Expense{}).
		Where("currency = ?", fromCurrency).
		Updates(map[string]interface{}{
			"currency": toCurrency,
			"amount":   gorm.Expr("amount * ?", exchangeRate),
		}).Error
}
