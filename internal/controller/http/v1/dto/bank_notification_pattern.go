package dto

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// CreateBankNotificationPatternRequest representa la estructura para crear un patrón de notificación
type CreateBankNotificationPatternRequest struct {
	BankAccountID       uint                       `json:"bank_account_id" validate:"required"`
	Name                string                     `json:"name" validate:"required,min=1,max=100"`
	Description         string                     `json:"description" validate:"omitempty,max=500"`
	Channel             entity.NotificationChannel `json:"channel" validate:"required,oneof=sms push email app"`
	MessagePattern      string                     `json:"message_pattern" validate:"omitempty,max=2000"`
	ExampleMessage      string                     `json:"example_message" validate:"omitempty,max=2000"`
	KeywordsTrigger     []string                   `json:"keywords_trigger" validate:"omitempty"`
	KeywordsExclude     []string                   `json:"keywords_exclude" validate:"omitempty"`
	AmountRegex         string                     `json:"amount_regex" validate:"omitempty,max=500"`
	DateRegex           string                     `json:"date_regex" validate:"omitempty,max=500"`
	DescriptionRegex    string                     `json:"description_regex" validate:"omitempty,max=500"`
	MerchantRegex       string                     `json:"merchant_regex" validate:"omitempty,max=500"`
	RequiresValidation  bool                       `json:"requires_validation"`
	ConfidenceThreshold float64                    `json:"confidence_threshold" validate:"omitempty,gte=0,lte=1"`
	AutoApprove         bool                       `json:"auto_approve"`
	Priority            int                        `json:"priority" validate:"omitempty,gte=1"`
	IsDefault           bool                       `json:"is_default"`
	Tags                []string                   `json:"tags" validate:"omitempty"`
	Metadata            map[string]interface{}     `json:"metadata" validate:"omitempty"`
}

// UpdateBankNotificationPatternRequest representa la estructura para actualizar un patrón
type UpdateBankNotificationPatternRequest struct {
	Name                *string                `json:"name" validate:"omitempty,min=1,max=100"`
	Description         *string                `json:"description" validate:"omitempty,max=500"`
	MessagePattern      *string                `json:"message_pattern" validate:"omitempty,max=2000"`
	ExampleMessage      *string                `json:"example_message" validate:"omitempty,max=2000"`
	KeywordsTrigger     []string               `json:"keywords_trigger"`
	KeywordsExclude     []string               `json:"keywords_exclude"`
	AmountRegex         *string                `json:"amount_regex" validate:"omitempty,max=500"`
	DateRegex           *string                `json:"date_regex" validate:"omitempty,max=500"`
	DescriptionRegex    *string                `json:"description_regex" validate:"omitempty,max=500"`
	MerchantRegex       *string                `json:"merchant_regex" validate:"omitempty,max=500"`
	RequiresValidation  *bool                  `json:"requires_validation"`
	ConfidenceThreshold *float64               `json:"confidence_threshold" validate:"omitempty,gte=0,lte=1"`
	AutoApprove         *bool                  `json:"auto_approve"`
	Priority            *int                   `json:"priority" validate:"omitempty,gte=1"`
	IsDefault           *bool                  `json:"is_default"`
	Tags                []string               `json:"tags"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// SetPatternStatusRequest representa la estructura para cambiar el estado de un patrón
type SetPatternStatusRequest struct {
	Status entity.NotificationPatternStatus `json:"status" validate:"required,oneof=active inactive learning"`
}

// ProcessNotificationRequest representa la estructura para procesar una notificación
type ProcessNotificationRequest struct {
	BankAccountID uint                       `json:"bank_account_id" validate:"required"`
	Channel       entity.NotificationChannel `json:"channel" validate:"required,oneof=sms push email app"`
	Message       string                     `json:"message" validate:"required,min=1"`
}

// BankNotificationPatternResponse representa la respuesta de un patrón de notificación
type BankNotificationPatternResponse struct {
	ID                  uint                             `json:"id"`
	BankAccountID       uint                             `json:"bank_account_id"`
	Name                string                           `json:"name"`
	Description         string                           `json:"description"`
	Channel             entity.NotificationChannel       `json:"channel"`
	Status              entity.NotificationPatternStatus `json:"status"`
	MessagePattern      string                           `json:"message_pattern"`
	ExampleMessage      string                           `json:"example_message"`
	KeywordsTrigger     []string                         `json:"keywords_trigger"`
	KeywordsExclude     []string                         `json:"keywords_exclude"`
	AmountRegex         string                           `json:"amount_regex"`
	DateRegex           string                           `json:"date_regex"`
	DescriptionRegex    string                           `json:"description_regex"`
	MerchantRegex       string                           `json:"merchant_regex"`
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
	BankAccountID      uint                       `json:"bank_account_id"`
	Channel            entity.NotificationChannel `json:"channel"`
	Message            string                     `json:"message"`
	Processed          bool                       `json:"processed"`
	PatternID          *uint                      `json:"pattern_id,omitempty"`
	PatternName        string                     `json:"pattern_name,omitempty"`
	Confidence         float64                    `json:"confidence"`
	RequiresValidation bool                       `json:"requires_validation"`
	ExtractedData      map[string]interface{}     `json:"extracted_data"`
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
