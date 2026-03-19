package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// BudgetSuggestionJobRepo persiste jobs de análisis SMS por lotes (async).
type BudgetSuggestionJobRepo interface {
	Create(job *entity.BudgetSuggestionJob) error
	FindByID(id string) (*entity.BudgetSuggestionJob, error)
	FindByIDAndUser(id string, userID uint) (*entity.BudgetSuggestionJob, error)
	Save(job *entity.BudgetSuggestionJob) error
}
