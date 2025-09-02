package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
)

// IncomePostgres implementa IncomeRepo usando PostgreSQL
type IncomePostgres struct {
	db *gorm.DB
}

// NewIncomePostgres crea una nueva instancia de IncomePostgres
func NewIncomePostgres(db *gorm.DB) repo.IncomeRepo {
	return &IncomePostgres{db: db}
}

// Create crea un nuevo ingreso
func (r *IncomePostgres) Create(income *entity.Income) error {
	return r.db.Create(income).Error
}

// GetByID obtiene un ingreso por su ID
func (r *IncomePostgres) GetByID(id uint) (*entity.Income, error) {
	var income entity.Income
	err := r.db.Preload("User").First(&income, id).Error
	if err != nil {
		return nil, err
	}
	return &income, nil
}

// Update actualiza un ingreso existente
func (r *IncomePostgres) Update(income *entity.Income) error {
	return r.db.Save(income).Error
}

// Delete elimina un ingreso (soft delete)
func (r *IncomePostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Income{}, id).Error
}

// GetByUserID obtiene ingresos por ID de usuario con paginación
func (r *IncomePostgres) GetByUserID(userID uint, limit, offset int) ([]*entity.Income, error) {
	var incomes []*entity.Income
	err := r.db.Where("user_id = ?", userID).
		Order("date DESC").
		Limit(limit).
		Offset(offset).
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetByUserAndDateRange obtiene ingresos por usuario en un rango de fechas
func (r *IncomePostgres) GetByUserAndDateRange(userID uint, startDate, endDate *time.Time) ([]*entity.Income, error) {
	var incomes []*entity.Income
	query := r.db.Where("user_id = ?", userID)

	if startDate != nil {
		query = query.Where("date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("date <= ?", *endDate)
	}

	err := query.Order("date DESC").Find(&incomes).Error
	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetByUserAndSource obtiene ingresos por usuario y fuente
func (r *IncomePostgres) GetByUserAndSource(userID uint, source entity.IncomeSource) ([]*entity.Income, error) {
	var incomes []*entity.Income
	err := r.db.Where("user_id = ? AND source = ?", userID, source).
		Order("date DESC").
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetRecurringIncomes obtiene todos los ingresos recurrentes de un usuario
func (r *IncomePostgres) GetRecurringIncomes(userID uint) ([]*entity.Income, error) {
	var incomes []*entity.Income
	err := r.db.Where("user_id = ? AND is_recurring = true", userID).
		Order("date DESC").
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetPendingRecurringIncomes obtiene ingresos recurrentes pendientes
func (r *IncomePostgres) GetPendingRecurringIncomes(userID uint) ([]*entity.Income, error) {
	var incomes []*entity.Income
	now := time.Now()

	err := r.db.Where("user_id = ? AND is_recurring = true AND next_date <= ?", userID, now).
		Where("(end_date IS NULL OR end_date > ?) AND (recurring_until IS NULL OR recurring_until > ?)", now, now).
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetTotalIncomeByUser obtiene el total de ingresos de un usuario en un rango de fechas
func (r *IncomePostgres) GetTotalIncomeByUser(userID uint, startDate, endDate *time.Time) (float64, error) {
	var total float64
	query := r.db.Model(&entity.Income{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("user_id = ?", userID)

	if startDate != nil {
		query = query.Where("date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("date <= ?", *endDate)
	}

	err := query.Row().Scan(&total)
	return total, err
}

// GetIncomeByMonth obtiene ingresos de un mes específico
func (r *IncomePostgres) GetIncomeByMonth(userID uint, year, month int) ([]*entity.Income, error) {
	var incomes []*entity.Income

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	err := r.db.Where("user_id = ? AND date >= ? AND date <= ?", userID, startDate, endDate).
		Order("date DESC").
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetIncomeByYear obtiene ingresos de un año específico
func (r *IncomePostgres) GetIncomeByYear(userID uint, year int) ([]*entity.Income, error) {
	var incomes []*entity.Income

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	err := r.db.Where("user_id = ? AND date >= ? AND date <= ?", userID, startDate, endDate).
		Order("date DESC").
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetDueRecurringIncomes obtiene ingresos recurrentes que deben procesarse
func (r *IncomePostgres) GetDueRecurringIncomes(date time.Time) ([]*entity.Income, error) {
	var incomes []*entity.Income

	err := r.db.Where("is_recurring = true AND next_date <= ?", date).
		Where("(end_date IS NULL OR end_date > ?) AND (recurring_until IS NULL OR recurring_until > ?)", date, date).
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// UpdateNextRecurringDate actualiza la próxima fecha de un ingreso recurrente
func (r *IncomePostgres) UpdateNextRecurringDate(id uint, nextDate time.Time) error {
	return r.db.Model(&entity.Income{}).
		Where("id = ?", id).
		Update("next_date", nextDate).Error
}

// Search busca ingresos por descripción o notas
func (r *IncomePostgres) Search(userID uint, query string, limit, offset int) ([]*entity.Income, error) {
	var incomes []*entity.Income
	searchQuery := fmt.Sprintf("%%%s%%", query)

	err := r.db.Where("user_id = ? AND (description ILIKE ? OR notes ILIKE ?)", userID, searchQuery, searchQuery).
		Order("date DESC").
		Limit(limit).
		Offset(offset).
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}

// GetRecentIncomes obtiene los ingresos más recientes
func (r *IncomePostgres) GetRecentIncomes(userID uint, limit int) ([]*entity.Income, error) {
	var incomes []*entity.Income
	err := r.db.Where("user_id = ?", userID).
		Order("date DESC, created_at DESC").
		Limit(limit).
		Find(&incomes).Error

	if err != nil {
		return nil, err
	}
	return incomes, nil
}
