package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// TransactionPostgres implementa la interfaz TransactionRepo usando PostgreSQL
type TransactionPostgres struct {
	db *gorm.DB
}

// NewTransactionPostgres crea una nueva instancia de TransactionPostgres
func NewTransactionPostgres(db *gorm.DB) repo.TransactionRepo {
	return &TransactionPostgres{db: db}
}

// Create crea una nueva transacción
func (r *TransactionPostgres) Create(transaction *entity.Transaction) error {
	return r.db.Create(transaction).Error
}

// CreateWithBalanceUpdate crea una transacción y actualiza los balances en una transacción de DB
func (r *TransactionPostgres) CreateWithBalanceUpdate(trans *entity.Transaction) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Crear la transacción
		if err := tx.Create(trans).Error; err != nil {
			return err
		}

		// Actualizar balance de la cuenta origen
		switch trans.Type {
		case entity.TransactionTypeIncome:
			// Sumar al balance
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance + ?", trans.Amount)).Error; err != nil {
				return err
			}
		case entity.TransactionTypeExpense:
			// Restar del balance
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance - ?", trans.Amount)).Error; err != nil {
				return err
			}
		case entity.TransactionTypeTransfer:
			// Restar de cuenta origen
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance - ?", trans.Amount)).Error; err != nil {
				return err
			}
			// Sumar a cuenta destino
			if trans.ToAccountID != nil {
				if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
					Where("id = ?", *trans.ToAccountID).
					Update("balance", gorm.Expr("balance + ?", trans.Amount)).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// GetByID obtiene una transacción por su ID
func (r *TransactionPostgres) GetByID(id uint) (*entity.Transaction, error) {
	var transaction entity.Transaction
	err := r.db.First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// GetByUserID obtiene transacciones de un usuario
func (r *TransactionPostgres) GetByUserID(userID uint) ([]*entity.Transaction, error) {
	var transactions []*entity.Transaction
	err := r.db.Where("user_id = ?", userID).Order("transaction_date DESC").Find(&transactions).Error
	return transactions, err
}

// GetByUserIDWithFilter obtiene transacciones con filtros
func (r *TransactionPostgres) GetByUserIDWithFilter(userID uint, filter *entity.TransactionFilter) ([]*entity.TransactionSummary, error) {
	query := r.db.Table("transactions t").
		Select(`t.id, t.type, t.status, t.amount, t.description, 
		        t.category_name, t.transaction_date, t.currency, t.created_at,
		        a.name as account_name, ta.name as to_account_name`).
		Joins("LEFT JOIN accounts a ON t.account_id = a.id").
		Joins("LEFT JOIN accounts ta ON t.to_account_id = ta.id").
		Where("t.user_id = ?", userID)

	// Aplicar filtros
	if filter.AccountID != nil {
		query = query.Where("t.account_id = ? OR t.to_account_id = ?", *filter.AccountID, *filter.AccountID)
	}
	if filter.Type != nil {
		query = query.Where("t.type = ?", *filter.Type)
	}
	if filter.Status != nil {
		query = query.Where("t.status = ?", *filter.Status)
	}
	if filter.CategoryID != nil {
		query = query.Where("t.category_id = ?", *filter.CategoryID)
	}
	if filter.FromDate != nil {
		query = query.Where("t.transaction_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("t.transaction_date <= ?", *filter.ToDate)
	}
	if filter.MinAmount != nil {
		query = query.Where("t.amount >= ?", *filter.MinAmount)
	}
	if filter.MaxAmount != nil {
		query = query.Where("t.amount <= ?", *filter.MaxAmount)
	}
	if filter.Search != "" {
		query = query.Where("t.description ILIKE ?", "%"+filter.Search+"%")
	}

	// Ordenamiento
	orderBy := "t.transaction_date"
	if filter.OrderBy != "" {
		orderBy = "t." + filter.OrderBy
	}
	orderDir := "DESC"
	if filter.OrderDir != "" {
		orderDir = filter.OrderDir
	}
	query = query.Order(orderBy + " " + orderDir)

	// Paginación
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	var summaries []*entity.TransactionSummary
	err := query.Find(&summaries).Error
	return summaries, err
}

// Update actualiza una transacción
func (r *TransactionPostgres) Update(transaction *entity.Transaction) error {
	return r.db.Save(transaction).Error
}

// Delete elimina una transacción (soft delete)
func (r *TransactionPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Transaction{}, id).Error
}

// DeleteWithBalanceUpdate elimina una transacción y revierte los cambios de balance
func (r *TransactionPostgres) DeleteWithBalanceUpdate(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Obtener la transacción primero
		var trans entity.Transaction
		if err := tx.First(&trans, id).Error; err != nil {
			return err
		}

		// Revertir cambios de balance
		switch trans.Type {
		case entity.TransactionTypeIncome:
			// Restar del balance (revertir suma)
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance - ?", trans.Amount)).Error; err != nil {
				return err
			}
		case entity.TransactionTypeExpense:
			// Sumar al balance (revertir resta)
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance + ?", trans.Amount)).Error; err != nil {
				return err
			}
		case entity.TransactionTypeTransfer:
			// Sumar a cuenta origen (revertir resta)
			if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
				Where("id = ?", trans.AccountID).
				Update("balance", gorm.Expr("balance + ?", trans.Amount)).Error; err != nil {
				return err
			}
			// Restar de cuenta destino (revertir suma)
			if trans.ToAccountID != nil {
				if err := tx.Model(&struct{ ID uint }{}).Table("accounts").
					Where("id = ?", *trans.ToAccountID).
					Update("balance", gorm.Expr("balance - ?", trans.Amount)).Error; err != nil {
					return err
				}
			}
		}

		// Eliminar la transacción
		return tx.Delete(&entity.Transaction{}, id).Error
	})
}

// CalculateAccountBalance calcula el balance de una cuenta basado en transacciones
func (r *TransactionPostgres) CalculateAccountBalance(accountID uint) (float64, error) {
	var balance float64

	// Sumar ingresos
	var income float64
	if err := r.db.Model(&entity.Transaction{}).
		Where("account_id = ? AND type = ? AND status = ?",
			accountID, entity.TransactionTypeIncome, entity.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&income).Error; err != nil {
		return 0, err
	}

	// Restar gastos
	var expenses float64
	if err := r.db.Model(&entity.Transaction{}).
		Where("account_id = ? AND type = ? AND status = ?",
			accountID, entity.TransactionTypeExpense, entity.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&expenses).Error; err != nil {
		return 0, err
	}

	// Restar transferencias salientes
	var transfersOut float64
	if err := r.db.Model(&entity.Transaction{}).
		Where("account_id = ? AND type = ? AND status = ?",
			accountID, entity.TransactionTypeTransfer, entity.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&transfersOut).Error; err != nil {
		return 0, err
	}

	// Sumar transferencias entrantes
	var transfersIn float64
	if err := r.db.Model(&entity.Transaction{}).
		Where("to_account_id = ? AND type = ? AND status = ?",
			accountID, entity.TransactionTypeTransfer, entity.TransactionStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&transfersIn).Error; err != nil {
		return 0, err
	}

	balance = income - expenses - transfersOut + transfersIn
	return balance, nil
}

// GetTotalsByType obtiene totales por tipo de transacción
func (r *TransactionPostgres) GetTotalsByType(userID uint, fromDate, toDate *time.Time) (map[entity.TransactionType]float64, error) {
	query := r.db.Model(&entity.Transaction{}).
		Select("type, SUM(amount) as total").
		Where("user_id = ? AND status = ?", userID, entity.TransactionStatusCompleted).
		Group("type")

	if fromDate != nil {
		query = query.Where("transaction_date >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("transaction_date <= ?", *toDate)
	}

	var results []struct {
		Type  entity.TransactionType `json:"type"`
		Total float64                `json:"total"`
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	totals := make(map[entity.TransactionType]float64)
	for _, result := range results {
		totals[result.Type] = result.Total
	}

	return totals, nil
}
