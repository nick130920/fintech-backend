package entity

import "time"

// RevokedToken almacena los refresh tokens que han sido invalidados por logout.
// Al intentar usar un refresh token revocado, el servidor devolverá 401.
// Se limpian automáticamente los registros cuyo ExpiresAt ya pasó.
type RevokedToken struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	TokenJTI  string    `gorm:"uniqueIndex;not null"` // JWT ID (jti) del refresh token
	UserID    uint      `gorm:"not null;index"`
	RevokedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"` // Mismo TTL que el refresh token (30 días)
}
