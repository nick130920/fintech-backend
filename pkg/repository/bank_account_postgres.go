package repository

import (
	"fmt"
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
	"gorm.io/gorm"
)

// BankAccountPostgres implementa BankAccountRepo usando PostgreSQL
type BankAccountPostgres struct {
	db *gorm.DB
}

// NewBankAccountPostgres crea una nueva instancia del repositorio de cuentas bancarias
func NewBankAccountPostgres(db *gorm.DB) repo.BankAccountRepo {
	return &BankAccountPostgres{db: db}
}

// Create crea una nueva cuenta bancaria
func (r *BankAccountPostgres) Create(bankAccount *entity.BankAccount) error {
	if err := r.db.Create(bankAccount).Error; err != nil {
		return fmt.Errorf("failed to create bank account: %w", err)
	}
	return nil
}

// GetByID obtiene una cuenta bancaria por ID
func (r *BankAccountPostgres) GetByID(id uint) (*entity.BankAccount, error) {
	var bankAccount entity.BankAccount
	if err := r.db.First(&bankAccount, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bank account by id %d: %w", id, err)
	}
	return &bankAccount, nil
}

// Update actualiza una cuenta bancaria existente
func (r *BankAccountPostgres) Update(bankAccount *entity.BankAccount) error {
	if err := r.db.Save(bankAccount).Error; err != nil {
		return fmt.Errorf("failed to update bank account: %w", err)
	}
	return nil
}

// Delete elimina una cuenta bancaria (soft delete)
func (r *BankAccountPostgres) Delete(id uint) error {
	if err := r.db.Delete(&entity.BankAccount{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete bank account: %w", err)
	}
	return nil
}

// GetByUserID obtiene todas las cuentas bancarias de un usuario
func (r *BankAccountPostgres) GetByUserID(userID uint) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank accounts for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}

// GetActiveByUserID obtiene todas las cuentas bancarias activas de un usuario
func (r *BankAccountPostgres) GetActiveByUserID(userID uint) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get active bank accounts for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}

// GetByUserIDAndType obtiene cuentas bancarias de un usuario por tipo
func (r *BankAccountPostgres) GetByUserIDAndType(userID uint, accountType entity.BankAccountType) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ? AND type = ?", userID, accountType).Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank accounts by type for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}

// GetByBankName obtiene cuentas bancarias de un usuario por nombre de banco
func (r *BankAccountPostgres) GetByBankName(userID uint, bankName string) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ? AND bank_name ILIKE ?", userID, "%"+bankName+"%").Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank accounts by bank name for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}

// GetByAccountNumberMask obtiene una cuenta bancaria por máscara de número
func (r *BankAccountPostgres) GetByAccountNumberMask(userID uint, mask string) (*entity.BankAccount, error) {
	var bankAccount entity.BankAccount
	if err := r.db.Where("user_id = ? AND account_number_mask = ?", userID, mask).First(&bankAccount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bank account by mask for user %d: %w", userID, err)
	}
	return &bankAccount, nil
}

// SearchByAlias busca cuentas bancarias por alias
func (r *BankAccountPostgres) SearchByAlias(userID uint, alias string) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ? AND account_alias ILIKE ?", userID, "%"+alias+"%").Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to search bank accounts by alias for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}

// GetWithFilters obtiene cuentas bancarias con filtros
func (r *BankAccountPostgres) GetWithFilters(userID uint, filter entity.BankAccountFilter) ([]*entity.BankAccount, int64, error) {
	query := r.db.Where("user_id = ?", userID)

	// Aplicar filtros
	if filter.BankName != "" {
		query = query.Where("bank_name ILIKE ?", "%"+filter.BankName+"%")
	}
	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Currency != "" {
		query = query.Where("currency = ?", filter.Currency)
	}
	if filter.Search != "" {
		query = query.Where("(account_alias ILIKE ? OR bank_name ILIKE ?)", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// Contar total
	var total int64
	if err := query.Model(&entity.BankAccount{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count bank accounts: %w", err)
	}

	// Aplicar ordenamiento
	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	orderDir := "DESC"
	if filter.OrderDir != "" {
		orderDir = filter.OrderDir
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, orderDir))

	// Aplicar paginación
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var bankAccounts []*entity.BankAccount
	if err := query.Find(&bankAccounts).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get bank accounts with filters: %w", err)
	}

	return bankAccounts, total, nil
}

// SetActive cambia el estado activo de una cuenta bancaria
func (r *BankAccountPostgres) SetActive(id uint, active bool) error {
	if err := r.db.Model(&entity.BankAccount{}).Where("id = ?", id).Update("is_active", active).Error; err != nil {
		return fmt.Errorf("failed to set active status for bank account %d: %w", id, err)
	}
	return nil
}

// SetNotificationEnabled cambia el estado de notificaciones de una cuenta bancaria
func (r *BankAccountPostgres) SetNotificationEnabled(id uint, enabled bool) error {
	if err := r.db.Model(&entity.BankAccount{}).Where("id = ?", id).Update("is_notification_enabled", enabled).Error; err != nil {
		return fmt.Errorf("failed to set notification status for bank account %d: %w", id, err)
	}
	return nil
}

// UpdateBalance actualiza el balance de una cuenta bancaria
func (r *BankAccountPostgres) UpdateBalance(id uint, balance float64) error {
	updates := map[string]interface{}{
		"last_balance":        balance,
		"last_balance_update": time.Now(),
	}
	if err := r.db.Model(&entity.BankAccount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update balance for bank account %d: %w", id, err)
	}
	return nil
}

// CountByUserID cuenta las cuentas bancarias de un usuario
func (r *BankAccountPostgres) CountByUserID(userID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&entity.BankAccount{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count bank accounts for user %d: %w", userID, err)
	}
	return count, nil
}

// GetSummaryByUserID obtiene un resumen de las cuentas bancarias de un usuario
func (r *BankAccountPostgres) GetSummaryByUserID(userID uint) ([]*entity.BankAccountSummary, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get bank accounts for summary: %w", err)
	}

	summaries := make([]*entity.BankAccountSummary, len(bankAccounts))
	for i, account := range bankAccounts {
		summary := account.ToSummary()
		summaries[i] = &summary
	}

	return summaries, nil
}

// GetNotificationEnabledAccounts obtiene cuentas bancarias con notificaciones habilitadas
func (r *BankAccountPostgres) GetNotificationEnabledAccounts(userID uint) ([]*entity.BankAccount, error) {
	var bankAccounts []*entity.BankAccount
	if err := r.db.Where("user_id = ? AND is_active = ? AND is_notification_enabled = ?", userID, true, true).Find(&bankAccounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification enabled bank accounts for user %d: %w", userID, err)
	}
	return bankAccounts, nil
}
