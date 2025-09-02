package dto

import "github.com/nick130920/fintech-backend/internal/entity"

// CreateAccountRequest representa la estructura para crear una cuenta
type CreateAccountRequest struct {
	Name           string             `json:"name" validate:"required,min=1,max=100"`
	Description    string             `json:"description" validate:"max=500"`
	Type           entity.AccountType `json:"type" validate:"required,oneof=checking savings credit investment cash"`
	InitialBalance float64            `json:"initial_balance" validate:"gte=0"`
	CreditLimit    float64            `json:"credit_limit" validate:"gte=0"`
	BankName       string             `json:"bank_name" validate:"max=100"`
	AccountNumber  string             `json:"account_number" validate:"max=50"`
	Currency       string             `json:"currency" validate:"omitempty,len=3"`
	Color          string             `json:"color" validate:"omitempty,hexcolor"`
	Icon           string             `json:"icon" validate:"max=50"`
}

// UpdateAccountRequest representa la estructura para actualizar una cuenta
type UpdateAccountRequest struct {
	Name        string             `json:"name" validate:"omitempty,min=1,max=100"`
	Description string             `json:"description" validate:"max=500"`
	Type        entity.AccountType `json:"type" validate:"omitempty,oneof=checking savings credit investment cash"`
	CreditLimit float64            `json:"credit_limit" validate:"gte=0"`
	BankName    string             `json:"bank_name" validate:"max=100"`
	Color       string             `json:"color" validate:"omitempty,hexcolor"`
	Icon        string             `json:"icon" validate:"max=50"`
	IsActive    *bool              `json:"is_active"`

	// Configuración de alertas
	LowBalanceAlert *bool   `json:"low_balance_alert"`
	LowBalanceLimit float64 `json:"low_balance_limit" validate:"gte=0"`
}
