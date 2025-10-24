package dto

import "time"

// BankNotificationWebhook representa una notificación bancaria recibida vía webhook
type BankNotificationWebhook struct {
	Message    string                 `json:"message" validate:"required" example:"Compra por $50.000 en SUPERMERCADO XYZ el 15/01/2024"`
	Phone      string                 `json:"phone" validate:"required" example:"+573001234567"`
	Channel    string                 `json:"channel" validate:"required,oneof=sms push email" example:"sms"`
	ReceivedAt time.Time              `json:"received_at" example:"2024-01-15T10:30:00Z"`
	BankCode   string                 `json:"bank_code,omitempty" example:"BANCOLOMBIA"`
	UserID     uint                   `json:"user_id,omitempty" example:"1"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SMSNotificationWebhook representa una notificación SMS específica
type SMSNotificationWebhook struct {
	Message    string    `json:"message" validate:"required"`
	From       string    `json:"from" validate:"required"` // Número del banco
	To         string    `json:"to" validate:"required"`   // Número del usuario
	ReceivedAt time.Time `json:"received_at"`
	Provider   string    `json:"provider,omitempty"` // Twilio, etc.
}

// ProcessPendingNotificationsRequest para procesar notificaciones pendientes
type ProcessPendingNotificationsRequest struct {
	UserID uint `json:"user_id" validate:"required" example:"1"`
	Limit  int  `json:"limit,omitempty" example:"10"`
}

// ProcessPendingNotificationsResponse respuesta del procesamiento
type ProcessPendingNotificationsResponse struct {
	TotalFound int    `json:"total_found" example:"15"`
	Processed  int    `json:"processed" example:"12"`
	Failed     int    `json:"failed" example:"3"`
	RequestID  string `json:"request_id"`
}

// NotificationStatsResponse estadísticas de notificaciones
type NotificationStatsResponse struct {
	TotalReceived     int                     `json:"total_received" example:"150"`
	TotalProcessed    int                     `json:"total_processed" example:"142"`
	TotalFailed       int                     `json:"total_failed" example:"8"`
	AutoCreated       int                     `json:"auto_created" example:"120"`
	PendingValidation int                     `json:"pending_validation" example:"22"`
	ByChannel         map[string]int          `json:"by_channel"`
	ByBank            map[string]int          `json:"by_bank"`
	ByDay             []DailyNotificationStat `json:"by_day"`
	AverageConfidence float64                 `json:"average_confidence" example:"0.85"`
	RequestID         string                  `json:"request_id"`
}

// DailyNotificationStat estadística diaria
type DailyNotificationStat struct {
	Date      string  `json:"date" example:"2024-01-15"`
	Count     int     `json:"count" example:"12"`
	Processed int     `json:"processed" example:"10"`
	Failed    int     `json:"failed" example:"2"`
	AvgAmount float64 `json:"avg_amount" example:"75000.50"`
}

// PendingNotification notificación pendiente de procesamiento
type PendingNotification struct {
	ID         uint      `json:"id"`
	RawMessage string    `json:"raw_message"`
	Channel    string    `json:"channel"`
	Phone      string    `json:"phone"`
	ReceivedAt time.Time `json:"received_at"`
	Attempts   int       `json:"attempts"`
	LastError  string    `json:"last_error,omitempty"`
}

// WebhookValidationRequest para validar webhooks
type WebhookValidationRequest struct {
	Challenge string `json:"challenge" validate:"required"`
	Token     string `json:"token" validate:"required"`
}

// WebhookValidationResponse respuesta de validación
type WebhookValidationResponse struct {
	Challenge string `json:"challenge"`
	Valid     bool   `json:"valid"`
}

// NotificationProcessingResult resultado del procesamiento
type NotificationProcessingResult struct {
	Success            bool    `json:"success"`
	TransactionID      uint    `json:"transaction_id,omitempty"`
	Amount             float64 `json:"amount,omitempty"`
	Description        string  `json:"description,omitempty"`
	Confidence         float64 `json:"confidence"`
	RequiresValidation bool    `json:"requires_validation"`
	Reason             string  `json:"reason,omitempty"`
	PatternUsed        string  `json:"pattern_used,omitempty"`
}

// BulkNotificationRequest para procesar múltiples notificaciones
type BulkNotificationRequest struct {
	Notifications []BankNotificationWebhook `json:"notifications" validate:"required,min=1,max=100"`
	UserID        uint                      `json:"user_id" validate:"required"`
}

// BulkNotificationResponse respuesta del procesamiento en lote
type BulkNotificationResponse struct {
	TotalReceived int                            `json:"total_received"`
	Processed     int                            `json:"processed"`
	Failed        int                            `json:"failed"`
	Results       []NotificationProcessingResult `json:"results"`
	RequestID     string                         `json:"request_id"`
}
