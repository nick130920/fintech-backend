package entity

import "time"

// BudgetSuggestionSlugStat agrega hits anónimos por slug de IA y día (UTC) para mejora de prompts.
type BudgetSuggestionSlugStat struct {
	ID           uint      `gorm:"primaryKey"`
	StatDate     time.Time `gorm:"type:date;not null;uniqueIndex:uq_budget_slug_stat"`
	CategorySlug string    `gorm:"size:32;not null;uniqueIndex:uq_budget_slug_stat"`
	HitCount     int64     `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (BudgetSuggestionSlugStat) TableName() string {
	return "budget_suggestion_slug_stats"
}
