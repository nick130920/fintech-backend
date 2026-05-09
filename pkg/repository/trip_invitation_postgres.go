package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TripInvitationPostgres implementa repo.TripInvitationRepo
type TripInvitationPostgres struct {
	db *gorm.DB
}

// NewTripInvitationPostgres construye el repositorio de invitaciones
func NewTripInvitationPostgres(db *gorm.DB) repo.TripInvitationRepo {
	return &TripInvitationPostgres{db: db}
}

func (r *TripInvitationPostgres) Create(invitation *entity.TripInvitation) error {
	return r.db.Create(invitation).Error
}

func (r *TripInvitationPostgres) GetByToken(token string) (*entity.TripInvitation, error) {
	var inv entity.TripInvitation
	err := r.db.Preload("Trip").Where("token = ?", token).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *TripInvitationPostgres) GetByTrip(tripID uint) ([]*entity.TripInvitation, error) {
	var invitations []*entity.TripInvitation
	err := r.db.
		Where("trip_id = ?", tripID).
		Order("created_at DESC").
		Find(&invitations).Error
	return invitations, err
}

func (r *TripInvitationPostgres) Update(invitation *entity.TripInvitation) error {
	return r.db.Save(invitation).Error
}

func (r *TripInvitationPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.TripInvitation{}, id).Error
}
