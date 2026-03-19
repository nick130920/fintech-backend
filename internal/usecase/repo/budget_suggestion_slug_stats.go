package repo

import (
	"context"
	"time"
)

// BudgetSuggestionSlugStatsRepo persiste conteos agregados por slug (sin user_id).
type BudgetSuggestionSlugStatsRepo interface {
	AddHits(ctx context.Context, statDate time.Time, counts map[string]int64) error
}
