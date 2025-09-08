package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// BankAccountRepo define la interfaz para operaciones de cuenta bancaria en la base de datos
type BankAccountRepo interface {
	// Operaciones básicas CRUD
	Create(bankAccount *entity.BankAccount) error
	GetByID(id uint) (*entity.BankAccount, error)
	Update(bankAccount *entity.BankAccount) error
	Delete(id uint) error

	// Operaciones específicas del usuario
	GetByUserID(userID uint) ([]*entity.BankAccount, error)
	GetActiveByUserID(userID uint) ([]*entity.BankAccount, error)
	GetByUserIDAndType(userID uint, accountType entity.BankAccountType) ([]*entity.BankAccount, error)

	// Operaciones de búsqueda
	GetByBankName(userID uint, bankName string) ([]*entity.BankAccount, error)
	GetByAccountNumberMask(userID uint, mask string) (*entity.BankAccount, error)
	SearchByAlias(userID uint, alias string) ([]*entity.BankAccount, error)
	GetWithFilters(userID uint, filter entity.BankAccountFilter) ([]*entity.BankAccount, int64, error)

	// Operaciones de estado
	SetActive(id uint, active bool) error
	SetNotificationEnabled(id uint, enabled bool) error
	UpdateBalance(id uint, balance float64) error

	// Estadísticas y consultas especiales
	CountByUserID(userID uint) (int64, error)
	GetSummaryByUserID(userID uint) ([]*entity.BankAccountSummary, error)
	GetNotificationEnabledAccounts(userID uint) ([]*entity.BankAccount, error)
}
