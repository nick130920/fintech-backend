package entity

import (
	"time"

	"gorm.io/gorm"
)

// BankAccountType define los tipos de cuenta bancaria
type BankAccountType string

const (
	BankAccountTypeChecking   BankAccountType = "checking"   // Cuenta corriente
	BankAccountTypeSavings    BankAccountType = "savings"    // Cuenta de ahorros
	BankAccountTypeCredit     BankAccountType = "credit"     // Tarjeta de crédito
	BankAccountTypeDebit      BankAccountType = "debit"      // Tarjeta de débito
	BankAccountTypeInvestment BankAccountType = "investment" // Cuenta de inversión
)

// BankAccount representa una cuenta bancaria real asociada a un usuario
type BankAccount struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relación con usuario
	UserID uint `json:"user_id" gorm:"not null;index"`

	// Información del banco
	BankName   string `json:"bank_name" gorm:"not null" validate:"required,min=1,max=100"`
	BankCode   string `json:"bank_code" validate:"max=10"`    // Código del banco (ej: 012 para BBVA)
	BranchCode string `json:"branch_code" validate:"max=10"`  // Código de sucursal
	BranchName string `json:"branch_name" validate:"max=100"` // Nombre de sucursal

	// Información de la cuenta
	AccountNumber     string          `json:"account_number" validate:"max=50"`                                     // Número completo (encriptado)
	AccountNumberMask string          `json:"account_number_mask" gorm:"not null" validate:"required,min=4,max=20"` // Últimos dígitos visibles (ej: ****1234)
	AccountAlias      string          `json:"account_alias" gorm:"not null" validate:"required,min=1,max=100"`      // Alias personalizado
	Type              BankAccountType `json:"type" gorm:"not null" validate:"required,oneof=checking savings credit debit investment"`

	// Información de identificación visual
	Color string `json:"color" gorm:"default:'#007bff'" validate:"hexcolor"`
	Icon  string `json:"icon" gorm:"default:'credit_card'" validate:"max=50"`

	// Estado y configuración
	IsActive              bool `json:"is_active" gorm:"default:true"`
	IsNotificationEnabled bool `json:"is_notification_enabled" gorm:"default:true"` // Si acepta notificaciones de esta cuenta

	// Información adicional
	Currency          string    `json:"currency" gorm:"default:'MXN'" validate:"len=3"`
	LastBalance       float64   `json:"last_balance" gorm:"type:decimal(15,2)"` // Último balance conocido
	LastBalanceUpdate time.Time `json:"last_balance_update"`                    // Fecha de última actualización de balance

	// Configuración de notificaciones
	NotificationPhone string  `json:"notification_phone" validate:"omitempty,min=10,max=15"`    // Teléfono para SMS
	NotificationEmail string  `json:"notification_email" validate:"omitempty,email"`            // Email para notificaciones
	MinAmountToNotify float64 `json:"min_amount_to_notify" gorm:"default:0;type:decimal(15,2)"` // Monto mínimo para notificar

	// Metadatos
	Notes        string `json:"notes" validate:"max=1000"`        // Notas adicionales
	ExternalID   string `json:"external_id" validate:"max=100"`   // ID externo del banco
	ImportedFrom string `json:"imported_from" validate:"max=100"` // Fuente de importación
}

// GetDisplayName retorna el nombre de visualización de la cuenta
func (ba *BankAccount) GetDisplayName() string {
	return ba.AccountAlias + " (" + ba.BankName + ")"
}

// GetMaskedAccountNumber retorna el número de cuenta enmascarado
func (ba *BankAccount) GetMaskedAccountNumber() string {
	if ba.AccountNumberMask != "" {
		return ba.AccountNumberMask
	}
	return "****"
}

// IsCredit verifica si la cuenta es de tipo crédito
func (ba *BankAccount) IsCredit() bool {
	return ba.Type == BankAccountTypeCredit
}

// IsDebit verifica si la cuenta es de tipo débito
func (ba *BankAccount) IsDebit() bool {
	return ba.Type == BankAccountTypeDebit
}

// CanReceiveNotifications verifica si la cuenta puede recibir notificaciones
func (ba *BankAccount) CanReceiveNotifications() bool {
	return ba.IsActive && ba.IsNotificationEnabled
}

// ShouldNotifyAmount verifica si un monto debe generar notificación
func (ba *BankAccount) ShouldNotifyAmount(amount float64) bool {
	if !ba.CanReceiveNotifications() {
		return false
	}
	return amount >= ba.MinAmountToNotify
}

// UpdateBalance actualiza el balance y la fecha de actualización
func (ba *BankAccount) UpdateBalance(newBalance float64) {
	ba.LastBalance = newBalance
	ba.LastBalanceUpdate = time.Now()
}

// GetShortBankName retorna una versión corta del nombre del banco
func (ba *BankAccount) GetShortBankName() string {
	switch ba.BankName {
	case "Banco Bilbao Vizcaya Argentaria":
		return "BBVA"
	default:
		return ba.BankName
	}
}

// BankAccountSummary representa un resumen de la cuenta bancaria
type BankAccountSummary struct {
	ID                uint            `json:"id"`
	BankName          string          `json:"bank_name"`
	ShortBankName     string          `json:"short_bank_name"`
	AccountAlias      string          `json:"account_alias"`
	AccountNumberMask string          `json:"account_number_mask"`
	Type              BankAccountType `json:"type"`
	Color             string          `json:"color"`
	Icon              string          `json:"icon"`
	IsActive          bool            `json:"is_active"`
	Currency          string          `json:"currency"`
	LastBalance       float64         `json:"last_balance"`
	LastBalanceUpdate time.Time       `json:"last_balance_update"`
	DisplayName       string          `json:"display_name"`
}

// ToSummary convierte una BankAccount a BankAccountSummary
func (ba *BankAccount) ToSummary() BankAccountSummary {
	return BankAccountSummary{
		ID:                ba.ID,
		BankName:          ba.BankName,
		ShortBankName:     ba.GetShortBankName(),
		AccountAlias:      ba.AccountAlias,
		AccountNumberMask: ba.GetMaskedAccountNumber(),
		Type:              ba.Type,
		Color:             ba.Color,
		Icon:              ba.Icon,
		IsActive:          ba.IsActive,
		Currency:          ba.Currency,
		LastBalance:       ba.LastBalance,
		LastBalanceUpdate: ba.LastBalanceUpdate,
		DisplayName:       ba.GetDisplayName(),
	}
}

// BankAccountFilter representa filtros para búsqueda de cuentas bancarias
type BankAccountFilter struct {
	BankName string           `json:"bank_name"`
	Type     *BankAccountType `json:"type"`
	IsActive *bool            `json:"is_active"`
	Currency string           `json:"currency"`
	Search   string           `json:"search"` // Búsqueda en alias o banco
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
	OrderBy  string           `json:"order_by"`  // Campo por el cual ordenar
	OrderDir string           `json:"order_dir"` // ASC o DESC
}
