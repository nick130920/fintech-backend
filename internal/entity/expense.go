package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ExpenseSource define la fuente del gasto
type ExpenseSource string

const (
	ExpenseSourceManual       ExpenseSource = "manual"       // Entrada manual del usuario
	ExpenseSourceSMS          ExpenseSource = "sms"          // Capturado desde SMS bancario
	ExpenseSourceWhatsApp     ExpenseSource = "whatsapp"     // Enviado por WhatsApp
	ExpenseSourceBankAPI      ExpenseSource = "bank_api"     // API bancaria
	ExpenseSourceNotification ExpenseSource = "notification" // Notificación del sistema
)

// ExpenseStatus define el estado del gasto
type ExpenseStatus string

const (
	ExpenseStatusPending   ExpenseStatus = "pending"   // Pendiente de confirmación
	ExpenseStatusConfirmed ExpenseStatus = "confirmed" // Confirmado
	ExpenseStatusCancelled ExpenseStatus = "cancelled" // Cancelado
)

// Expense representa un gasto registrado
type Expense struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relaciones
	UserID       uint `json:"user_id" gorm:"not null;index"`
	BudgetID     uint `json:"budget_id" gorm:"not null;index"`
	CategoryID   uint `json:"category_id" gorm:"not null;index"`
	AllocationID uint `json:"allocation_id" gorm:"not null;index"`

	// Información del gasto
	Amount      float64       `json:"amount" gorm:"not null;type:decimal(15,2)" validate:"required,gt=0"`
	Description string        `json:"description" gorm:"not null" validate:"required,min=1,max=500"`
	Date        time.Time     `json:"date" gorm:"not null;index"`
	Source      ExpenseSource `json:"source" gorm:"not null" validate:"required"`
	Status      ExpenseStatus `json:"status" gorm:"default:'confirmed'" validate:"oneof=pending confirmed cancelled"`

	// Ubicación y contexto
	Location  string `json:"location" validate:"max=200"`
	Merchant  string `json:"merchant" validate:"max=100"`  // Comercio donde se realizó el gasto
	Reference string `json:"reference" validate:"max=100"` // Referencia bancaria o número de transacción

	// Metadatos para captura automática
	RawData    string  `json:"raw_data"`   // Datos originales (SMS, JSON, etc.)
	Confidence float64 `json:"confidence"` // Confianza en la clasificación automática (0-1)
	Tags       string  `json:"tags"`       // JSON array de tags

	// Información adicional
	Notes        string  `json:"notes" validate:"max=1000"`
	ReceiptURL   string  `json:"receipt_url" validate:"url"` // URL del comprobante
	ExchangeRate float64 `json:"exchange_rate" gorm:"default:1;type:decimal(10,6)"`
	Currency     string  `json:"currency" gorm:"default:'MXN'" validate:"len=3"`

	// Control de alertas
	TriggeredAlert bool `json:"triggered_alert" gorm:"default:false"` // Si disparó una alerta
	AlertSent      bool `json:"alert_sent" gorm:"default:false"`      // Si se envió la alerta

	// Relaciones
	User       User             `json:"user" gorm:"foreignKey:UserID"`
	Budget     Budget           `json:"budget" gorm:"foreignKey:BudgetID"`
	Category   Category         `json:"category" gorm:"foreignKey:CategoryID"`
	Allocation BudgetAllocation `json:"allocation" gorm:"foreignKey:AllocationID"`
}

// GetTags convierte el campo Tags (JSON string) a slice de strings
func (e *Expense) GetTags() []string {
	if e.Tags == "" {
		return []string{}
	}

	var tags []string
	if err := json.Unmarshal([]byte(e.Tags), &tags); err != nil {
		return []string{}
	}

	return tags
}

// SetTags convierte un slice de strings a JSON string para el campo Tags
func (e *Expense) SetTags(tags []string) error {
	if len(tags) == 0 {
		e.Tags = ""
		return nil
	}

	jsonTags, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	e.Tags = string(jsonTags)
	return nil
}

// IsToday verifica si el gasto es del día de hoy
func (e *Expense) IsToday() bool {
	now := time.Now()
	return e.Date.Year() == now.Year() && e.Date.YearDay() == now.YearDay()
}

// IsThisMonth verifica si el gasto es del mes actual
func (e *Expense) IsThisMonth() bool {
	now := time.Now()
	return e.Date.Year() == now.Year() && e.Date.Month() == now.Month()
}

// IsConfirmed verifica si el gasto está confirmado
func (e *Expense) IsConfirmed() bool {
	return e.Status == ExpenseStatusConfirmed
}

// IsPending verifica si el gasto está pendiente
func (e *Expense) IsPending() bool {
	return e.Status == ExpenseStatusPending
}

// IsCancelled verifica si el gasto está cancelado
func (e *Expense) IsCancelled() bool {
	return e.Status == ExpenseStatusCancelled
}

// IsAutomatic verifica si el gasto fue capturado automáticamente
func (e *Expense) IsAutomatic() bool {
	return e.Source == ExpenseSourceSMS ||
		e.Source == ExpenseSourceBankAPI ||
		e.Source == ExpenseSourceNotification
}

// IsManual verifica si el gasto fue ingresado manualmente
func (e *Expense) IsManual() bool {
	return e.Source == ExpenseSourceManual || e.Source == ExpenseSourceWhatsApp
}

// Confirm confirma un gasto pendiente
func (e *Expense) Confirm() {
	e.Status = ExpenseStatusConfirmed
}

// Cancel cancela un gasto
func (e *Expense) Cancel() {
	e.Status = ExpenseStatusCancelled
}

// GetFormattedAmount retorna el monto formateado con moneda
func (e *Expense) GetFormattedAmount() string {
	return FormatCurrency(e.Amount, e.Currency)
}

// GetAmountInBaseCurrency retorna el monto en la moneda base
func (e *Expense) GetAmountInBaseCurrency(baseCurrency string) float64 {
	if e.Currency == baseCurrency {
		return e.Amount
	}

	return e.Amount * e.ExchangeRate
}

// ShouldTriggerAlert verifica si debe disparar una alerta
func (e *Expense) ShouldTriggerAlert(allocation *BudgetAllocation) bool {
	if e.TriggeredAlert || !e.IsConfirmed() {
		return false
	}

	// Verificar si con este gasto se excede el umbral de alerta
	newSpent := allocation.SpentAmount + e.Amount
	percentage := newSpent / allocation.AllocatedAmount

	return percentage >= allocation.AlertThreshold
}

// CanBeModified verifica si el gasto puede ser modificado
func (e *Expense) CanBeModified() bool {
	// Los gastos automáticos no se pueden modificar después de confirmados
	if e.IsAutomatic() && e.IsConfirmed() {
		return false
	}

	// Los gastos cancelados no se pueden modificar
	if e.IsCancelled() {
		return false
	}

	return true
}

// CanBeCancelled verifica si el gasto puede ser cancelado
func (e *Expense) CanBeCancelled() bool {
	return e.Status == ExpenseStatusPending || e.Status == ExpenseStatusConfirmed
}

// GetTimeAgo retorna una descripción de cuánto tiempo hace que se registró
func (e *Expense) GetTimeAgo() string {
	duration := time.Since(e.CreatedAt)

	if duration.Hours() < 1 {
		return "Hace unos minutos"
	} else if duration.Hours() < 24 {
		hours := int(duration.Hours())
		if hours == 1 {
			return "Hace 1 hora"
		}
		return fmt.Sprintf("Hace %d horas", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "Hace 1 día"
		}
		return fmt.Sprintf("Hace %d días", days)
	}
}

// ExpenseSummary representa un resumen de gasto para listas
type ExpenseSummary struct {
	ID           uint          `json:"id"`
	Amount       float64       `json:"amount"`
	Description  string        `json:"description"`
	Date         time.Time     `json:"date"`
	CategoryName string        `json:"category_name"`
	CategoryIcon string        `json:"category_icon"`
	Source       ExpenseSource `json:"source"`
	Status       ExpenseStatus `json:"status"`
	Currency     string        `json:"currency"`
	CreatedAt    time.Time     `json:"created_at"`
}

// ExpenseFilter representa filtros para búsqueda de gastos
type ExpenseFilter struct {
	CategoryID *uint          `json:"category_id"`
	Source     *ExpenseSource `json:"source"`
	Status     *ExpenseStatus `json:"status"`
	FromDate   *time.Time     `json:"from_date"`
	ToDate     *time.Time     `json:"to_date"`
	MinAmount  *float64       `json:"min_amount"`
	MaxAmount  *float64       `json:"max_amount"`
	Search     string         `json:"search"` // Búsqueda en descripción
	Merchant   string         `json:"merchant"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	OrderBy    string         `json:"order_by"`  // Campo por el cual ordenar
	OrderDir   string         `json:"order_dir"` // ASC o DESC
}

// FormatCurrency formatea una cantidad con la moneda especificada
func FormatCurrency(amount float64, currency string) string {
	// TODO: Implementar formateo según la moneda
	switch currency {
	case "MXN":
		return fmt.Sprintf("$%.2f MXN", amount)
	case "USD":
		return fmt.Sprintf("$%.2f USD", amount)
	default:
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}
