package repo

import (
	"time"

	"github.com/nick130920/proyecto-fintech/internal/entity"
)

// TransactionRepo define la interfaz para operaciones de transacción en la base de datos
type TransactionRepo interface {
	// Operaciones básicas CRUD
	Create(transaction *entity.Transaction) error
	GetByID(id uint) (*entity.Transaction, error)
	GetByUserID(userID uint) ([]*entity.Transaction, error)
	Update(transaction *entity.Transaction) error
	Delete(id uint) error

	// Operaciones con manejo de balance
	CreateWithBalanceUpdate(transaction *entity.Transaction) error
	DeleteWithBalanceUpdate(id uint) error

	// Consultas específicas
	GetByUserIDWithFilter(userID uint, filter *entity.TransactionFilter) ([]*entity.TransactionSummary, error)
	CalculateAccountBalance(accountID uint) (float64, error)
	GetTotalsByType(userID uint, fromDate, toDate *time.Time) (map[entity.TransactionType]float64, error)
}
