package dto

import "github.com/nick130920/proyecto-fintech/internal/entity"

// CreateTransactionRequest representa la estructura para crear una transacción
type CreateTransactionRequest struct {
	AccountID       uint                   `json:"account_id" validate:"required"`
	ToAccountID     *uint                  `json:"to_account_id"`
	Type            entity.TransactionType `json:"type" validate:"required,oneof=income expense transfer"`
	Amount          float64                `json:"amount" validate:"required,gt=0"`
	Description     string                 `json:"description" validate:"required,min=1,max=500"`
	CategoryID      *uint                  `json:"category_id"`
	Tags            []string               `json:"tags"`
	TransactionDate string                 `json:"transaction_date" validate:"required"`
	Location        string                 `json:"location" validate:"max=200"`
	Reference       string                 `json:"reference" validate:"max=100"`
	Notes           string                 `json:"notes" validate:"max=1000"`
	Currency        string                 `json:"currency" validate:"omitempty,len=3"`
}

// UpdateTransactionRequest representa la estructura para actualizar una transacción
type UpdateTransactionRequest struct {
	Description     string                   `json:"description" validate:"omitempty,min=1,max=500"`
	CategoryID      *uint                    `json:"category_id"`
	Tags            []string                 `json:"tags"`
	TransactionDate string                   `json:"transaction_date"`
	Location        string                   `json:"location" validate:"max=200"`
	Reference       string                   `json:"reference" validate:"max=100"`
	Notes           string                   `json:"notes" validate:"max=1000"`
	Status          entity.TransactionStatus `json:"status" validate:"omitempty,oneof=pending completed cancelled"`
}
