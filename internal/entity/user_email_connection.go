package entity

import (
	"time"

	"gorm.io/gorm"
)

// UserEmailConnection vincula un usuario con una cuenta de correo (OAuth).
type UserEmailConnection struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID uint `json:"user_id" gorm:"not null;index:idx_user_email_provider,unique,priority:1"`

	// Provider ej. "gmail"
	Provider string `json:"provider" gorm:"size:32;not null;index:idx_user_email_provider,unique,priority:2"`

	EmailAddress string `json:"email_address" gorm:"size:255;not null"`

	RefreshTokenEnc string `json:"-" gorm:"type:text"`
	AccessTokenEnc  string `json:"-" gorm:"type:text"`
	AccessExpiresAt *time.Time `json:"-"`

	// LastHistoryID último historyId de Gmail para sync incremental (string por compatibilidad API).
	LastHistoryID string `json:"last_history_id" gorm:"size:64"`
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	RevokedAt     *time.Time `json:"revoked_at"`
}
