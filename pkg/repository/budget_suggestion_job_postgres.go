package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// BudgetSuggestionJobPostgres implementa BudgetSuggestionJobRepo.
type BudgetSuggestionJobPostgres struct {
	db *gorm.DB
}

// NewBudgetSuggestionJobPostgres crea el repositorio.
func NewBudgetSuggestionJobPostgres(db *gorm.DB) *BudgetSuggestionJobPostgres {
	return &BudgetSuggestionJobPostgres{db: db}
}

var _ repo.BudgetSuggestionJobRepo = (*BudgetSuggestionJobPostgres)(nil)

func (r *BudgetSuggestionJobPostgres) Create(job *entity.BudgetSuggestionJob) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}
	return r.db.Create(job).Error
}

func (r *BudgetSuggestionJobPostgres) FindByID(id string) (*entity.BudgetSuggestionJob, error) {
	var j entity.BudgetSuggestionJob
	if err := r.db.Where("id = ?", id).First(&j).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

func (r *BudgetSuggestionJobPostgres) FindByIDAndUser(id string, userID uint) (*entity.BudgetSuggestionJob, error) {
	var j entity.BudgetSuggestionJob
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&j).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &j, nil
}

func (r *BudgetSuggestionJobPostgres) Save(job *entity.BudgetSuggestionJob) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}
	return r.db.Save(job).Error
}
