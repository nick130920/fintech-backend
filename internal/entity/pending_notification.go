package entity

import (
	"time"

	"gorm.io/gorm"
)

type PendingNotificationStatus string

const (
	PendingNotificationStatusPending   PendingNotificationStatus = "pending"
	PendingNotificationStatusProcessed PendingNotificationStatus = "processed"
)

// PendingNotification guarda notificaciones que requieren validación manual.
type PendingNotification struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	UserID uint `json:"user_id" gorm:"not null;index"`

	RawMessage string `json:"raw_message" gorm:"type:text;not null"`
	Channel    string `json:"channel" gorm:"type:varchar(20);not null;default:'sms'"`
	Phone      string `json:"phone" gorm:"type:varchar(30)"`
	ReceivedAt time.Time `json:"received_at" gorm:"index"`

	Attempts  int                       `json:"attempts" gorm:"default:0"`
	LastError string                    `json:"last_error" gorm:"type:text"`
	Status    PendingNotificationStatus `json:"status" gorm:"type:varchar(20);default:'pending';index"`
}
