package repo

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// UserRepo define la interfaz para operaciones de usuario en la base de datos
type UserRepo interface {
	// Operaciones básicas CRUD
	Create(user *entity.User) error
	GetByID(id uint) (*entity.User, error)
	GetByEmail(email string) (*entity.User, error)
	Update(user *entity.User) error
	Delete(id uint) error

	// Operaciones específicas
	UpdateLastLogin(id uint) error
	SetActive(id uint, active bool) error
	SetVerified(id uint, verified bool) error

	// Consultas adicionales
	GetActiveUsers() ([]*entity.User, error)
	CountUsers() (int64, error)
	GetUsersRegisteredAfter(date time.Time) ([]*entity.User, error)
}
