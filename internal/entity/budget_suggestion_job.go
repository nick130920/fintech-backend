package entity

import "time"

// Estados del job de análisis SMS (sugerencias de presupuesto).
const (
	BudgetSuggestionJobPending    = "pending"
	BudgetSuggestionJobProcessing = "processing"
	BudgetSuggestionJobCompleted  = "completed"
	BudgetSuggestionJobFailed     = "failed"
)

// BudgetSuggestionJob almacena trabajo asíncrono para no mantener HTTP abierto durante la IA.
type BudgetSuggestionJob struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	UserID       uint      `gorm:"not null;index"`
	Status       string    `gorm:"size:20;not null;index"`
	MessagesJSON string    `gorm:"type:text;not null"`
	ResultJSON   string    `gorm:"type:text"`
	ErrorMessage string    `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (BudgetSuggestionJob) TableName() string {
	return "budget_suggestion_jobs"
}
