package entity

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// TransactionType define los tipos de transacción
type TransactionType string

const (
	TransactionTypeIncome   TransactionType = "income"   // Ingreso
	TransactionTypeExpense  TransactionType = "expense"  // Gasto
	TransactionTypeTransfer TransactionType = "transfer" // Transferencia entre cuentas
)

// TransactionStatus define el estado de la transacción
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"   // Pendiente
	TransactionStatusCompleted TransactionStatus = "completed" // Completada
	TransactionStatusCancelled TransactionStatus = "cancelled" // Cancelada
)

// TransactionSource define el origen de la transacción
type TransactionSource string

const (
	TransactionSourceNotification TransactionSource = "notification" // Desde notificación bancaria
	TransactionSourceManual       TransactionSource = "manual"       // Ingresada manualmente
	TransactionSourceIntegration  TransactionSource = "integration"  // Desde integración bancaria
	TransactionSourceImport       TransactionSource = "import"       // Importada desde archivo
)

// ValidationStatus define el estado de validación de la transacción
type ValidationStatus string

const (
	ValidationStatusAuto     ValidationStatus = "auto"             // Validada automáticamente
	ValidationStatusPending  ValidationStatus = "pending_review"   // Pendiente de revisión
	ValidationStatusManual   ValidationStatus = "manual_validated" // Validada manualmente
	ValidationStatusRejected ValidationStatus = "rejected"         // Rechazada
)

// Transaction representa una transacción financiera
type Transaction struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relaciones
	UserID        uint  `json:"user_id" gorm:"not null;index"`
	AccountID     uint  `json:"account_id" gorm:"not null;index"` // Cuenta origen
	BankAccountID *uint `json:"bank_account_id" gorm:"index"`     // Referencia a cuenta bancaria (nullable)

	// Para transferencias
	ToAccountID *uint `json:"to_account_id" gorm:"index"` // Cuenta destino (para transferencias)

	// Información de la transacción
	Type        TransactionType   `json:"type" gorm:"not null" validate:"required,oneof=income expense transfer"`
	Status      TransactionStatus `json:"status" gorm:"default:'completed'" validate:"oneof=pending completed cancelled"`
	Amount      float64           `json:"amount" gorm:"not null;type:decimal(15,2)" validate:"required,gt=0"`
	Description string            `json:"description" gorm:"not null" validate:"required,min=1,max=500"`

	// Categorización
	CategoryID   *uint  `json:"category_id" gorm:"index"`
	CategoryName string `json:"category_name" validate:"max=100"` // Desnormalizado para performance
	Tags         string `json:"tags"`                             // JSON array de tags

	// Fecha y ubicación
	TransactionDate time.Time `json:"transaction_date" gorm:"not null;index"`
	Location        string    `json:"location" validate:"max=200"`

	// Información adicional
	Reference   string `json:"reference" validate:"max=100"`   // Número de referencia
	Notes       string `json:"notes" validate:"max=1000"`      // Notas adicionales
	Recurring   bool   `json:"recurring" gorm:"default:false"` // Si es una transacción recurrente
	RecurringID *uint  `json:"recurring_id" gorm:"index"`      // ID del patrón recurrente

	// Moneda (normalmente heredada de la cuenta)
	Currency     string  `json:"currency" gorm:"default:'MXN'" validate:"len=3"`
	ExchangeRate float64 `json:"exchange_rate" gorm:"default:1;type:decimal(10,6)"` // Para conversiones

	// Origen y validación de la transacción
	Source           TransactionSource `json:"source" gorm:"default:'manual'" validate:"oneof=notification manual integration import"`
	ValidationStatus ValidationStatus  `json:"validation_status" gorm:"default:'auto'" validate:"oneof=auto pending_review manual_validated rejected"`
	RawNotification  string            `json:"raw_notification" gorm:"type:text"`                // Notificación original (para transacciones desde notificación)
	AIConfidence     float64           `json:"ai_confidence" gorm:"default:0;type:decimal(3,2)"` // Confianza del AI (0-1)
	PatternID        *uint             `json:"pattern_id" gorm:"index"`                          // ID del patrón que procesó la notificación

	// Metadatos
	ImportedFrom string `json:"imported_from" validate:"max=100"` // Fuente de importación
	ExternalID   string `json:"external_id" validate:"max=100"`   // ID externo
}

// GetTags convierte el campo Tags (JSON string) a slice de strings
func (t *Transaction) GetTags() []string {
	if t.Tags == "" {
		return []string{}
	}

	var tags []string
	if err := json.Unmarshal([]byte(t.Tags), &tags); err != nil {
		return []string{}
	}

	return tags
}

// SetTags convierte un slice de strings a JSON string para el campo Tags
func (t *Transaction) SetTags(tags []string) error {
	if len(tags) == 0 {
		t.Tags = ""
		return nil
	}

	jsonTags, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	t.Tags = string(jsonTags)
	return nil
}

// IsIncome verifica si la transacción es un ingreso
func (t *Transaction) IsIncome() bool {
	return t.Type == TransactionTypeIncome
}

// IsExpense verifica si la transacción es un gasto
func (t *Transaction) IsExpense() bool {
	return t.Type == TransactionTypeExpense
}

// IsTransfer verifica si la transacción es una transferencia
func (t *Transaction) IsTransfer() bool {
	return t.Type == TransactionTypeTransfer
}

// IsCompleted verifica si la transacción está completada
func (t *Transaction) IsCompleted() bool {
	return t.Status == TransactionStatusCompleted
}

// IsPending verifica si la transacción está pendiente
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// IsCancelled verifica si la transacción está cancelada
func (t *Transaction) IsCancelled() bool {
	return t.Status == TransactionStatusCancelled
}

// GetSignedAmount retorna el monto con signo apropiado para balances
func (t *Transaction) GetSignedAmount() float64 {
	switch t.Type {
	case TransactionTypeIncome:
		return t.Amount
	case TransactionTypeExpense:
		return -t.Amount
	case TransactionTypeTransfer:
		// Para transferencias, el signo depende de la perspectiva de la cuenta
		return -t.Amount // Por defecto negativo para la cuenta origen
	default:
		return t.Amount
	}
}

// GetSignedAmountForAccount retorna el monto con signo apropiado para una cuenta específica
func (t *Transaction) GetSignedAmountForAccount(accountID uint) float64 {
	switch t.Type {
	case TransactionTypeIncome:
		return t.Amount
	case TransactionTypeExpense:
		return -t.Amount
	case TransactionTypeTransfer:
		if t.AccountID == accountID {
			return -t.Amount // Cuenta origen: negativo
		} else if t.ToAccountID != nil && *t.ToAccountID == accountID {
			return t.Amount // Cuenta destino: positivo
		}
		return 0
	default:
		return t.Amount
	}
}

// CanBeModified verifica si la transacción puede ser modificada
func (t *Transaction) CanBeModified() bool {
	return t.Status == TransactionStatusPending || t.Status == TransactionStatusCompleted
}

// CanBeCancelled verifica si la transacción puede ser cancelada
func (t *Transaction) CanBeCancelled() bool {
	return t.Status == TransactionStatusPending
}

// Complete marca la transacción como completada
func (t *Transaction) Complete() {
	t.Status = TransactionStatusCompleted
}

// Cancel marca la transacción como cancelada
func (t *Transaction) Cancel() {
	if t.CanBeCancelled() {
		t.Status = TransactionStatusCancelled
	}
}

// GetAmountInBaseCurrency retorna el monto en la moneda base usando el tipo de cambio
func (t *Transaction) GetAmountInBaseCurrency(baseCurrency string) float64 {
	if t.Currency == baseCurrency {
		return t.Amount
	}

	// Si no es la moneda base, aplicar tipo de cambio
	return t.Amount * t.ExchangeRate
}

// IsFromNotification verifica si la transacción proviene de una notificación
func (t *Transaction) IsFromNotification() bool {
	return t.Source == TransactionSourceNotification
}

// IsManual verifica si la transacción fue ingresada manualmente
func (t *Transaction) IsManual() bool {
	return t.Source == TransactionSourceManual
}

// IsFromIntegration verifica si la transacción proviene de integración bancaria
func (t *Transaction) IsFromIntegration() bool {
	return t.Source == TransactionSourceIntegration
}

// IsAutoValidated verifica si la transacción fue validada automáticamente
func (t *Transaction) IsAutoValidated() bool {
	return t.ValidationStatus == ValidationStatusAuto
}

// IsPendingReview verifica si la transacción está pendiente de revisión
func (t *Transaction) IsPendingReview() bool {
	return t.ValidationStatus == ValidationStatusPending
}

// IsManuallyValidated verifica si la transacción fue validada manualmente
func (t *Transaction) IsManuallyValidated() bool {
	return t.ValidationStatus == ValidationStatusManual
}

// IsRejected verifica si la transacción fue rechazada
func (t *Transaction) IsRejected() bool {
	return t.ValidationStatus == ValidationStatusRejected
}

// NeedsReview verifica si la transacción necesita revisión manual
func (t *Transaction) NeedsReview() bool {
	return t.IsPendingReview() || (t.IsFromNotification() && t.AIConfidence < 0.8)
}

// HasHighConfidence verifica si la transacción tiene alta confianza del AI
func (t *Transaction) HasHighConfidence() bool {
	return t.AIConfidence >= 0.8
}

// Approve marca la transacción como validada manualmente
func (t *Transaction) Approve() {
	t.ValidationStatus = ValidationStatusManual
	if t.Status == TransactionStatusPending {
		t.Status = TransactionStatusCompleted
	}
}

// Reject marca la transacción como rechazada
func (t *Transaction) Reject() {
	t.ValidationStatus = ValidationStatusRejected
	t.Status = TransactionStatusCancelled
}

// SetAIConfidence establece la confianza del AI y ajusta el estado de validación
func (t *Transaction) SetAIConfidence(confidence float64) {
	t.AIConfidence = confidence

	// Si tiene alta confianza y es de notificación, auto-validar
	if t.IsFromNotification() && confidence >= 0.9 {
		t.ValidationStatus = ValidationStatusAuto
		t.Status = TransactionStatusCompleted
	} else if t.IsFromNotification() && confidence < 0.7 {
		t.ValidationStatus = ValidationStatusPending
	}
}

// TransactionSummary representa un resumen de la transacción
type TransactionSummary struct {
	ID               uint              `json:"id"`
	Type             TransactionType   `json:"type"`
	Status           TransactionStatus `json:"status"`
	Amount           float64           `json:"amount"`
	Description      string            `json:"description"`
	CategoryName     string            `json:"category_name"`
	TransactionDate  time.Time         `json:"transaction_date"`
	AccountName      string            `json:"account_name"`
	ToAccountName    string            `json:"to_account_name,omitempty"`
	BankAccountAlias string            `json:"bank_account_alias,omitempty"`
	Currency         string            `json:"currency"`
	Source           TransactionSource `json:"source"`
	ValidationStatus ValidationStatus  `json:"validation_status"`
	AIConfidence     float64           `json:"ai_confidence"`
	NeedsReview      bool              `json:"needs_review"`
	CreatedAt        time.Time         `json:"created_at"`
}

// TransactionFilter representa filtros para búsqueda de transacciones
type TransactionFilter struct {
	AccountID        *uint              `json:"account_id"`
	BankAccountID    *uint              `json:"bank_account_id"`
	Type             *TransactionType   `json:"type"`
	Status           *TransactionStatus `json:"status"`
	Source           *TransactionSource `json:"source"`
	ValidationStatus *ValidationStatus  `json:"validation_status"`
	CategoryID       *uint              `json:"category_id"`
	FromDate         *time.Time         `json:"from_date"`
	ToDate           *time.Time         `json:"to_date"`
	MinAmount        *float64           `json:"min_amount"`
	MaxAmount        *float64           `json:"max_amount"`
	MinConfidence    *float64           `json:"min_confidence"`
	MaxConfidence    *float64           `json:"max_confidence"`
	NeedsReview      *bool              `json:"needs_review"`
	Search           string             `json:"search"` // Búsqueda en descripción
	Limit            int                `json:"limit"`
	Offset           int                `json:"offset"`
	OrderBy          string             `json:"order_by"`  // Campo por el cual ordenar
	OrderDir         string             `json:"order_dir"` // ASC o DESC
}
