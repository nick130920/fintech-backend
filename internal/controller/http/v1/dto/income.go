package dto

// CreateIncomeRequest representa la petición para crear un ingreso
type CreateIncomeRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description string  `json:"description" validate:"required,max=255"`
	Source      string  `json:"source" validate:"required,oneof=salary freelance investment business rental bonus gift other"`
	Date        string  `json:"date" validate:"required"`
	Notes       string  `json:"notes,omitempty"`
	TaxDeducted float64 `json:"tax_deducted,omitempty" validate:"gte=0"`

	// Campos para ingresos recurrentes
	IsRecurring bool    `json:"is_recurring"`
	Frequency   *string `json:"frequency,omitempty" validate:"omitempty,oneof=weekly biweekly monthly quarterly yearly"`
	EndDate     *string `json:"end_date,omitempty"`
}

// UpdateIncomeRequest representa la petición para actualizar un ingreso
type UpdateIncomeRequest struct {
	Amount      *float64 `json:"amount,omitempty" validate:"omitempty,gt=0"`
	Description string   `json:"description,omitempty" validate:"omitempty,max=255"`
	Source      string   `json:"source,omitempty" validate:"omitempty,oneof=salary freelance investment business rental bonus gift other"`
	Date        string   `json:"date,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	TaxDeducted *float64 `json:"tax_deducted,omitempty" validate:"omitempty,gte=0"`

	// Campos para ingresos recurrentes
	IsRecurring *bool   `json:"is_recurring,omitempty"`
	Frequency   *string `json:"frequency,omitempty" validate:"omitempty,oneof=weekly biweekly monthly quarterly yearly"`
	EndDate     *string `json:"end_date,omitempty"`
}

// IncomeResponse representa la respuesta de un ingreso
type IncomeResponse struct {
	ID                 uint    `json:"id"`
	Amount             float64 `json:"amount"`
	FormattedAmount    string  `json:"formatted_amount"`
	NetAmount          float64 `json:"net_amount"`
	FormattedNetAmount string  `json:"formatted_net_amount"`
	Description        string  `json:"description"`
	Source             string  `json:"source"`
	SourceDisplayName  string  `json:"source_display_name"`
	Date               string  `json:"date"`
	Notes              string  `json:"notes"`
	Currency           string  `json:"currency"`
	TaxDeducted        float64 `json:"tax_deducted"`

	// Campos para ingresos recurrentes
	IsRecurring          bool    `json:"is_recurring"`
	Frequency            *string `json:"frequency,omitempty"`
	FrequencyDisplayName *string `json:"frequency_display_name,omitempty"`
	NextDate             *string `json:"next_date,omitempty"`
	EndDate              *string `json:"end_date,omitempty"`

	// Metadatos
	CanBeModified bool   `json:"can_be_modified"`
	CanBeDeleted  bool   `json:"can_be_deleted"`
	IsFuture      bool   `json:"is_future"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// IncomeSummaryResponse representa un resumen de ingresos
type IncomeSummaryResponse struct {
	ID                uint    `json:"id"`
	Amount            float64 `json:"amount"`
	FormattedAmount   string  `json:"formatted_amount"`
	Description       string  `json:"description"`
	Source            string  `json:"source"`
	SourceDisplayName string  `json:"source_display_name"`
	Date              string  `json:"date"`
	Currency          string  `json:"currency"`
	IsRecurring       bool    `json:"is_recurring"`
	CreatedAt         string  `json:"created_at"`
}

// IncomeStatsResponse representa estadísticas de ingresos
type IncomeStatsResponse struct {
	TotalIncome             float64                  `json:"total_income"`
	FormattedTotalIncome    string                   `json:"formatted_total_income"`
	MonthlyAverage          float64                  `json:"monthly_average"`
	FormattedMonthlyAverage string                   `json:"formatted_monthly_average"`
	IncomeBySource          []IncomeBySourceResponse `json:"income_by_source"`
	MonthlyIncome           []MonthlyIncomeResponse  `json:"monthly_income"`
	RecurringIncome         []IncomeSummaryResponse  `json:"recurring_income"`
	Currency                string                   `json:"currency"`
	Period                  string                   `json:"period"`
}

// IncomeBySourceResponse representa ingresos agrupados por fuente
type IncomeBySourceResponse struct {
	Source            string  `json:"source"`
	SourceDisplayName string  `json:"source_display_name"`
	TotalAmount       float64 `json:"total_amount"`
	FormattedAmount   string  `json:"formatted_amount"`
	Count             int     `json:"count"`
	Percentage        float64 `json:"percentage"`
}

// MonthlyIncomeResponse representa ingresos agrupados por mes
type MonthlyIncomeResponse struct {
	Year            int     `json:"year"`
	Month           int     `json:"month"`
	MonthName       string  `json:"month_name"`
	TotalAmount     float64 `json:"total_amount"`
	FormattedAmount string  `json:"formatted_amount"`
	Count           int     `json:"count"`
}

// RecurringIncomeProcessResponse respuesta para procesamiento de ingresos recurrentes
type RecurringIncomeProcessResponse struct {
	ProcessedCount   int                     `json:"processed_count"`
	ProcessedIncomes []IncomeSummaryResponse `json:"processed_incomes"`
	Message          string                  `json:"message"`
}
