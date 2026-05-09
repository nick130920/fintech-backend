package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TripBudgetAllocationPostgres implementa repo.TripBudgetAllocationRepo
type TripBudgetAllocationPostgres struct {
	db *gorm.DB
}

// NewTripBudgetAllocationPostgres construye el repositorio de asignaciones de viaje
func NewTripBudgetAllocationPostgres(db *gorm.DB) repo.TripBudgetAllocationRepo {
	return &TripBudgetAllocationPostgres{db: db}
}

func (r *TripBudgetAllocationPostgres) Create(allocation *entity.TripBudgetAllocation) error {
	return r.db.Create(allocation).Error
}

func (r *TripBudgetAllocationPostgres) Upsert(allocation *entity.TripBudgetAllocation) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "trip_id"}, {Name: "category_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"estimated_amount", "currency", "notes", "updated_at",
		}),
	}).Create(allocation).Error
}

func (r *TripBudgetAllocationPostgres) GetByID(id uint) (*entity.TripBudgetAllocation, error) {
	var allocation entity.TripBudgetAllocation
	err := r.db.Preload("Category").First(&allocation, id).Error
	if err != nil {
		return nil, err
	}
	return &allocation, nil
}

func (r *TripBudgetAllocationPostgres) GetByTrip(tripID uint) ([]*entity.TripBudgetAllocation, error) {
	var allocations []*entity.TripBudgetAllocation
	err := r.db.Preload("Category").
		Where("trip_id = ?", tripID).
		Order("category_id ASC").
		Find(&allocations).Error
	return allocations, err
}

func (r *TripBudgetAllocationPostgres) GetByTripAndCategory(tripID, categoryID uint) (*entity.TripBudgetAllocation, error) {
	var allocation entity.TripBudgetAllocation
	err := r.db.Preload("Category").
		Where("trip_id = ? AND category_id = ?", tripID, categoryID).
		First(&allocation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &allocation, nil
}

func (r *TripBudgetAllocationPostgres) Update(allocation *entity.TripBudgetAllocation) error {
	return r.db.Save(allocation).Error
}

func (r *TripBudgetAllocationPostgres) UpdateSpent(allocationID uint, spent float64) error {
	return r.db.Model(&entity.TripBudgetAllocation{}).
		Where("id = ?", allocationID).
		Update("spent_amount", spent).Error
}

func (r *TripBudgetAllocationPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.TripBudgetAllocation{}, id).Error
}
