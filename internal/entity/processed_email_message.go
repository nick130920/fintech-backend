package entity

import (
	"time"
)

// ProcessedEmailMessage evita reprocesar el mismo mensaje de proveedor.
type ProcessedEmailMessage struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`

	UserID uint `json:"user_id" gorm:"not null;uniqueIndex:idx_proc_email_dedupe,priority:1"`

	Provider string `json:"provider" gorm:"size:32;not null;uniqueIndex:idx_proc_email_dedupe,priority:2"`

	ProviderMessageID string `json:"provider_message_id" gorm:"size:128;not null;uniqueIndex:idx_proc_email_dedupe,priority:3"`
}
