package repo

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// IncomeRepo define la interfaz para el repositorio de ingresos
type IncomeRepo interface {
	// CRUD básico
	Create(income *entity.Income) error
	GetByID(id uint) (*entity.Income, error)
	Update(income *entity.Income) error
	Delete(id uint) error

	// Consultas específicas
	GetByUserID(userID uint, limit, offset int) ([]*entity.Income, error)
	GetByUserAndDateRange(userID uint, startDate, endDate *time.Time) ([]*entity.Income, error)
	GetByUserAndSource(userID uint, source entity.IncomeSource) ([]*entity.Income, error)
	GetRecurringIncomes(userID uint) ([]*entity.Income, error)
	GetPendingRecurringIncomes(userID uint) ([]*entity.Income, error)

	// Estadísticas y resúmenes
	GetTotalIncomeByUser(userID uint, startDate, endDate *time.Time) (float64, error)
	GetIncomeByMonth(userID uint, year, month int) ([]*entity.Income, error)
	GetIncomeByYear(userID uint, year int) ([]*entity.Income, error)

	// Ingresos recurrentes
	GetDueRecurringIncomes(date time.Time) ([]*entity.Income, error)
	UpdateNextRecurringDate(id uint, nextDate time.Time) error

	// Buscar y filtrar
	Search(userID uint, query string, limit, offset int) ([]*entity.Income, error)
	GetRecentIncomes(userID uint, limit int) ([]*entity.Income, error)
}
