package usecase

import (
	"errors"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
)

// AccountUseCase contiene la lógica de negocio para cuentas
type AccountUseCase struct {
	accountRepo repo.AccountRepo
	userRepo    repo.UserRepo
}

// NewAccountUseCase crea una nueva instancia de AccountUseCase
func NewAccountUseCase(accountRepo repo.AccountRepo, userRepo repo.UserRepo) *AccountUseCase {
	return &AccountUseCase{
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

// Create crea una nueva cuenta
func (uc *AccountUseCase) Create(userID uint, req *dto.CreateAccountRequest) (*entity.Account, error) {
	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsAccountActive() {
		return nil, errors.New("user account is not active")
	}

	// Crear la entidad cuenta
	newAccount := &entity.Account{
		UserID:         userID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		InitialBalance: req.InitialBalance,
		Balance:        req.InitialBalance, // El balance inicial es igual al balance actual
		CreditLimit:    req.CreditLimit,
		BankName:       req.BankName,
		AccountNumber:  req.AccountNumber,
		Currency:       req.Currency,
		Color:          req.Color,
		Icon:           req.Icon,
		IsActive:       true,
	}

	// Configurar valores por defecto
	if newAccount.Currency == "" {
		newAccount.Currency = "MXN"
	}
	if newAccount.Color == "" {
		newAccount.Color = "#007bff"
	}

	// Validaciones de negocio
	if err := uc.validateAccountRules(newAccount); err != nil {
		return nil, err
	}

	if err := uc.accountRepo.Create(newAccount); err != nil {
		return nil, err
	}

	return newAccount, nil
}

// GetByUserID obtiene todas las cuentas de un usuario
func (uc *AccountUseCase) GetByUserID(userID uint) ([]*entity.Account, error) {
	// Verificar que el usuario existe
	if _, err := uc.userRepo.GetByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	return uc.accountRepo.GetByUserID(userID)
}

// GetByID obtiene una cuenta por ID
func (uc *AccountUseCase) GetByID(userID, accountID uint) (*entity.Account, error) {
	account, err := uc.accountRepo.GetByID(accountID)
	if err != nil {
		return nil, err
	}

	// Verificar que la cuenta pertenece al usuario
	if account.UserID != userID {
		return nil, errors.New("account not found")
	}

	return account, nil
}

// Update actualiza una cuenta
func (uc *AccountUseCase) Update(userID, accountID uint, req *dto.UpdateAccountRequest) (*entity.Account, error) {
	account, err := uc.GetByID(userID, accountID)
	if err != nil {
		return nil, err
	}

	// Actualizar campos si se proporcionan
	if req.Name != "" {
		account.Name = req.Name
	}
	if req.Description != "" {
		account.Description = req.Description
	}
	if req.Type != "" {
		account.Type = req.Type
	}
	if req.CreditLimit >= 0 {
		account.CreditLimit = req.CreditLimit
	}
	if req.BankName != "" {
		account.BankName = req.BankName
	}
	if req.Color != "" {
		account.Color = req.Color
	}
	if req.Icon != "" {
		account.Icon = req.Icon
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.LowBalanceAlert != nil {
		account.LowBalanceAlert = *req.LowBalanceAlert
	}
	if req.LowBalanceLimit >= 0 {
		account.LowBalanceLimit = req.LowBalanceLimit
	}

	// Validaciones de negocio
	if err := uc.validateAccountRules(account); err != nil {
		return nil, err
	}

	if err := uc.accountRepo.Update(account); err != nil {
		return nil, err
	}

	return account, nil
}

// Delete elimina una cuenta (soft delete)
func (uc *AccountUseCase) Delete(userID, accountID uint) error {
	_, err := uc.GetByID(userID, accountID)
	if err != nil {
		return err
	}

	// Verificar que la cuenta no tenga transacciones
	hasTransactions, err := uc.accountRepo.HasTransactions(accountID)
	if err != nil {
		return err
	}

	if hasTransactions {
		return errors.New("cannot delete account with existing transactions")
	}

	return uc.accountRepo.Delete(accountID)
}

// GetAccountSummaries obtiene resúmenes de todas las cuentas del usuario
func (uc *AccountUseCase) GetAccountSummaries(userID uint) ([]entity.AccountSummary, error) {
	accounts, err := uc.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	summaries := make([]entity.AccountSummary, len(accounts))
	for i, account := range accounts {
		summaries[i] = account.ToSummary()
	}

	return summaries, nil
}

// GetAccountsByType obtiene cuentas por tipo
func (uc *AccountUseCase) GetAccountsByType(userID uint, accountType entity.AccountType, activeOnly bool) ([]*entity.Account, error) {
	// Verificar que el usuario existe
	if _, err := uc.userRepo.GetByID(userID); err != nil {
		return nil, errors.New("user not found")
	}

	return uc.accountRepo.GetAccountsByTypeAndUser(userID, accountType, activeOnly)
}

// GetTotalBalance obtiene el balance total de todas las cuentas activas
func (uc *AccountUseCase) GetTotalBalance(userID uint) (float64, error) {
	// Verificar que el usuario existe
	if _, err := uc.userRepo.GetByID(userID); err != nil {
		return 0, errors.New("user not found")
	}

	return uc.accountRepo.GetTotalBalance(userID)
}

// TransferBalance transfiere dinero entre cuentas del mismo usuario
func (uc *AccountUseCase) TransferBalance(userID uint, fromAccountID, toAccountID uint, amount float64) error {
	// Obtener cuentas
	fromAccount, err := uc.GetByID(userID, fromAccountID)
	if err != nil {
		return err
	}

	toAccount, err := uc.GetByID(userID, toAccountID)
	if err != nil {
		return err
	}

	// Validar transferencia
	if !fromAccount.IsActive || !toAccount.IsActive {
		return errors.New("both accounts must be active")
	}

	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}

	if !fromAccount.CanDebit(amount) {
		return errors.New("insufficient funds")
	}

	// Realizar transferencia
	fromAccount.Debit(amount)
	toAccount.Credit(amount)

	// Guardar cambios
	if err := uc.accountRepo.Update(fromAccount); err != nil {
		return err
	}

	if err := uc.accountRepo.Update(toAccount); err != nil {
		// Revertir cambios en caso de error
		fromAccount.Credit(amount)
		uc.accountRepo.Update(fromAccount)
		return err
	}

	return nil
}

// validateAccountRules valida las reglas de negocio para cuentas
func (uc *AccountUseCase) validateAccountRules(account *entity.Account) error {
	// Validar que las tarjetas de crédito tengan límite
	if account.IsCredit() && account.CreditLimit <= 0 {
		return errors.New("credit accounts must have a positive credit limit")
	}

	// Validar que el balance inicial no sea negativo para cuentas no-crédito
	if !account.IsCredit() && account.InitialBalance < 0 {
		return errors.New("initial balance cannot be negative for non-credit accounts")
	}

	// Validar límites de alertas
	if account.LowBalanceAlert && account.LowBalanceLimit < 0 {
		return errors.New("low balance limit cannot be negative")
	}

	return nil
}
