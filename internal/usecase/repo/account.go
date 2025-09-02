package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// AccountRepo define la interfaz para operaciones de cuenta en la base de datos
type AccountRepo interface {
	// Operaciones básicas CRUD
	Create(account *entity.Account) error
	GetByID(id uint) (*entity.Account, error)
	GetByUserID(userID uint) ([]*entity.Account, error)
	Update(account *entity.Account) error
	Delete(id uint) error

	// Operaciones específicas de cuentas
	GetByUserIDAndType(userID uint, accountType entity.AccountType) ([]*entity.Account, error)
	UpdateBalance(id uint, newBalance float64) error
	AddToBalance(id uint, amount float64) error
	SubtractFromBalance(id uint, amount float64) error

	// Validaciones y consultas
	HasTransactions(id uint) (bool, error)
	SetActive(id uint, active bool) error
	GetActiveAccounts(userID uint) ([]*entity.Account, error)
	GetTotalBalance(userID uint) (float64, error)
	GetAccountsByTypeAndUser(userID uint, accountType entity.AccountType, activeOnly bool) ([]*entity.Account, error)
}
