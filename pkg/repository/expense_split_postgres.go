package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// ExpenseSplitPostgres implementa repo.ExpenseSplitRepo
type ExpenseSplitPostgres struct {
	db *gorm.DB
}

// NewExpenseSplitPostgres construye el repositorio de splits
func NewExpenseSplitPostgres(db *gorm.DB) repo.ExpenseSplitRepo {
	return &ExpenseSplitPostgres{db: db}
}

func (r *ExpenseSplitPostgres) Create(split *entity.ExpenseSplit) error {
	return r.db.Create(split).Error
}

func (r *ExpenseSplitPostgres) GetByID(id uint) (*entity.ExpenseSplit, error) {
	var split entity.ExpenseSplit
	if err := r.db.Preload("Member").First(&split, id).Error; err != nil {
		return nil, err
	}
	return &split, nil
}

func (r *ExpenseSplitPostgres) GetByExpense(expenseID uint) ([]*entity.ExpenseSplit, error) {
	var splits []*entity.ExpenseSplit
	err := r.db.
		Preload("Member").
		Where("expense_id = ?", expenseID).
		Order("id ASC").
		Find(&splits).Error
	return splits, err
}

func (r *ExpenseSplitPostgres) GetByMember(memberID uint) ([]*entity.ExpenseSplit, error) {
	var splits []*entity.ExpenseSplit
	err := r.db.
		Preload("Expense").
		Where("member_id = ?", memberID).
		Order("created_at DESC").
		Find(&splits).Error
	return splits, err
}

func (r *ExpenseSplitPostgres) GetByTrip(tripID uint) ([]*entity.ExpenseSplit, error) {
	var splits []*entity.ExpenseSplit
	err := r.db.
		Preload("Expense").
		Preload("Member").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expenses.trip_id = ?", tripID).
		Where("expenses.deleted_at IS NULL").
		Order("expenses.date DESC").
		Find(&splits).Error
	return splits, err
}

func (r *ExpenseSplitPostgres) Update(split *entity.ExpenseSplit) error {
	return r.db.Save(split).Error
}

func (r *ExpenseSplitPostgres) DeleteByExpense(expenseID uint) error {
	return r.db.Where("expense_id = ?", expenseID).Delete(&entity.ExpenseSplit{}).Error
}

func (r *ExpenseSplitPostgres) BatchCreate(splits []*entity.ExpenseSplit) error {
	if len(splits) == 0 {
		return nil
	}
	return r.db.Create(&splits).Error
}
