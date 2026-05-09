package entity

import (
	"time"

	"gorm.io/gorm"
)

// TripInvitation representa un link de invitación para que un usuario real
// se una a un viaje. Tiene TTL y es de un solo uso.
type TripInvitation struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	TripID uint `json:"trip_id" gorm:"not null;index"`

	// Token firmado JWT (se distribuye por link)
	Token string `json:"token" gorm:"uniqueIndex;not null"`

	// Email opcional al que se dirigió la invitación (puede ser vacío para link abierto)
	Email string `json:"email" gorm:"type:varchar(150);index" validate:"omitempty,email"`

	// Rol que se asignará al aceptar
	Role TripMemberRole `json:"role" gorm:"type:varchar(20);not null;default:'member'" validate:"oneof=admin member viewer"`

	// Quién creó la invitación
	CreatedByUserID uint `json:"created_by_user_id" gorm:"not null;index"`

	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	UsedAt    *time.Time `json:"used_at"`

	// Si quien aceptó fue otro usuario, se guarda aquí para auditoría
	AcceptedByUserID *uint `json:"accepted_by_user_id" gorm:"index"`

	// Relaciones
	Trip *Trip `json:"trip,omitempty" gorm:"foreignKey:TripID"`
}

// IsExpired verifica si la invitación ya expiró
func (i *TripInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsUsed verifica si la invitación ya fue consumida
func (i *TripInvitation) IsUsed() bool {
	return i.UsedAt != nil
}

// IsValid verifica si la invitación puede ser aceptada
func (i *TripInvitation) IsValid() bool {
	return !i.IsExpired() && !i.IsUsed()
}

// MarkUsed marca la invitación como utilizada por un usuario dado
func (i *TripInvitation) MarkUsed(userID uint) {
	now := time.Now()
	i.UsedAt = &now
	i.AcceptedByUserID = &userID
}
