package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// UserEmailConnectionRepo persistencia de conexiones correo OAuth.
type UserEmailConnectionRepo interface {
	CreateOrUpdate(conn *entity.UserEmailConnection) error
	GetByUserAndProvider(userID uint, provider string) (*entity.UserEmailConnection, error)
	SoftRevoke(userID uint, provider string) error
	ListActiveByProvider(provider string) ([]entity.UserEmailConnection, error)
}

// ProcessedEmailMessageRepo deduplicación de mensajes procesados.
type ProcessedEmailMessageRepo interface {
	Exists(userID uint, provider, messageID string) (bool, error)
	Create(m *entity.ProcessedEmailMessage) error
}
