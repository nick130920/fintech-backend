package dto

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// CreateBankNotificationPatternRequest representa la estructura para crear una configuración de notificación
type CreateBankNotificationPatternRequest struct {
	BankAccountID       uint                       `json:"bank_account_id" validate:"required"`
	Name                string                     `json:"name" validate:"required,min=1,max=100"`
	Description         string                     `json:"description" validate:"omitempty,max=500"`
	Channel             entity.NotificationChannel `json:"channel" validate:"required,oneof=sms push email app"`
	ExampleMessage      string                     `json:"example_message" validate:"omitempty,max=2000"`
	RequiresValidation  bool                       `json:"requires_validation"`
	ConfidenceThreshold float64                    `json:"confidence_threshold" validate:"omitempty,gte=0,lte=1"`
	AutoApprove         bool                       `json:"auto_approve"`
	Priority            int                        `json:"priority" validate:"omitempty,gte=1"`
	IsDefault           bool                       `json:"is_default"`
	Tags                []string                   `json:"tags" validate:"omitempty"`
	Metadata            map[string]interface{}     `json:"metadata" validate:"omitempty"`
}

// UpdateBankNotificationPatternRequest representa la estructura para actualizar una configuración
type UpdateBankNotificationPatternRequest struct {
	Name                *string                `json:"name" validate:"omitempty,min=1,max=100"`
	Description         *string                `json:"description" validate:"omitempty,max=500"`
	ExampleMessage      *string                `json:"example_message" validate:"omitempty,max=2000"`
	RequiresValidation  *bool                  `json:"requires_validation"`
	ConfidenceThreshold *float64               `json:"confidence_threshold" validate:"omitempty,gte=0,lte=1"`
	AutoApprove         *bool                  `json:"auto_approve"`
	Priority            *int                   `json:"priority" validate:"omitempty,gte=1"`
	IsDefault           *bool                  `json:"is_default"`
	Tags                []string               `json:"tags"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// SetPatternStatusRequest representa la estructura para cambiar el estado
type SetPatternStatusRequest struct {
	Status entity.NotificationPatternStatus `json:"status" validate:"required,oneof=active inactive learning"`
}

// ProcessNotificationRequest representa la estructura para procesar una notificación
type ProcessNotificationRequest struct {
	Message    string    `json:"message" validate:"required,min=1"`
	Channel    string    `json:"channel" validate:"required"`
	Phone      string    `json:"phone,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
	BankCode   string    `json:"bank_code,omitempty"`
	UserID     uint      `json:"user_id,omitempty"`
}

// ProcessSMSWithAIRequest is the DTO for processing an SMS directly with AI.
// This is the new simplified flow that uses OpenRouter/Mistral to analyze SMS.
type ProcessSMSWithAIRequest struct {
	Message string `json:"message" validate:"required,min=1" example:"BBVA: Compra por $150.00 en OXXO el 28/01/26"`
}

// AnalyzeSMSBatchRequest is the DTO for analyzing multiple SMS for budget suggestions (no transaction creation).
type AnalyzeSMSBatchRequest struct {
	Messages []SMSMessageForAnalysis `json:"messages" validate:"required,dive"`
}

// SMSMessageForAnalysis represents one SMS in the batch.
type SMSMessageForAnalysis struct {
	Body string `json:"body" validate:"required"`
	Date string `json:"date"` // ISO8601 optional, for aggregation by period
}

// BudgetSuggestionCategory is one category in the budget suggestion response.
type BudgetSuggestionCategory struct {
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
	Count        int     `json:"count"`
}

// AnalyzeSMSBatchResponse is the response for analyze-sms-batch (suggestions only, no transactions created).
type AnalyzeSMSBatchResponse struct {
	Suggestions BudgetSuggestions `json:"suggestions"`
}

// StartSMSBatchJobResponse respuesta inmediata al crear job async (sin esperar a la IA).
type StartSMSBatchJobResponse struct {
	JobID       string              `json:"job_id,omitempty"`
	Status      string              `json:"status"` // pending | completed (vacío sync)
	Suggestions *BudgetSuggestions  `json:"suggestions,omitempty"`
}

// SMSBatchJobStatusResponse estado para GET .../jobs/:jobId (polling).
type SMSBatchJobStatusResponse struct {
	Status      string              `json:"status"` // pending | processing | completed | failed
	Suggestions *BudgetSuggestions  `json:"suggestions,omitempty"`
	Error       string              `json:"error,omitempty"`
}

// BudgetSuggestions is used by both analyze-sms-batch and analyze-statement.
type BudgetSuggestions struct {
	TotalExpense3m float64                    `json:"total_expense_3m"`
	ByCategory     []BudgetSuggestionCategory `json:"by_category"`
}

// BankNotificationPatternResponse representa la respuesta de una configuración de notificación
type BankNotificationPatternResponse struct {
	ID                  uint                             `json:"id"`
	BankAccountID       uint                             `json:"bank_account_id"`
	Name                string                           `json:"name"`
	Description         string                           `json:"description"`
	Channel             entity.NotificationChannel       `json:"channel"`
	Status              entity.NotificationPatternStatus `json:"status"`
	ExampleMessage      string                           `json:"example_message"`
	RequiresValidation  bool                             `json:"requires_validation"`
	ConfidenceThreshold float64                          `json:"confidence_threshold"`
	AutoApprove         bool                             `json:"auto_approve"`
	MatchCount          int                              `json:"match_count"`
	SuccessCount        int                              `json:"success_count"`
	SuccessRate         float64                          `json:"success_rate"`
	LastMatchedAt       *time.Time                       `json:"last_matched_at"`
	Priority            int                              `json:"priority"`
	IsDefault           bool                             `json:"is_default"`
	Tags                []string                         `json:"tags"`
	Metadata            map[string]interface{}           `json:"metadata"`
	CreatedAt           time.Time                        `json:"created_at"`
	UpdatedAt           time.Time                        `json:"updated_at"`
}

// ProcessedNotificationResponse representa la respuesta de procesamiento de una notificación
type ProcessedNotificationResponse struct {
	Success            bool                       `json:"success"`
	TransactionCreated bool                       `json:"transaction_created"`
	TransactionID      uint                       `json:"transaction_id,omitempty"`
	Amount             float64                    `json:"amount,omitempty"`
	Description        string                     `json:"description,omitempty"`
	BankAccountID      uint                       `json:"bank_account_id,omitempty"`
	Channel            entity.NotificationChannel `json:"channel,omitempty"`
	Message            string                     `json:"message,omitempty"`
	PatternID          *uint                      `json:"pattern_id,omitempty"`
	PatternUsed        string                     `json:"pattern_used,omitempty"`
	Confidence         float64                    `json:"confidence"`
	RequiresValidation bool                       `json:"requires_validation"`
	Reason             string                     `json:"reason,omitempty"`
	ExtractedData      map[string]interface{}     `json:"extracted_data,omitempty"`
}

// PatternStatisticsResponse representa estadísticas de patrones
type PatternStatisticsResponse struct {
	TotalPatterns      int     `json:"total_patterns"`
	ActivePatterns     int     `json:"active_patterns"`
	LearningPatterns   int     `json:"learning_patterns"`
	TotalMatches       int     `json:"total_matches"`
	TotalSuccesses     int     `json:"total_successes"`
	OverallSuccessRate float64 `json:"overall_success_rate"`
}

// PaginatedBankNotificationPatternResponse representa una respuesta paginada de patrones
type PaginatedBankNotificationPatternResponse struct {
	Data       []*BankNotificationPatternResponse `json:"data"`
	Total      int                                `json:"total"`
	Page       int                                `json:"page"`
	PerPage    int                                `json:"per_page"`
	TotalPages int                                `json:"total_pages"`
}
