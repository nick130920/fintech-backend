package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

type RevokedTokenPostgres struct {
	db *gorm.DB
}

func NewRevokedTokenPostgres(db *gorm.DB) repo.RevokedTokenRepo {
	return &RevokedTokenPostgres{db: db}
}

func (r *RevokedTokenPostgres) Revoke(token *entity.RevokedToken) error {
	return r.db.Create(token).Error
}

func (r *RevokedTokenPostgres) IsRevoked(jti string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.RevokedToken{}).
		Where("token_jti = ? AND expires_at > ?", jti, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RevokedTokenPostgres) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&entity.RevokedToken{}).Error
}
