package entity

import (
	"time"

	"gorm.io/gorm"
)

// User representa un usuario del sistema
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Información personal
	FirstName   string     `json:"first_name" gorm:"not null" validate:"required,min=2,max=50"`
	LastName    string     `json:"last_name" gorm:"not null" validate:"required,min=2,max=50"`
	Email       string     `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Phone       string     `json:"phone" gorm:"" validate:"omitempty,min=10,max=15"`
	DateOfBirth *time.Time `json:"date_of_birth"`

	// Autenticación
	Password string `json:"-" gorm:"not null" validate:"required,min=8"`

	// Estado y configuración
	IsActive   bool   `json:"is_active" gorm:"default:true"`
	IsVerified bool   `json:"is_verified" gorm:"default:false"`
	Locale     string `json:"locale" gorm:"default:'es'"`
	Timezone   string `json:"timezone" gorm:"default:'America/Mexico_City'"`
	Currency   string `json:"currency" gorm:"default:'USD'" validate:"len=3"`

	// Campos de auditoría
	LastLoginAt *time.Time `json:"last_login_at"`
	LoginCount  int        `json:"login_count" gorm:"default:0"`
}

// ToPublic convierte un User a información pública (sin datos sensibles)
func (u *User) ToPublic() *UserPublic {
	userPublic := &UserPublic{
		ID:         u.ID,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Email:      u.Email,
		Phone:      u.Phone,
		IsActive:   u.IsActive,
		IsVerified: u.IsVerified,
		Locale:     u.Locale,
		Timezone:   u.Timezone,
		Currency:   u.Currency,
		CreatedAt:  u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Fecha de nacimiento formateada
	if u.DateOfBirth != nil {
		userPublic.DateOfBirth = u.DateOfBirth.Format("2006-01-02")
	}

	// Último login formateado
	if u.LastLoginAt != nil {
		userPublic.LastLoginAt = u.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return userPublic
}

// FullName retorna el nombre completo del usuario
func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// IsAccountActive verifica si la cuenta del usuario está activa
func (u *User) IsAccountActive() bool {
	return u.IsActive
}

// IsAccountVerified verifica si la cuenta del usuario está verificada
func (u *User) IsAccountVerified() bool {
	return u.IsVerified
}

// UpdateLastLogin actualiza la información de último acceso
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.LoginCount++
}

// UserPublic representa la información pública del usuario
type UserPublic struct {
	ID          uint   `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"date_of_birth,omitempty"` // Format: YYYY-MM-DD
	IsActive    bool   `json:"is_active"`
	IsVerified  bool   `json:"is_verified"`
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
	Currency    string `json:"currency"`
	CreatedAt   string `json:"created_at"`              // ISO format string
	LastLoginAt string `json:"last_login_at,omitempty"` // ISO format string
}
