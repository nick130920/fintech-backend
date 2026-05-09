package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TripPostgres implementa repo.TripRepo usando PostgreSQL
type TripPostgres struct {
	db *gorm.DB
}

// NewTripPostgres construye el repositorio de viajes
func NewTripPostgres(db *gorm.DB) repo.TripRepo {
	return &TripPostgres{db: db}
}

func (r *TripPostgres) Create(trip *entity.Trip) error {
	return r.db.Create(trip).Error
}

func (r *TripPostgres) GetByID(id uint) (*entity.Trip, error) {
	var trip entity.Trip
	if err := r.db.First(&trip, id).Error; err != nil {
		return nil, err
	}
	return &trip, nil
}

func (r *TripPostgres) GetByIDDeep(id uint) (*entity.Trip, error) {
	var trip entity.Trip
	err := r.db.
		Preload("Members.User").
		Preload("Allocations.Category").
		Preload("Itinerary").
		First(&trip, id).Error
	if err != nil {
		return nil, err
	}
	return &trip, nil
}

func (r *TripPostgres) GetByUser(userID uint, status string) ([]*entity.Trip, error) {
	var trips []*entity.Trip
	query := r.db.
		Joins("LEFT JOIN trip_members ON trip_members.trip_id = trips.id AND trip_members.deleted_at IS NULL").
		Where("trips.owner_user_id = ? OR trip_members.user_id = ?", userID, userID).
		Where("trips.deleted_at IS NULL").
		Group("trips.id").
		Order("trips.start_date DESC")

	if status != "" {
		query = query.Where("trips.status = ?", status)
	}

	err := query.Find(&trips).Error
	return trips, err
}

func (r *TripPostgres) Update(trip *entity.Trip) error {
	return r.db.Save(trip).Error
}

func (r *TripPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Trip{}, id).Error
}

func (r *TripPostgres) UpdateTotals(tripID uint, estimatedTotal, spentTotal float64) error {
	return r.db.Model(&entity.Trip{}).
		Where("id = ?", tripID).
		Updates(map[string]interface{}{
			"estimated_total": estimatedTotal,
			"spent_total":     spentTotal,
		}).Error
}

func (r *TripPostgres) GetActiveOverlappingDates(userID uint, from, to time.Time) ([]*entity.Trip, error) {
	var trips []*entity.Trip
	err := r.db.
		Joins("LEFT JOIN trip_members ON trip_members.trip_id = trips.id AND trip_members.deleted_at IS NULL").
		Where("trips.owner_user_id = ? OR trip_members.user_id = ?", userID, userID).
		Where("trips.status IN ?", []string{string(entity.TripStatusPlanning), string(entity.TripStatusActive)}).
		Where("trips.start_date <= ? AND trips.end_date >= ?", to, from).
		Group("trips.id").
		Find(&trips).Error
	return trips, err
}
