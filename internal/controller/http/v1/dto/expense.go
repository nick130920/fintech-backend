package dto

import "github.com/nick130920/fintech-backend/internal/entity"

// CreateExpenseRequest representa la estructura para crear un gasto
type CreateExpenseRequest struct {
	CategoryID  uint                 `json:"category_id" validate:"required"`
	Amount      float64              `json:"amount" validate:"required,gt=0"`
	Description string               `json:"description" validate:"required,min=1,max=500"`
	Date        string               `json:"date" validate:"required"` // Formato: 2006-01-02 o 2006-01-02T15:04:05Z
	Location    string               `json:"location" validate:"max=200"`
	Merchant    string               `json:"merchant" validate:"max=100"`
	Tags        []string             `json:"tags"`
	Notes       string               `json:"notes" validate:"max=1000"`
	Source      entity.ExpenseSource `json:"source" validate:"omitempty,oneof=manual whatsapp"`
	ReceiptURL  string               `json:"receipt_url" validate:"omitempty,url"`
}

// UpdateExpenseRequest representa la estructura para actualizar un gasto
type UpdateExpenseRequest struct {
	CategoryID  *uint    `json:"category_id"`
	Amount      *float64 `json:"amount" validate:"omitempty,gt=0"`
	Description string   `json:"description" validate:"omitempty,min=1,max=500"`
	Date        string   `json:"date"` // Formato: 2006-01-02 o 2006-01-02T15:04:05Z
	Location    string   `json:"location" validate:"max=200"`
	Merchant    string   `json:"merchant" validate:"max=100"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes" validate:"max=1000"`
	ReceiptURL  string   `json:"receipt_url" validate:"omitempty,url"`
}

// ExpenseSummaryResponse representa un resumen de gasto
type ExpenseSummaryResponse struct {
	ID              uint                    `json:"id"`
	Amount          float64                 `json:"amount"`
	FormattedAmount string                  `json:"formatted_amount"`
	Description     string                  `json:"description"`
	Date            string                  `json:"date"`     // ISO format
	TimeAgo         string                  `json:"time_ago"` // "Hace 2 horas"
	Category        CategorySummaryResponse `json:"category"`
	Source          entity.ExpenseSource    `json:"source"`
	Status          entity.ExpenseStatus    `json:"status"`
	Location        string                  `json:"location"`
	Merchant        string                  `json:"merchant"`
	Tags            []string                `json:"tags"`
	Notes           string                  `json:"notes"`
	Currency        string                  `json:"currency"`
	CanBeModified   bool                    `json:"can_be_modified"`
	CanBeCancelled  bool                    `json:"can_be_cancelled"`
	TriggeredAlert  bool                    `json:"triggered_alert"`
	CreatedAt       string                  `json:"created_at"`
}

// ExpenseDetailResponse representa el detalle completo de un gasto
type ExpenseDetailResponse struct {
	ExpenseSummaryResponse
	Reference    string           `json:"reference"`
	ReceiptURL   string           `json:"receipt_url"`
	ExchangeRate float64          `json:"exchange_rate"`
	RawData      string           `json:"raw_data,omitempty"`   // Solo para admin/debug
	Confidence   float64          `json:"confidence,omitempty"` // Solo para gastos automáticos
	BudgetImpact BudgetImpactInfo `json:"budget_impact"`
}

// BudgetImpactInfo representa el impacto en el presupuesto
type BudgetImpactInfo struct {
	AllocationID            uint    `json:"allocation_id"`
	PreviousSpent           float64 `json:"previous_spent"`
	NewSpent                float64 `json:"new_spent"`
	RemainingBudget         float64 `json:"remaining_budget"`
	PreviousProgressPercent float64 `json:"previous_progress_percent"`
	NewProgressPercent      float64 `json:"new_progress_percent"`
	ExceededBudget          bool    `json:"exceeded_budget"`
	ExceededDailyLimit      bool    `json:"exceeded_daily_limit"`
	NewDailyLimit           float64 `json:"new_daily_limit"`
}

// CreateExpenseResponse representa la respuesta al crear un gasto
type CreateExpenseResponse struct {
	Expense     ExpenseDetailResponse `json:"expense"`
	Alert       *ExpenseAlert         `json:"alert,omitempty"`       // Si se generó una alerta
	Suggestions []ExpenseSuggestion   `json:"suggestions,omitempty"` // Sugerencias para próximos gastos
}

// ExpenseAlert representa una alerta generada por un gasto
type ExpenseAlert struct {
	Type            string  `json:"type"` // "warning", "danger", "over_budget", "daily_limit"
	Title           string  `json:"title"`
	Message         string  `json:"message"`
	CategoryName    string  `json:"category_name"`
	SpentAmount     float64 `json:"spent_amount"`
	BudgetAmount    float64 `json:"budget_amount"`
	ProgressPercent float64 `json:"progress_percent"`
	Severity        string  `json:"severity"` // "low", "medium", "high"
}

// ExpenseSuggestion representa una sugerencia para el usuario
type ExpenseSuggestion struct {
	Type        string `json:"type"` // "reduce_spending", "switch_category", "adjust_budget"
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionText  string `json:"action_text,omitempty"`
	Impact      string `json:"impact,omitempty"` // "positive", "neutral", "negative"
}

// ExpenseStatsResponse representa estadísticas de gastos
type ExpenseStatsResponse struct {
	Period        string                 `json:"period"` // "today", "week", "month"
	TotalAmount   float64                `json:"total_amount"`
	TotalCount    int                    `json:"total_count"`
	AverageAmount float64                `json:"average_amount"`
	ByCategory    []CategoryExpenseStats `json:"by_category"`
	BySource      []SourceExpenseStats   `json:"by_source"`
	TopMerchants  []MerchantExpenseStats `json:"top_merchants"`
	DailyTrend    []DailyExpenseStats    `json:"daily_trend"`
}

// CategoryExpenseStats representa estadísticas por categoría
type CategoryExpenseStats struct {
	Category      CategorySummaryResponse `json:"category"`
	TotalAmount   float64                 `json:"total_amount"`
	Count         int                     `json:"count"`
	AverageAmount float64                 `json:"average_amount"`
	Percentage    float64                 `json:"percentage"` // Del total
}

// SourceExpenseStats representa estadísticas por fuente
type SourceExpenseStats struct {
	Source      entity.ExpenseSource `json:"source"`
	TotalAmount float64              `json:"total_amount"`
	Count       int                  `json:"count"`
	Percentage  float64              `json:"percentage"`
}

// MerchantExpenseStats representa estadísticas por comercio
type MerchantExpenseStats struct {
	Merchant      string  `json:"merchant"`
	TotalAmount   float64 `json:"total_amount"`
	Count         int     `json:"count"`
	AverageAmount float64 `json:"average_amount"`
}

// DailyExpenseStats representa estadísticas diarias
type DailyExpenseStats struct {
	Date        string  `json:"date"` // YYYY-MM-DD
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}

// QuickAddExpenseRequest representa una estructura simplificada para agregar gastos rápidos
type QuickAddExpenseRequest struct {
	CategoryID  uint    `json:"category_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description string  `json:"description" validate:"required,min=1,max=100"`
	Merchant    string  `json:"merchant" validate:"max=50"`
}

// ExpenseConfirmationRequest para confirmar gastos automáticos pendientes
type ExpenseConfirmationRequest struct {
	ExpenseIDs []uint `json:"expense_ids" validate:"required,min=1"`
	ConfirmAll bool   `json:"confirm_all"`
}

// SMSExpenseRequest para procesar gastos desde SMS
type SMSExpenseRequest struct {
	RawMessage   string `json:"raw_message" validate:"required"`
	SenderNumber string `json:"sender_number" validate:"required"`
	ReceivedAt   string `json:"received_at" validate:"required"`
}
