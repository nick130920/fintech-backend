package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// BudgetPostgres implementa BudgetRepo usando PostgreSQL
type BudgetPostgres struct {
	db *gorm.DB
}

// NewBudgetPostgres crea una nueva instancia de BudgetPostgres
func NewBudgetPostgres(db *gorm.DB) repo.BudgetRepo {
	return &BudgetPostgres{db: db}
}

// Create crea un nuevo presupuesto
func (r *BudgetPostgres) Create(budget *entity.Budget) error {
	return r.db.Create(budget).Error
}

// GetByID obtiene un presupuesto por ID
func (r *BudgetPostgres) GetByID(id uint) (*entity.Budget, error) {
	var budget entity.Budget
	err := r.db.First(&budget, id).Error
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// GetByIDWithAllocations obtiene un presupuesto por ID con sus asignaciones
func (r *BudgetPostgres) GetByIDWithAllocations(id uint) (*entity.Budget, error) {
	var budget entity.Budget
	err := r.db.Preload("Allocations.Category").First(&budget, id).Error
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// GetByUserID obtiene todos los presupuestos de un usuario
func (r *BudgetPostgres) GetByUserID(userID uint) ([]*entity.Budget, error) {
	var budgets []*entity.Budget
	err := r.db.Where("user_id = ?", userID).
		Order("year DESC, month DESC").
		Find(&budgets).Error
	return budgets, err
}

// GetByUserAndMonth obtiene el presupuesto de un usuario para un mes específico
func (r *BudgetPostgres) GetByUserAndMonth(userID uint, year, month int) (*entity.Budget, error) {
	var budget entity.Budget
	err := r.db.Preload("Allocations.Category").
		Where("user_id = ? AND year = ? AND month = ?", userID, year, month).
		First(&budget).Error
	if err != nil {
		return nil, err
	}
	return &budget, nil
}

// Update actualiza un presupuesto
func (r *BudgetPostgres) Update(budget *entity.Budget) error {
	return r.db.Save(budget).Error
}

// Delete elimina un presupuesto (soft delete)
func (r *BudgetPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Budget{}, id).Error
}

// GetCurrentBudget obtiene el presupuesto del mes actual
func (r *BudgetPostgres) GetCurrentBudget(userID uint) (*entity.Budget, error) {
	now := time.Now()
	return r.GetByUserAndMonth(userID, now.Year(), int(now.Month()))
}

// GetActiveBudgets obtiene todos los presupuestos activos de un usuario
func (r *BudgetPostgres) GetActiveBudgets(userID uint) ([]*entity.Budget, error) {
	var budgets []*entity.Budget
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("year DESC, month DESC").
		Find(&budgets).Error
	return budgets, err
}

// SetActive establece el estado activo de un presupuesto
func (r *BudgetPostgres) SetActive(id uint, active bool) error {
	return r.db.Model(&entity.Budget{}).Where("id = ?", id).Update("is_active", active).Error
}

// CreateAllocation crea una nueva asignación de presupuesto
func (r *BudgetPostgres) CreateAllocation(allocation *entity.BudgetAllocation) error {
	return r.db.Create(allocation).Error
}

// GetAllocationByID obtiene una asignación por ID
func (r *BudgetPostgres) GetAllocationByID(id uint) (*entity.BudgetAllocation, error) {
	var allocation entity.BudgetAllocation
	err := r.db.Preload("Category").First(&allocation, id).Error
	if err != nil {
		return nil, err
	}
	return &allocation, nil
}

// GetAllocationsByBudgetID obtiene todas las asignaciones de un presupuesto
func (r *BudgetPostgres) GetAllocationsByBudgetID(budgetID uint) ([]*entity.BudgetAllocation, error) {
	var allocations []*entity.BudgetAllocation
	err := r.db.Preload("Category").
		Where("budget_id = ?", budgetID).
		Order("category_id").
		Find(&allocations).Error
	return allocations, err
}

// GetAllocationByBudgetAndCategory obtiene una asignación específica por presupuesto y categoría
func (r *BudgetPostgres) GetAllocationByBudgetAndCategory(budgetID, categoryID uint) (*entity.BudgetAllocation, error) {
	var allocation entity.BudgetAllocation
	err := r.db.Preload("Category").
		Where("budget_id = ? AND category_id = ?", budgetID, categoryID).
		First(&allocation).Error
	if err != nil {
		return nil, err
	}
	return &allocation, nil
}

// UpdateAllocation actualiza una asignación de presupuesto
func (r *BudgetPostgres) UpdateAllocation(allocation *entity.BudgetAllocation) error {
	return r.db.Save(allocation).Error
}

// DeleteAllocation elimina una asignación (soft delete)
func (r *BudgetPostgres) DeleteAllocation(id uint) error {
	return r.db.Delete(&entity.BudgetAllocation{}, id).Error
}

// UpdateBudgetSpentAmount actualiza el monto gastado de un presupuesto basado en sus gastos
func (r *BudgetPostgres) UpdateBudgetSpentAmount(budgetID uint) error {
	// Crear la subconsulta correctamente con GORM
	var totalSpent float64

	// Primero obtenemos el total gastado
	err := r.db.Model(&entity.Expense{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("budget_id = ? AND status IN ?", budgetID, []string{"confirmed", "pending"}).
		Row().Scan(&totalSpent)

	if err != nil {
		return err
	}

	// Luego actualizamos el presupuesto con el valor calculado
	return r.db.Model(&entity.Budget{}).
		Where("id = ?", budgetID).
		Updates(map[string]interface{}{
			"spent_amount":     totalSpent,
			"remaining_amount": gorm.Expr("total_amount - ?", totalSpent),
		}).Error
}

// UpdateAllocationSpentAmount actualiza el monto gastado de una asignación basado en sus gastos
func (r *BudgetPostgres) UpdateAllocationSpentAmount(allocationID uint) error {
	// Crear la subconsulta correctamente con GORM
	var totalSpent float64

	// Primero obtenemos el total gastado
	err := r.db.Model(&entity.Expense{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("allocation_id = ? AND status IN ?", allocationID, []string{"confirmed", "pending"}).
		Row().Scan(&totalSpent)

	if err != nil {
		return err
	}

	// Luego actualizamos la asignación con el valor calculado
	return r.db.Model(&entity.BudgetAllocation{}).
		Where("id = ?", allocationID).
		Updates(map[string]interface{}{
			"spent_amount":     totalSpent,
			"remaining_amount": gorm.Expr("allocated_amount - ?", totalSpent),
		}).Error
}

// GetBudgetSummary obtiene un resumen del presupuesto con estadísticas
func (r *BudgetPostgres) GetBudgetSummary(userID uint, year, month int) (*entity.Budget, error) {
	var budget entity.Budget

	err := r.db.Preload("Allocations", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Category").Order("category_id")
	}).Where("user_id = ? AND year = ? AND month = ?", userID, year, month).
		First(&budget).Error

	if err != nil {
		return nil, err
	}

	// Actualizar montos gastados
	r.UpdateBudgetSpentAmount(budget.ID)
	for _, allocation := range budget.Allocations {
		r.UpdateAllocationSpentAmount(allocation.ID)
	}

	// Recargar con datos actualizados
	err = r.db.Preload("Allocations", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Category").Order("category_id")
	}).First(&budget, budget.ID).Error

	return &budget, err
}

// GetAllocationsNeedingDailyUpdate obtiene asignaciones que necesitan actualización de límite diario
func (r *BudgetPostgres) GetAllocationsNeedingDailyUpdate(userID uint) ([]*entity.BudgetAllocation, error) {
	var allocations []*entity.BudgetAllocation

	// Obtener asignaciones del presupuesto actual que no han sido actualizadas hoy
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err := r.db.Preload("Category").
		Joins("JOIN budgets ON budget_allocations.budget_id = budgets.id").
		Where("budgets.user_id = ? AND budgets.year = ? AND budgets.month = ? AND budgets.is_active = ?",
			userID, now.Year(), int(now.Month()), true).
		Where("budget_allocations.last_calculated_at < ? OR budget_allocations.last_calculated_at IS NULL", today).
		Find(&allocations).Error

	return allocations, err
}

// BatchUpdateAllocations actualiza múltiples asignaciones en una transacción
func (r *BudgetPostgres) BatchUpdateAllocations(allocations []*entity.BudgetAllocation) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, allocation := range allocations {
			if err := tx.Save(allocation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
