package repository

import (
	"errors"
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"gorm.io/gorm"
)

type userEmailConnectionPostgres struct {
	db *gorm.DB
}

// NewUserEmailConnectionPostgres constructor.
func NewUserEmailConnectionPostgres(db *gorm.DB) repo.UserEmailConnectionRepo {
	return &userEmailConnectionPostgres{db: db}
}

func (r *userEmailConnectionPostgres) CreateOrUpdate(conn *entity.UserEmailConnection) error {
	var existing entity.UserEmailConnection
	err := r.db.Unscoped().Where("user_id = ? AND provider = ?", conn.UserID, conn.Provider).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(conn).Error
	}
	if err != nil {
		return err
	}
	conn.ID = existing.ID
	conn.CreatedAt = existing.CreatedAt
	return r.db.Unscoped().Save(conn).Error
}

func (r *userEmailConnectionPostgres) GetByUserAndProvider(userID uint, provider string) (*entity.UserEmailConnection, error) {
	var c entity.UserEmailConnection
	err := r.db.Where("user_id = ? AND provider = ? AND revoked_at IS NULL", userID, provider).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *userEmailConnectionPostgres) SoftRevoke(userID uint, provider string) error {
	now := time.Now().UTC()
	return r.db.Model(&entity.UserEmailConnection{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Updates(map[string]interface{}{
			"revoked_at":        now,
			"refresh_token_enc": "",
			"access_token_enc":  "",
			"updated_at":        now,
		}).Error
}

func (r *userEmailConnectionPostgres) ListActiveByProvider(provider string) ([]entity.UserEmailConnection, error) {
	var list []entity.UserEmailConnection
	err := r.db.Where("provider = ? AND revoked_at IS NULL AND deleted_at IS NULL AND refresh_token_enc <> ''", provider).
		Find(&list).Error
	return list, err
}

type processedEmailMessagePostgres struct {
	db *gorm.DB
}

// NewProcessedEmailMessagePostgres constructor.
func NewProcessedEmailMessagePostgres(db *gorm.DB) repo.ProcessedEmailMessageRepo {
	return &processedEmailMessagePostgres{db: db}
}

func (r *processedEmailMessagePostgres) Exists(userID uint, provider, messageID string) (bool, error) {
	var count int64
	err := r.db.Model(&entity.ProcessedEmailMessage{}).
		Where("user_id = ? AND provider = ? AND provider_message_id = ?", userID, provider, messageID).
		Count(&count).Error
	return count > 0, err
}

func (r *processedEmailMessagePostgres) Create(m *entity.ProcessedEmailMessage) error {
	return r.db.Create(m).Error
}
