package usecase

import (
	"errors"
	"time"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
)

// TransactionUseCase contiene la lógica de negocio para transacciones
type TransactionUseCase struct {
	transactionRepo repo.TransactionRepo
	accountRepo     repo.AccountRepo
	userRepo        repo.UserRepo
}

// NewTransactionUseCase crea una nueva instancia de TransactionUseCase
func NewTransactionUseCase(
	transactionRepo repo.TransactionRepo,
	accountRepo repo.AccountRepo,
	userRepo repo.UserRepo,
) *TransactionUseCase {
	return &TransactionUseCase{
		transactionRepo: transactionRepo,
		accountRepo:     accountRepo,
		userRepo:        userRepo,
	}
}

// Create crea una nueva transacción
func (uc *TransactionUseCase) Create(userID uint, req *dto.CreateTransactionRequest) (*entity.Transaction, error) {
	// Verificar que el usuario existe y está activo
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if !user.IsAccountActive() {
		return nil, errors.New("user account is not active")
	}

	// Verificar que la cuenta existe y pertenece al usuario
	account, err := uc.accountRepo.GetByID(req.AccountID)
	if err != nil || account.UserID != userID {
		return nil, errors.New("account not found")
	}

	// Para transferencias, verificar cuenta destino
	var toAccount *entity.Account
	if req.Type == entity.TransactionTypeTransfer {
		if req.ToAccountID == nil {
			return nil, errors.New("to_account_id is required for transfers")
		}

		toAccount, err = uc.accountRepo.GetByID(*req.ToAccountID)
		if err != nil || toAccount.UserID != userID {
			return nil, errors.New("destination account not found")
		}

		if req.AccountID == *req.ToAccountID {
			return nil, errors.New("cannot transfer to the same account")
		}
	}

	// Validar fondos suficientes para gastos y transferencias
	if req.Type == entity.TransactionTypeExpense || req.Type == entity.TransactionTypeTransfer {
		if !account.CanDebit(req.Amount) {
			return nil, errors.New("insufficient funds")
		}
	}

	// Parsear fecha de transacción
	transactionDate, err := uc.parseTransactionDate(req.TransactionDate)
	if err != nil {
		return nil, errors.New("invalid transaction date format")
	}

	// Crear la entidad transacción
	newTransaction := &entity.Transaction{
		UserID:          userID,
		AccountID:       req.AccountID,
		ToAccountID:     req.ToAccountID,
		Type:            req.Type,
		Status:          entity.TransactionStatusCompleted,
		Amount:          req.Amount,
		Description:     req.Description,
		CategoryID:      req.CategoryID,
		TransactionDate: transactionDate,
		Location:        req.Location,
		Reference:       req.Reference,
		Notes:           req.Notes,
		Currency:        account.Currency, // Heredar moneda de la cuenta
		ExchangeRate:    1.0,
	}

	// Configurar moneda si se especifica
	if req.Currency != "" {
		newTransaction.Currency = req.Currency
	}

	// Configurar tags
	if len(req.Tags) > 0 {
		if err := newTransaction.SetTags(req.Tags); err != nil {
			return nil, errors.New("invalid tags format")
		}
	}

	// Validaciones adicionales de negocio
	if err := uc.validateTransactionRules(newTransaction, account, toAccount); err != nil {
		return nil, err
	}

	// Guardar la transacción usando transacción de base de datos
	if err := uc.transactionRepo.CreateWithBalanceUpdate(newTransaction); err != nil {
		return nil, err
	}

	return newTransaction, nil
}

// GetByUserID obtiene transacciones de un usuario con filtros
func (uc *TransactionUseCase) GetByUserID(userID uint, filter *entity.TransactionFilter) ([]*entity.TransactionSummary, error) {
	// Verificar que el usuario existe
	if _, err := uc.userRepo.GetByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	// Aplicar límites por defecto si no se especifican
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000 // Límite máximo
	}

	return uc.transactionRepo.GetByUserIDWithFilter(userID, filter)
}

// GetByID obtiene una transacción por ID
func (uc *TransactionUseCase) GetByID(userID, transactionID uint) (*entity.Transaction, error) {
	transaction, err := uc.transactionRepo.GetByID(transactionID)
	if err != nil {
		return nil, err
	}

	// Verificar que la transacción pertenece al usuario
	if transaction.UserID != userID {
		return nil, errors.New("transaction not found")
	}

	return transaction, nil
}

// Update actualiza una transacción
func (uc *TransactionUseCase) Update(userID, transactionID uint, req *dto.UpdateTransactionRequest) (*entity.Transaction, error) {
	transaction, err := uc.GetByID(userID, transactionID)
	if err != nil {
		return nil, err
	}

	// Verificar que la transacción puede ser modificada
	if !transaction.CanBeModified() {
		return nil, errors.New("transaction cannot be modified")
	}

	// Actualizar campos si se proporcionan
	if req.Description != "" {
		transaction.Description = req.Description
	}
	if req.CategoryID != nil {
		transaction.CategoryID = req.CategoryID
	}
	if req.Location != "" {
		transaction.Location = req.Location
	}
	if req.Reference != "" {
		transaction.Reference = req.Reference
	}
	if req.Notes != "" {
		transaction.Notes = req.Notes
	}
	if req.Status != "" {
		transaction.Status = req.Status
	}

	// Parsear fecha de transacción si se proporciona
	if req.TransactionDate != "" {
		if transactionDate, err := uc.parseTransactionDate(req.TransactionDate); err == nil {
			transaction.TransactionDate = transactionDate
		}
	}

	// Configurar tags si se proporcionan
	if len(req.Tags) > 0 {
		if err := transaction.SetTags(req.Tags); err != nil {
			return nil, errors.New("invalid tags format")
		}
	}

	if err := uc.transactionRepo.Update(transaction); err != nil {
		return nil, err
	}

	return transaction, nil
}

// Delete elimina una transacción (soft delete)
func (uc *TransactionUseCase) Delete(userID, transactionID uint) error {
	transaction, err := uc.GetByID(userID, transactionID)
	if err != nil {
		return err
	}

	// Verificar que la transacción puede ser eliminada
	if !transaction.CanBeModified() {
		return errors.New("transaction cannot be deleted")
	}

	// Eliminar la transacción y revertir el balance
	return uc.transactionRepo.DeleteWithBalanceUpdate(transactionID)
}

// Cancel cancela una transacción pendiente
func (uc *TransactionUseCase) Cancel(userID, transactionID uint) error {
	transaction, err := uc.GetByID(userID, transactionID)
	if err != nil {
		return err
	}

	if !transaction.CanBeCancelled() {
		return errors.New("transaction cannot be cancelled")
	}

	transaction.Cancel()
	return uc.transactionRepo.Update(transaction)
}

// GetAccountBalance calcula el balance actual de una cuenta basado en transacciones
func (uc *TransactionUseCase) GetAccountBalance(userID, accountID uint) (float64, error) {
	// Verificar que la cuenta pertenece al usuario
	account, err := uc.accountRepo.GetByID(accountID)
	if err != nil || account.UserID != userID {
		return 0, errors.New("account not found")
	}

	return uc.transactionRepo.CalculateAccountBalance(accountID)
}

// GetUserTotalsByType obtiene totales por tipo de transacción para un usuario
func (uc *TransactionUseCase) GetUserTotalsByType(userID uint, fromDate, toDate *time.Time) (map[entity.TransactionType]float64, error) {
	// Verificar que el usuario existe
	if _, err := uc.userRepo.GetByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	return uc.transactionRepo.GetTotalsByType(userID, fromDate, toDate)
}

// GetRecentTransactions obtiene las transacciones más recientes
func (uc *TransactionUseCase) GetRecentTransactions(userID uint, limit int) ([]*entity.TransactionSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // Valor por defecto
	}

	filter := &entity.TransactionFilter{
		Limit:    limit,
		Offset:   0,
		OrderBy:  "created_at",
		OrderDir: "DESC",
	}

	return uc.GetByUserID(userID, filter)
}

// validateTransactionRules valida las reglas de negocio para transacciones
func (uc *TransactionUseCase) validateTransactionRules(
	transaction *entity.Transaction,
	account *entity.Account,
	toAccount *entity.Account,
) error {
	// Validar que las cuentas estén activas
	if !account.IsActive {
		return errors.New("origin account is not active")
	}

	if toAccount != nil && !toAccount.IsActive {
		return errors.New("destination account is not active")
	}

	// Validar monto
	if transaction.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}

	// Validar fecha de transacción
	if transaction.TransactionDate.After(time.Now().Add(24 * time.Hour)) {
		return errors.New("transaction date cannot be more than 1 day in the future")
	}

	// Validar moneda
	if toAccount != nil && transaction.Currency != toAccount.Currency {
		return errors.New("currency mismatch between accounts")
	}

	return nil
}

// parseTransactionDate parsea una fecha de transacción desde diferentes formatos
func (uc *TransactionUseCase) parseTransactionDate(dateStr string) (time.Time, error) {
	// Intentar formato con hora
	if transactionDate, err := time.Parse("2006-01-02T15:04:05Z", dateStr); err == nil {
		return transactionDate, nil
	}

	// Intentar formato ISO 8601
	if transactionDate, err := time.Parse("2006-01-02T15:04:05Z07:00", dateStr); err == nil {
		return transactionDate, nil
	}

	// Intentar formato solo fecha
	if transactionDate, err := time.Parse("2006-01-02", dateStr); err == nil {
		return transactionDate, nil
	}

	return time.Time{}, errors.New("invalid date format")
}
