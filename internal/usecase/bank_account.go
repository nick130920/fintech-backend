package usecase

import (
	"errors"
	"fmt"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// BankAccountUseCase contiene la lógica de negocio para cuentas bancarias
type BankAccountUseCase struct {
	bankAccountRepo repo.BankAccountRepo
	userRepo        repo.UserRepo
}

// NewBankAccountUseCase crea una nueva instancia de BankAccountUseCase
func NewBankAccountUseCase(bankAccountRepo repo.BankAccountRepo, userRepo repo.UserRepo) *BankAccountUseCase {
	return &BankAccountUseCase{
		bankAccountRepo: bankAccountRepo,
		userRepo:        userRepo,
	}
}

// CreateBankAccount crea una nueva cuenta bancaria
func (uc *BankAccountUseCase) CreateBankAccount(userID uint, req *dto.CreateBankAccountRequest) (*dto.BankAccountResponse, error) {
	// Verificar que el usuario existe
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Verificar que no existe otra cuenta con la misma máscara para este usuario
	existingAccount, err := uc.bankAccountRepo.GetByAccountNumberMask(userID, req.AccountNumberMask)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}
	if existingAccount != nil {
		return nil, errors.New("bank account with this number mask already exists")
	}

	// Crear la entidad cuenta bancaria
	bankAccount := &entity.BankAccount{
		UserID:                userID,
		BankName:              req.BankName,
		BankCode:              req.BankCode,
		BranchCode:            req.BranchCode,
		BranchName:            req.BranchName,
		AccountNumber:         req.AccountNumber, // Debería estar encriptado
		AccountNumberMask:     req.AccountNumberMask,
		AccountAlias:          req.AccountAlias,
		Type:                  req.Type,
		Color:                 req.Color,
		Icon:                  req.Icon,
		IsActive:              true,
		IsNotificationEnabled: req.IsNotificationEnabled,
		Currency:              req.Currency,
		NotificationPhone:     req.NotificationPhone,
		NotificationEmail:     req.NotificationEmail,
		MinAmountToNotify:     req.MinAmountToNotify,
		Notes:                 req.Notes,
	}

	// Crear la cuenta bancaria
	if err := uc.bankAccountRepo.Create(bankAccount); err != nil {
		return nil, fmt.Errorf("failed to create bank account: %w", err)
	}

	// Convertir a DTO de respuesta
	response := uc.toDTO(bankAccount)
	return response, nil
}

// GetBankAccount obtiene una cuenta bancaria por ID
func (uc *BankAccountUseCase) GetBankAccount(userID, bankAccountID uint) (*dto.BankAccountResponse, error) {
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return nil, errors.New("bank account not found")
	}

	// Verificar que la cuenta pertenece al usuario
	if bankAccount.UserID != userID {
		return nil, errors.New("unauthorized access to bank account")
	}

	response := uc.toDTO(bankAccount)
	return response, nil
}

// GetUserBankAccounts obtiene todas las cuentas bancarias de un usuario
func (uc *BankAccountUseCase) GetUserBankAccounts(userID uint, activeOnly bool) ([]*dto.BankAccountResponse, error) {
	var bankAccounts []*entity.BankAccount
	var err error

	if activeOnly {
		bankAccounts, err = uc.bankAccountRepo.GetActiveByUserID(userID)
	} else {
		bankAccounts, err = uc.bankAccountRepo.GetByUserID(userID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get bank accounts: %w", err)
	}

	responses := make([]*dto.BankAccountResponse, len(bankAccounts))
	for i, account := range bankAccounts {
		responses[i] = uc.toDTO(account)
	}

	return responses, nil
}

// GetBankAccountsByType obtiene cuentas bancarias por tipo
func (uc *BankAccountUseCase) GetBankAccountsByType(userID uint, accountType entity.BankAccountType) ([]*dto.BankAccountResponse, error) {
	bankAccounts, err := uc.bankAccountRepo.GetByUserIDAndType(userID, accountType)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank accounts by type: %w", err)
	}

	responses := make([]*dto.BankAccountResponse, len(bankAccounts))
	for i, account := range bankAccounts {
		responses[i] = uc.toDTO(account)
	}

	return responses, nil
}

// UpdateBankAccount actualiza una cuenta bancaria
func (uc *BankAccountUseCase) UpdateBankAccount(userID, bankAccountID uint, req *dto.UpdateBankAccountRequest) (*dto.BankAccountResponse, error) {
	// Obtener la cuenta bancaria existente
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return nil, errors.New("bank account not found")
	}

	// Verificar que la cuenta pertenece al usuario
	if bankAccount.UserID != userID {
		return nil, errors.New("unauthorized access to bank account")
	}

	// Actualizar los campos
	if req.BankName != "" {
		bankAccount.BankName = req.BankName
	}
	if req.AccountAlias != "" {
		bankAccount.AccountAlias = req.AccountAlias
	}
	if req.Color != "" {
		bankAccount.Color = req.Color
	}
	if req.Icon != "" {
		bankAccount.Icon = req.Icon
	}
	if req.IsNotificationEnabled != nil {
		bankAccount.IsNotificationEnabled = *req.IsNotificationEnabled
	}
	if req.NotificationPhone != "" {
		bankAccount.NotificationPhone = req.NotificationPhone
	}
	if req.NotificationEmail != "" {
		bankAccount.NotificationEmail = req.NotificationEmail
	}
	if req.MinAmountToNotify != nil {
		bankAccount.MinAmountToNotify = *req.MinAmountToNotify
	}
	if req.Notes != "" {
		bankAccount.Notes = req.Notes
	}

	// Guardar cambios
	if err := uc.bankAccountRepo.Update(bankAccount); err != nil {
		return nil, fmt.Errorf("failed to update bank account: %w", err)
	}

	response := uc.toDTO(bankAccount)
	return response, nil
}

// DeleteBankAccount elimina una cuenta bancaria
func (uc *BankAccountUseCase) DeleteBankAccount(userID, bankAccountID uint) error {
	// Verificar que la cuenta existe y pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return errors.New("bank account not found")
	}
	if bankAccount.UserID != userID {
		return errors.New("unauthorized access to bank account")
	}

	// Eliminar la cuenta bancaria
	if err := uc.bankAccountRepo.Delete(bankAccountID); err != nil {
		return fmt.Errorf("failed to delete bank account: %w", err)
	}

	return nil
}

// SetBankAccountActive cambia el estado activo de una cuenta bancaria
func (uc *BankAccountUseCase) SetBankAccountActive(userID, bankAccountID uint, active bool) error {
	// Verificar que la cuenta existe y pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return errors.New("bank account not found")
	}
	if bankAccount.UserID != userID {
		return errors.New("unauthorized access to bank account")
	}

	// Cambiar el estado
	if err := uc.bankAccountRepo.SetActive(bankAccountID, active); err != nil {
		return fmt.Errorf("failed to set bank account active status: %w", err)
	}

	return nil
}

// UpdateBankAccountBalance actualiza el balance de una cuenta bancaria
func (uc *BankAccountUseCase) UpdateBankAccountBalance(userID, bankAccountID uint, balance float64) error {
	// Verificar que la cuenta existe y pertenece al usuario
	bankAccount, err := uc.bankAccountRepo.GetByID(bankAccountID)
	if err != nil {
		return fmt.Errorf("failed to get bank account: %w", err)
	}
	if bankAccount == nil {
		return errors.New("bank account not found")
	}
	if bankAccount.UserID != userID {
		return errors.New("unauthorized access to bank account")
	}

	// Actualizar el balance
	if err := uc.bankAccountRepo.UpdateBalance(bankAccountID, balance); err != nil {
		return fmt.Errorf("failed to update bank account balance: %w", err)
	}

	return nil
}

// GetBankAccountSummary obtiene un resumen de las cuentas bancarias del usuario
func (uc *BankAccountUseCase) GetBankAccountSummary(userID uint) ([]*dto.BankAccountSummaryResponse, error) {
	summaries, err := uc.bankAccountRepo.GetSummaryByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bank account summary: %w", err)
	}

	responses := make([]*dto.BankAccountSummaryResponse, len(summaries))
	for i, summary := range summaries {
		responses[i] = uc.summaryToDTO(summary)
	}

	return responses, nil
}

// SearchBankAccounts busca cuentas bancarias con filtros
func (uc *BankAccountUseCase) SearchBankAccounts(userID uint, filter entity.BankAccountFilter) (*dto.PaginatedBankAccountResponse, error) {
	bankAccounts, total, err := uc.bankAccountRepo.GetWithFilters(userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search bank accounts: %w", err)
	}

	responses := make([]*dto.BankAccountResponse, len(bankAccounts))
	for i, account := range bankAccounts {
		responses[i] = uc.toDTO(account)
	}

	// Calcular páginas
	totalPages := int(total)
	if filter.Limit > 0 {
		totalPages = (int(total) + filter.Limit - 1) / filter.Limit
	}

	return &dto.PaginatedBankAccountResponse{
		Data:       responses,
		Total:      int(total),
		Page:       filter.Offset/filter.Limit + 1,
		PerPage:    filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// toDTO convierte una entidad BankAccount a DTO de respuesta
func (uc *BankAccountUseCase) toDTO(bankAccount *entity.BankAccount) *dto.BankAccountResponse {
	return &dto.BankAccountResponse{
		ID:                    bankAccount.ID,
		BankName:              bankAccount.BankName,
		BankCode:              bankAccount.BankCode,
		BranchCode:            bankAccount.BranchCode,
		BranchName:            bankAccount.BranchName,
		AccountNumberMask:     bankAccount.AccountNumberMask,
		AccountAlias:          bankAccount.AccountAlias,
		Type:                  bankAccount.Type,
		Color:                 bankAccount.Color,
		Icon:                  bankAccount.Icon,
		IsActive:              bankAccount.IsActive,
		IsNotificationEnabled: bankAccount.IsNotificationEnabled,
		Currency:              bankAccount.Currency,
		LastBalance:           bankAccount.LastBalance,
		LastBalanceUpdate:     bankAccount.LastBalanceUpdate,
		NotificationPhone:     bankAccount.NotificationPhone,
		NotificationEmail:     bankAccount.NotificationEmail,
		MinAmountToNotify:     bankAccount.MinAmountToNotify,
		Notes:                 bankAccount.Notes,
		DisplayName:           bankAccount.GetDisplayName(),
		CreatedAt:             bankAccount.CreatedAt,
		UpdatedAt:             bankAccount.UpdatedAt,
	}
}

// summaryToDTO convierte un resumen de cuenta bancaria a DTO
func (uc *BankAccountUseCase) summaryToDTO(summary *entity.BankAccountSummary) *dto.BankAccountSummaryResponse {
	return &dto.BankAccountSummaryResponse{
		ID:                summary.ID,
		BankName:          summary.BankName,
		ShortBankName:     summary.ShortBankName,
		AccountAlias:      summary.AccountAlias,
		AccountNumberMask: summary.AccountNumberMask,
		Type:              summary.Type,
		Color:             summary.Color,
		Icon:              summary.Icon,
		IsActive:          summary.IsActive,
		Currency:          summary.Currency,
		LastBalance:       summary.LastBalance,
		LastBalanceUpdate: summary.LastBalanceUpdate,
		DisplayName:       summary.DisplayName,
	}
}
