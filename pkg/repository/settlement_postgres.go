package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// SettlementPostgres implementa repo.SettlementRepo
type SettlementPostgres struct {
	db *gorm.DB
}

// NewSettlementPostgres construye el repositorio de settlements
func NewSettlementPostgres(db *gorm.DB) repo.SettlementRepo {
	return &SettlementPostgres{db: db}
}

func (r *SettlementPostgres) Create(settlement *entity.Settlement) error {
	return r.db.Create(settlement).Error
}

func (r *SettlementPostgres) GetByID(id uint) (*entity.Settlement, error) {
	var settlement entity.Settlement
	err := r.db.
		Preload("FromMember").
		Preload("ToMember").
		First(&settlement, id).Error
	if err != nil {
		return nil, err
	}
	return &settlement, nil
}

func (r *SettlementPostgres) GetByTrip(tripID uint) ([]*entity.Settlement, error) {
	var settlements []*entity.Settlement
	err := r.db.
		Preload("FromMember").
		Preload("ToMember").
		Where("trip_id = ?", tripID).
		Order("paid_at DESC").
		Find(&settlements).Error
	return settlements, err
}

func (r *SettlementPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Settlement{}, id).Error
}
