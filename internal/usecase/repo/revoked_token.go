package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// RevokedTokenRepo define operaciones sobre la blacklist de refresh tokens.
type RevokedTokenRepo interface {
	// Revoke agrega un token a la blacklist.
	Revoke(token *entity.RevokedToken) error
	// IsRevoked devuelve true si el jti está en la blacklist y no ha expirado.
	IsRevoked(jti string) (bool, error)
	// DeleteExpired elimina registros cuyo ExpiresAt ya pasó (llamado periódicamente).
	DeleteExpired() error
}
