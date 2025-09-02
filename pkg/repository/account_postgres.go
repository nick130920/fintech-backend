package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// AccountPostgres implementa la interfaz AccountRepo usando PostgreSQL
type AccountPostgres struct {
	db *gorm.DB
}

// NewAccountPostgres crea una nueva instancia de AccountPostgres
func NewAccountPostgres(db *gorm.DB) repo.AccountRepo {
	return &AccountPostgres{db: db}
}

// Create crea una nueva cuenta en la base de datos
func (r *AccountPostgres) Create(account *entity.Account) error {
	return r.db.Create(account).Error
}

// GetByID obtiene una cuenta por su ID
func (r *AccountPostgres) GetByID(id uint) (*entity.Account, error) {
	var account entity.Account
	err := r.db.First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetByUserID obtiene todas las cuentas de un usuario
func (r *AccountPostgres) GetByUserID(userID uint) ([]*entity.Account, error) {
	var accounts []*entity.Account
	err := r.db.Where("user_id = ?", userID).Order("created_at ASC").Find(&accounts).Error
	return accounts, err
}

// GetByUserIDAndType obtiene cuentas de un usuario por tipo
func (r *AccountPostgres) GetByUserIDAndType(userID uint, accountType entity.AccountType) ([]*entity.Account, error) {
	var accounts []*entity.Account
	err := r.db.Where("user_id = ? AND type = ?", userID, accountType).Find(&accounts).Error
	return accounts, err
}

// Update actualiza una cuenta en la base de datos
func (r *AccountPostgres) Update(account *entity.Account) error {
	return r.db.Save(account).Error
}

// Delete elimina una cuenta (soft delete)
func (r *AccountPostgres) Delete(id uint) error {
	return r.db.Delete(&entity.Account{}, id).Error
}

// UpdateBalance actualiza el balance de una cuenta
func (r *AccountPostgres) UpdateBalance(id uint, newBalance float64) error {
	return r.db.Model(&entity.Account{}).Where("id = ?", id).Update("balance", newBalance).Error
}

// AddToBalance suma una cantidad al balance actual
func (r *AccountPostgres) AddToBalance(id uint, amount float64) error {
	return r.db.Model(&entity.Account{}).Where("id = ?", id).Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// SubtractFromBalance resta una cantidad del balance actual
func (r *AccountPostgres) SubtractFromBalance(id uint, amount float64) error {
	return r.db.Model(&entity.Account{}).Where("id = ?", id).Update("balance", gorm.Expr("balance - ?", amount)).Error
}

// HasTransactions verifica si una cuenta tiene transacciones asociadas
func (r *AccountPostgres) HasTransactions(id uint) (bool, error) {
	var count int64
	err := r.db.Table("transactions").Where("account_id = ? OR to_account_id = ?", id, id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetActive activa o desactiva una cuenta
func (r *AccountPostgres) SetActive(id uint, active bool) error {
	return r.db.Model(&entity.Account{}).Where("id = ?", id).Update("is_active", active).Error
}

// GetActiveAccounts obtiene todas las cuentas activas de un usuario
func (r *AccountPostgres) GetActiveAccounts(userID uint) ([]*entity.Account, error) {
	var accounts []*entity.Account
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&accounts).Error
	return accounts, err
}

// GetTotalBalance obtiene el balance total de todas las cuentas de un usuario
func (r *AccountPostgres) GetTotalBalance(userID uint) (float64, error) {
	var total float64
	err := r.db.Model(&entity.Account{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&total).Error
	return total, err
}

// GetAccountsByTypeAndUser obtiene cuentas por tipo y usuario con filtros adicionales
func (r *AccountPostgres) GetAccountsByTypeAndUser(userID uint, accountType entity.AccountType, activeOnly bool) ([]*entity.Account, error) {
	query := r.db.Where("user_id = ? AND type = ?", userID, accountType)

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var accounts []*entity.Account
	err := query.Find(&accounts).Error
	return accounts, err
}
