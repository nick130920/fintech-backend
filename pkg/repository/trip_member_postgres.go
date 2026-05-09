package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TripMemberPostgres implementa repo.TripMemberRepo
type TripMemberPostgres struct {
	db *gorm.DB
}

// NewTripMemberPostgres construye el repositorio de miembros de viaje
func NewTripMemberPostgres(db *gorm.DB) repo.TripMemberRepo {
	return &TripMemberPostgres{db: db}
}

func (r *TripMemberPostgres) Create(member *entity.TripMember) error {
	return r.db.Create(member).Error
}

func (r *TripMemberPostgres) GetByID(id uint) (*entity.TripMember, error) {
	var member entity.TripMember
	if err := r.db.Preload("User").First(&member, id).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TripMemberPostgres) GetByTrip(tripID uint) ([]*entity.TripMember, error) {
	var members []*entity.TripMember
	err := r.db.
		Preload("User").
		Where("trip_id = ?", tripID).
		Order("role ASC, joined_at ASC").
		Find(&members).Error
	return members, err
}

func (r *TripMemberPostgres) GetByTripAndUser(tripID, userID uint) (*entity.TripMember, error) {
	var member entity.TripMember
	err := r.db.
		Preload("User").
		Where("trip_id = ? AND user_id = ?", tripID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TripMemberPostgres) GetByTripAndID(tripID, memberID uint) (*entity.TripMember, error) {
	var member entity.TripMember
	err := r.db.
		Preload("User").
		Where("trip_id = ? AND id = ?", tripID, memberID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *TripMemberPostgres) Update(member *entity.TripMember) error {
	return r.db.Save(member).Error
}

func (r *TripMemberPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.TripMember{}, id).Error
}

func (r *TripMemberPostgres) HasPendingSplits(memberID uint) (bool, error) {
	var count int64
	err := r.db.Model(&entity.ExpenseSplit{}).
		Where("member_id = ? AND is_paid = ?", memberID, false).
		Count(&count).Error
	return count > 0, err
}
