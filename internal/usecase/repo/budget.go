package repo

import (
	"github.com/nick130920/fintech-backend/internal/entity"
)

// BudgetRepo define la interfaz para operaciones de presupuesto en la base de datos
type BudgetRepo interface {
	// Operaciones básicas CRUD para Budget
	Create(budget *entity.Budget) error
	GetByID(id uint) (*entity.Budget, error)
	GetByIDWithAllocations(id uint) (*entity.Budget, error)
	GetByUserID(userID uint) ([]*entity.Budget, error)
	GetByUserAndMonth(userID uint, year, month int) (*entity.Budget, error)
	Update(budget *entity.Budget) error
	Delete(id uint) error

	// Operaciones específicas de Budget
	GetCurrentBudget(userID uint) (*entity.Budget, error)
	GetActiveBudgets(userID uint) ([]*entity.Budget, error)
	SetActive(id uint, active bool) error

	// Operaciones para BudgetAllocation
	CreateAllocation(allocation *entity.BudgetAllocation) error
	GetAllocationByID(id uint) (*entity.BudgetAllocation, error)
	GetAllocationsByBudgetID(budgetID uint) ([]*entity.BudgetAllocation, error)
	GetAllocationByBudgetAndCategory(budgetID, categoryID uint) (*entity.BudgetAllocation, error)
	UpdateAllocation(allocation *entity.BudgetAllocation) error
	DeleteAllocation(id uint) error

	// Operaciones de cálculo y estadísticas
	UpdateBudgetSpentAmount(budgetID uint) error
	UpdateAllocationSpentAmount(allocationID uint) error
	GetBudgetSummary(userID uint, year, month int) (*entity.Budget, error)

	// Operaciones para rollover y límites diarios
	GetAllocationsNeedingDailyUpdate(userID uint) ([]*entity.BudgetAllocation, error)
	BatchUpdateAllocations(allocations []*entity.BudgetAllocation) error
}
