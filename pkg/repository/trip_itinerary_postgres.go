package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TripItineraryPostgres implementa repo.TripItineraryRepo
type TripItineraryPostgres struct {
	db *gorm.DB
}

// NewTripItineraryPostgres construye el repositorio de items del itinerario
func NewTripItineraryPostgres(db *gorm.DB) repo.TripItineraryRepo {
	return &TripItineraryPostgres{db: db}
}

func (r *TripItineraryPostgres) Create(item *entity.TripItineraryItem) error {
	return r.db.Create(item).Error
}

func (r *TripItineraryPostgres) GetByID(id uint) (*entity.TripItineraryItem, error) {
	var item entity.TripItineraryItem
	if err := r.db.Preload("Expense").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TripItineraryPostgres) GetByTrip(tripID uint) ([]*entity.TripItineraryItem, error) {
	var items []*entity.TripItineraryItem
	err := r.db.
		Preload("Expense").
		Where("trip_id = ?", tripID).
		Order("day ASC, time ASC, id ASC").
		Find(&items).Error
	return items, err
}

func (r *TripItineraryPostgres) Update(item *entity.TripItineraryItem) error {
	return r.db.Save(item).Error
}

func (r *TripItineraryPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.TripItineraryItem{}, id).Error
}

func (r *TripItineraryPostgres) LinkExpense(itemID, expenseID uint) error {
	return r.db.Model(&entity.TripItineraryItem{}).
		Where("id = ?", itemID).
		Update("expense_id", expenseID).Error
}
