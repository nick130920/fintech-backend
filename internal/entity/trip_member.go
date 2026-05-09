package entity

import (
	"time"

	"gorm.io/gorm"
)

// TripMemberRole define el rol de un miembro dentro de un viaje
type TripMemberRole string

const (
	TripMemberRoleOwner  TripMemberRole = "owner"
	TripMemberRoleAdmin  TripMemberRole = "admin"
	TripMemberRoleMember TripMemberRole = "member"
	TripMemberRoleViewer TripMemberRole = "viewer"
)

// TripMember representa a una persona dentro de un viaje. Puede ser un usuario
// real (UserID != nil) o un invitado "fantasma" (IsGhost = true) que solo
// existe para distribuir gastos sin acceder a la app.
type TripMember struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	TripID uint  `json:"trip_id" gorm:"not null;index"`
	UserID *uint `json:"user_id" gorm:"index"`

	// Datos visibles del miembro
	DisplayName string  `json:"display_name" gorm:"not null" validate:"required,min=1,max=120"`
	Email       *string `json:"email" gorm:"type:varchar(150)" validate:"omitempty,email"`
	AvatarURL   string  `json:"avatar_url" validate:"omitempty,url"`

	// Rol y bandera de invitado fantasma
	Role    TripMemberRole `json:"role" gorm:"type:varchar(20);not null;default:'member'" validate:"oneof=owner admin member viewer"`
	IsGhost bool           `json:"is_ghost" gorm:"default:false;index"`

	JoinedAt time.Time `json:"joined_at" gorm:"not null"`

	// Relaciones
	Trip *Trip `json:"trip,omitempty" gorm:"foreignKey:TripID"`
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// IsOwner indica si el miembro es el dueño del viaje
func (m *TripMember) IsOwner() bool {
	return m.Role == TripMemberRoleOwner
}

// CanManage indica si el miembro puede modificar el viaje y otros miembros
func (m *TripMember) CanManage() bool {
	return m.Role == TripMemberRoleOwner || m.Role == TripMemberRoleAdmin
}

// CanRegisterExpenses indica si el miembro puede registrar gastos
func (m *TripMember) CanRegisterExpenses() bool {
	return m.Role != TripMemberRoleViewer
}

// MatchesUser comprueba si el miembro corresponde a un usuario real dado
func (m *TripMember) MatchesUser(userID uint) bool {
	return m.UserID != nil && *m.UserID == userID
}
