package repository

import (
	"context"
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BudgetSuggestionSlugStatsPostgres implementa BudgetSuggestionSlugStatsRepo.
type BudgetSuggestionSlugStatsPostgres struct {
	db *gorm.DB
}

func NewBudgetSuggestionSlugStatsPostgres(db *gorm.DB) *BudgetSuggestionSlugStatsPostgres {
	return &BudgetSuggestionSlugStatsPostgres{db: db}
}

// AddHits suma hits por slug para el día UTC indicado (upsert).
func (r *BudgetSuggestionSlugStatsPostgres) AddHits(ctx context.Context, statDate time.Time, counts map[string]int64) error {
	if len(counts) == 0 {
		return nil
	}
	d := time.Date(statDate.Year(), statDate.Month(), statDate.Day(), 0, 0, 0, 0, time.UTC)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for slug, n := range counts {
			if n <= 0 {
				continue
			}
			row := entity.BudgetSuggestionSlugStat{
				StatDate:     d,
				CategorySlug: slug,
				HitCount:     n,
			}
			err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "stat_date"}, {Name: "category_slug"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"hit_count": gorm.Expr("budget_suggestion_slug_stats.hit_count + ?", n),
				}),
			}).Create(&row).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}
