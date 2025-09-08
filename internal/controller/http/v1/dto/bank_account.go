package dto

import (
	"time"

	"github.com/nick130920/fintech-backend/internal/entity"
)

// CreateBankAccountRequest representa la estructura para crear una cuenta bancaria
type CreateBankAccountRequest struct {
	BankName              string                 `json:"bank_name" validate:"required,min=1,max=100"`
	BankCode              string                 `json:"bank_code" validate:"omitempty,max=10"`
	BranchCode            string                 `json:"branch_code" validate:"omitempty,max=10"`
	BranchName            string                 `json:"branch_name" validate:"omitempty,max=100"`
	AccountNumber         string                 `json:"account_number" validate:"omitempty,max=50"` // Debería estar encriptado
	AccountNumberMask     string                 `json:"account_number_mask" validate:"required,min=4,max=20"`
	AccountAlias          string                 `json:"account_alias" validate:"required,min=1,max=100"`
	Type                  entity.BankAccountType `json:"type" validate:"required,oneof=checking savings credit debit investment"`
	Color                 string                 `json:"color" validate:"omitempty,hexcolor"`
	Icon                  string                 `json:"icon" validate:"omitempty,max=50"`
	IsNotificationEnabled bool                   `json:"is_notification_enabled"`
	Currency              string                 `json:"currency" validate:"omitempty,len=3"`
	NotificationPhone     string                 `json:"notification_phone" validate:"omitempty,min=10,max=15"`
	NotificationEmail     string                 `json:"notification_email" validate:"omitempty,email"`
	MinAmountToNotify     float64                `json:"min_amount_to_notify" validate:"omitempty,gte=0"`
	Notes                 string                 `json:"notes" validate:"omitempty,max=1000"`
}

// UpdateBankAccountRequest representa la estructura para actualizar una cuenta bancaria
type UpdateBankAccountRequest struct {
	BankName              string   `json:"bank_name" validate:"omitempty,min=1,max=100"`
	AccountAlias          string   `json:"account_alias" validate:"omitempty,min=1,max=100"`
	Color                 string   `json:"color" validate:"omitempty,hexcolor"`
	Icon                  string   `json:"icon" validate:"omitempty,max=50"`
	IsNotificationEnabled *bool    `json:"is_notification_enabled"`
	NotificationPhone     string   `json:"notification_phone" validate:"omitempty,min=10,max=15"`
	NotificationEmail     string   `json:"notification_email" validate:"omitempty,email"`
	MinAmountToNotify     *float64 `json:"min_amount_to_notify" validate:"omitempty,gte=0"`
	Notes                 string   `json:"notes" validate:"omitempty,max=1000"`
}

// SetBankAccountActiveRequest representa la estructura para cambiar el estado activo
type SetBankAccountActiveRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

// UpdateBankAccountBalanceRequest representa la estructura para actualizar el balance
type UpdateBankAccountBalanceRequest struct {
	Balance float64 `json:"balance" validate:"required"`
}

// BankAccountResponse representa la respuesta de una cuenta bancaria
type BankAccountResponse struct {
	ID                    uint                   `json:"id"`
	BankName              string                 `json:"bank_name"`
	BankCode              string                 `json:"bank_code"`
	BranchCode            string                 `json:"branch_code"`
	BranchName            string                 `json:"branch_name"`
	AccountNumberMask     string                 `json:"account_number_mask"`
	AccountAlias          string                 `json:"account_alias"`
	Type                  entity.BankAccountType `json:"type"`
	Color                 string                 `json:"color"`
	Icon                  string                 `json:"icon"`
	IsActive              bool                   `json:"is_active"`
	IsNotificationEnabled bool                   `json:"is_notification_enabled"`
	Currency              string                 `json:"currency"`
	LastBalance           float64                `json:"last_balance"`
	LastBalanceUpdate     time.Time              `json:"last_balance_update"`
	NotificationPhone     string                 `json:"notification_phone"`
	NotificationEmail     string                 `json:"notification_email"`
	MinAmountToNotify     float64                `json:"min_amount_to_notify"`
	Notes                 string                 `json:"notes"`
	DisplayName           string                 `json:"display_name"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// BankAccountSummaryResponse representa un resumen de cuenta bancaria
type BankAccountSummaryResponse struct {
	ID                uint                   `json:"id"`
	BankName          string                 `json:"bank_name"`
	ShortBankName     string                 `json:"short_bank_name"`
	AccountAlias      string                 `json:"account_alias"`
	AccountNumberMask string                 `json:"account_number_mask"`
	Type              entity.BankAccountType `json:"type"`
	Color             string                 `json:"color"`
	Icon              string                 `json:"icon"`
	IsActive          bool                   `json:"is_active"`
	Currency          string                 `json:"currency"`
	LastBalance       float64                `json:"last_balance"`
	LastBalanceUpdate time.Time              `json:"last_balance_update"`
	DisplayName       string                 `json:"display_name"`
}

// PaginatedBankAccountResponse representa una respuesta paginada de cuentas bancarias
type PaginatedBankAccountResponse struct {
	Data       []*BankAccountResponse `json:"data"`
	Total      int                    `json:"total"`
	Page       int                    `json:"page"`
	PerPage    int                    `json:"per_page"`
	TotalPages int                    `json:"total_pages"`
}
