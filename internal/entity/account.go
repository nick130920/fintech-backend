package entity

import (
	"time"

	"gorm.io/gorm"
)

// AccountType define los tipos de cuenta disponibles
type AccountType string

const (
	AccountTypeChecking   AccountType = "checking"   // Cuenta corriente
	AccountTypeSavings    AccountType = "savings"    // Cuenta de ahorros
	AccountTypeCredit     AccountType = "credit"     // Tarjeta de crédito
	AccountTypeInvestment AccountType = "investment" // Cuenta de inversión
	AccountTypeCash       AccountType = "cash"       // Efectivo
)

// Account representa una cuenta financiera
type Account struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relación con usuario
	UserID uint `json:"user_id" gorm:"not null;index"`

	// Información de la cuenta
	Name        string      `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Description string      `json:"description" validate:"max=500"`
	Type        AccountType `json:"type" gorm:"not null" validate:"required,oneof=checking savings credit investment cash"`

	// Información financiera
	Balance        float64 `json:"balance" gorm:"default:0;type:decimal(15,2)"`
	InitialBalance float64 `json:"initial_balance" gorm:"default:0;type:decimal(15,2)"`
	CreditLimit    float64 `json:"credit_limit" gorm:"default:0;type:decimal(15,2)"` // Para tarjetas de crédito

	// Información bancaria (opcional)
	BankName      string `json:"bank_name" validate:"max=100"`
	AccountNumber string `json:"account_number" validate:"max=50"`

	// Estado y configuración
	IsActive bool   `json:"is_active" gorm:"default:true"`
	Currency string `json:"currency" gorm:"default:'MXN'" validate:"len=3"`
	Color    string `json:"color" gorm:"default:'#007bff'" validate:"hexcolor"`
	Icon     string `json:"icon" validate:"max=50"`

	// Configuración de alertas
	LowBalanceAlert bool    `json:"low_balance_alert" gorm:"default:false"`
	LowBalanceLimit float64 `json:"low_balance_limit" gorm:"default:0;type:decimal(15,2)"`
}

// ToSummary convierte una Account a AccountSummary
func (a *Account) ToSummary() AccountSummary {
	summary := AccountSummary{
		ID:       a.ID,
		Name:     a.Name,
		Type:     a.Type,
		Balance:  a.Balance,
		Currency: a.Currency,
		Color:    a.Color,
		Icon:     a.Icon,
		IsActive: a.IsActive,
		BankName: a.BankName,
	}

	// Calcular crédito disponible para tarjetas de crédito
	if a.Type == AccountTypeCredit {
		summary.AvailableCredit = a.CreditLimit - a.Balance
	}

	return summary
}

// GetDisplayName retorna el nombre de la cuenta con información adicional
func (a *Account) GetDisplayName() string {
	if a.BankName != "" {
		return a.Name + " (" + a.BankName + ")"
	}
	return a.Name
}

// IsCredit verifica si la cuenta es de tipo crédito
func (a *Account) IsCredit() bool {
	return a.Type == AccountTypeCredit
}

// GetAvailableBalance retorna el balance disponible considerando límites de crédito
func (a *Account) GetAvailableBalance() float64 {
	if a.IsCredit() {
		return a.CreditLimit - a.Balance
	}
	return a.Balance
}

// CanDebit verifica si se puede debitar un monto de la cuenta
func (a *Account) CanDebit(amount float64) bool {
	if !a.IsActive {
		return false
	}

	available := a.GetAvailableBalance()
	return available >= amount
}

// Debit debita un monto de la cuenta
func (a *Account) Debit(amount float64) bool {
	if !a.CanDebit(amount) {
		return false
	}

	if a.IsCredit() {
		a.Balance += amount // Para crédito, aumentar deuda
	} else {
		a.Balance -= amount // Para otras cuentas, disminuir balance
	}

	return true
}

// Credit acredita un monto a la cuenta
func (a *Account) Credit(amount float64) {
	if a.IsCredit() {
		a.Balance -= amount // Para crédito, disminuir deuda
		if a.Balance < 0 {
			a.Balance = 0 // No permitir saldo positivo en tarjetas de crédito
		}
	} else {
		a.Balance += amount // Para otras cuentas, aumentar balance
	}
}

// ShouldAlert verifica si debe activarse una alerta de balance bajo
func (a *Account) ShouldAlert() bool {
	if !a.LowBalanceAlert || !a.IsActive {
		return false
	}

	if a.IsCredit() {
		return a.GetAvailableBalance() <= a.LowBalanceLimit
	}

	return a.Balance <= a.LowBalanceLimit
}

// AccountSummary representa un resumen de la cuenta
type AccountSummary struct {
	ID              uint        `json:"id"`
	Name            string      `json:"name"`
	Type            AccountType `json:"type"`
	Balance         float64     `json:"balance"`
	Currency        string      `json:"currency"`
	Color           string      `json:"color"`
	Icon            string      `json:"icon"`
	IsActive        bool        `json:"is_active"`
	BankName        string      `json:"bank_name"`
	AvailableCredit float64     `json:"available_credit,omitempty"` // Para tarjetas de crédito
}
