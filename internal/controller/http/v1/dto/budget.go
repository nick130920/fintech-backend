package dto

// CreateBudgetRequest representa la estructura para crear un presupuesto
type CreateBudgetRequest struct {
	Year        int                       `json:"year" validate:"required,min=2020,max=2030"`
	Month       int                       `json:"month" validate:"required,min=1,max=12"`
	TotalAmount float64                   `json:"total_amount" validate:"required,gt=0"`
	Allocations []CreateAllocationRequest `json:"allocations" validate:"required,min=1,dive"`
}

// CreateAllocationRequest representa la asignación por categoría
type CreateAllocationRequest struct {
	CategoryID      uint    `json:"category_id" validate:"required"`
	AllocatedAmount float64 `json:"allocated_amount" validate:"required,gte=0"`
	AlertThreshold  float64 `json:"alert_threshold" validate:"min=0,max=1"` // 0.0 a 1.0
}

// UpdateBudgetRequest representa la estructura para actualizar un presupuesto
type UpdateBudgetRequest struct {
	TotalAmount    *float64                  `json:"total_amount" validate:"omitempty,gt=0"`
	Allocations    []UpdateAllocationRequest `json:"allocations" validate:"omitempty,dive"`
	AutoCreateNext *bool                     `json:"auto_create_next"`
}

// UpdateAllocationRequest representa la actualización de asignación
type UpdateAllocationRequest struct {
	ID              uint     `json:"id" validate:"required"`
	AllocatedAmount *float64 `json:"allocated_amount" validate:"omitempty,gte=0"`
	AlertThreshold  *float64 `json:"alert_threshold" validate:"omitempty,min=0,max=1"`
}

// BudgetSummaryResponse representa un resumen del presupuesto
type BudgetSummaryResponse struct {
	ID              uint                        `json:"id"`
	Year            int                         `json:"year"`
	Month           int                         `json:"month"`
	PeriodString    string                      `json:"period_string"`
	TotalAmount     float64                     `json:"total_amount"`
	SpentAmount     float64                     `json:"spent_amount"`
	RemainingAmount float64                     `json:"remaining_amount"`
	ProgressPercent float64                     `json:"progress_percent"`
	RemainingDays   int                         `json:"remaining_days"`
	IsActive        bool                        `json:"is_active"`
	IsCurrentMonth  bool                        `json:"is_current_month"`
	Allocations     []AllocationSummaryResponse `json:"allocations"`
}

// AllocationSummaryResponse representa el resumen de una asignación
type AllocationSummaryResponse struct {
	ID                uint                    `json:"id"`
	Category          CategorySummaryResponse `json:"category"`
	AllocatedAmount   float64                 `json:"allocated_amount"`
	SpentAmount       float64                 `json:"spent_amount"`
	RemainingAmount   float64                 `json:"remaining_amount"`
	ProgressPercent   float64                 `json:"progress_percent"`
	DailyLimit        float64                 `json:"daily_limit"`
	CurrentDailyLimit float64                 `json:"current_daily_limit"`
	AlertThreshold    float64                 `json:"alert_threshold"`
	IsOverBudget      bool                    `json:"is_over_budget"`
	ShouldAlert       bool                    `json:"should_alert"`
	AllocationPercent float64                 `json:"allocation_percent"`
}

// BudgetDashboardResponse representa el dashboard principal
type BudgetDashboardResponse struct {
	CurrentBudget  *BudgetSummaryResponse   `json:"current_budget"`
	TodayExpenses  []ExpenseSummaryResponse `json:"today_expenses"`
	TodayTotal     float64                  `json:"today_total"`
	WeekTotal      float64                  `json:"week_total"`
	MonthTotal     float64                  `json:"month_total"`
	CategoryAlerts []AllocationAlert        `json:"category_alerts"`
	QuickStats     BudgetQuickStats         `json:"quick_stats"`
}

// AllocationAlert representa una alerta de categoría
type AllocationAlert struct {
	CategoryName    string  `json:"category_name"`
	CategoryIcon    string  `json:"category_icon"`
	AllocatedAmount float64 `json:"allocated_amount"`
	SpentAmount     float64 `json:"spent_amount"`
	ProgressPercent float64 `json:"progress_percent"`
	AlertType       string  `json:"alert_type"` // "warning", "danger", "over_budget"
	Message         string  `json:"message"`
}

// BudgetQuickStats representa estadísticas rápidas
type BudgetQuickStats struct {
	DaysUntilPayday      int     `json:"days_until_payday"`
	AverageDailySpent    float64 `json:"average_daily_spent"`
	RecommendedDaily     float64 `json:"recommended_daily"`
	TotalCategories      int     `json:"total_categories"`
	CategoriesOnTrack    int     `json:"categories_on_track"`
	CategoriesOverBudget int     `json:"categories_over_budget"`
}

// UpdateSingleAllocationRequest representa la estructura para actualizar una asignación individual
type UpdateSingleAllocationRequest struct {
	AllocatedAmount *float64 `json:"allocated_amount" validate:"omitempty,gte=0"`
	AlertThreshold  *float64 `json:"alert_threshold" validate:"omitempty,min=0,max=1"`
}
