package dto

import "time"

// CreateTripRequest representa la estructura para crear un viaje
type CreateTripRequest struct {
	Name            string    `json:"name" validate:"required,min=1,max=120"`
	Destination     string    `json:"destination" validate:"required,min=1,max=200"`
	CountryCode     string    `json:"country_code" validate:"omitempty,len=2"`
	StartDate       time.Time `json:"start_date" validate:"required"`
	EndDate         time.Time `json:"end_date" validate:"required,gtefield=StartDate"`
	PrimaryCurrency string    `json:"primary_currency" validate:"required,len=3"`
	CoverImageURL   string    `json:"cover_image_url" validate:"omitempty,url"`
	Notes           string    `json:"notes" validate:"max=2000"`
}

// UpdateTripRequest permite modificar campos editables de un viaje
type UpdateTripRequest struct {
	Name            *string    `json:"name" validate:"omitempty,min=1,max=120"`
	Destination     *string    `json:"destination" validate:"omitempty,min=1,max=200"`
	CountryCode     *string    `json:"country_code" validate:"omitempty,len=2"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	PrimaryCurrency *string    `json:"primary_currency" validate:"omitempty,len=3"`
	CoverImageURL   *string    `json:"cover_image_url" validate:"omitempty,url"`
	Notes           *string    `json:"notes" validate:"omitempty,max=2000"`
}

// TripResponse representa la respuesta resumida del viaje
type TripResponse struct {
	ID                uint                       `json:"id"`
	OwnerUserID       uint                       `json:"owner_user_id"`
	Name              string                     `json:"name"`
	Destination       string                     `json:"destination"`
	CountryCode       string                     `json:"country_code,omitempty"`
	StartDate         time.Time                  `json:"start_date"`
	EndDate           time.Time                  `json:"end_date"`
	PrimaryCurrency   string                     `json:"primary_currency"`
	Status            string                     `json:"status"`
	CoverImageURL     string                     `json:"cover_image_url,omitempty"`
	EstimatedTotal    float64                    `json:"estimated_total"`
	SpentTotal        float64                    `json:"spent_total"`
	RemainingAmount   float64                    `json:"remaining_amount"`
	ProgressPercent   float64                    `json:"progress_percent"`
	DaysTotal         int                        `json:"days_total"`
	DaysRemaining     int                        `json:"days_remaining"`
	IsActiveNow       bool                       `json:"is_active_now"`
	Notes             string                     `json:"notes,omitempty"`
	Members           []TripMemberResponse       `json:"members,omitempty"`
	Allocations       []TripAllocationResponse   `json:"allocations,omitempty"`
	Itinerary         []TripItineraryResponse    `json:"itinerary,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
}

// AddTripMemberRequest agrega un miembro fantasma al viaje
type AddTripMemberRequest struct {
	DisplayName string  `json:"display_name" validate:"required,min=1,max=120"`
	Email       *string `json:"email" validate:"omitempty,email"`
	AvatarURL   string  `json:"avatar_url" validate:"omitempty,url"`
	Role        string  `json:"role" validate:"omitempty,oneof=admin member viewer"`
}

// UpdateTripMemberRequest actualiza datos de un miembro
type UpdateTripMemberRequest struct {
	DisplayName *string `json:"display_name" validate:"omitempty,min=1,max=120"`
	AvatarURL   *string `json:"avatar_url" validate:"omitempty,url"`
	Role        *string `json:"role" validate:"omitempty,oneof=admin member viewer"`
}

// TripMemberResponse representa la información pública de un miembro
type TripMemberResponse struct {
	ID          uint      `json:"id"`
	UserID      *uint     `json:"user_id,omitempty"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Role        string    `json:"role"`
	IsGhost     bool      `json:"is_ghost"`
	JoinedAt    time.Time `json:"joined_at"`
}

// CreateInvitationRequest genera un link de invitación
type CreateInvitationRequest struct {
	Email        string `json:"email" validate:"omitempty,email"`
	Role         string `json:"role" validate:"omitempty,oneof=admin member viewer"`
	ExpiresInDay int    `json:"expires_in_days" validate:"omitempty,min=1,max=30"`
}

// InvitationResponse representa una invitación generada
type InvitationResponse struct {
	ID        uint      `json:"id"`
	TripID    uint      `json:"trip_id"`
	Token     string    `json:"token"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AcceptInvitationRequest acepta una invitación por token
type AcceptInvitationRequest struct {
	Token string `json:"token" validate:"required"`
}

// UpsertTripBudgetRequest crea o reemplaza el presupuesto del viaje
type UpsertTripBudgetRequest struct {
	Allocations []TripAllocationRequest `json:"allocations" validate:"required,min=1,dive"`
}

// TripAllocationRequest representa una asignación a categoría dentro del viaje
type TripAllocationRequest struct {
	CategoryID      uint    `json:"category_id" validate:"required"`
	EstimatedAmount float64 `json:"estimated_amount" validate:"gte=0"`
	Currency        string  `json:"currency" validate:"omitempty,len=3"`
	Notes           string  `json:"notes" validate:"max=500"`
}

// TripAllocationResponse representa la asignación de presupuesto en respuesta
type TripAllocationResponse struct {
	ID              uint                    `json:"id"`
	Category        CategorySummaryResponse `json:"category"`
	EstimatedAmount float64                 `json:"estimated_amount"`
	SpentAmount     float64                 `json:"spent_amount"`
	RemainingAmount float64                 `json:"remaining_amount"`
	ProgressPercent float64                 `json:"progress_percent"`
	DailySuggested  float64                 `json:"daily_suggested"`
	IsOverBudget    bool                    `json:"is_over_budget"`
	Currency        string                  `json:"currency"`
	Notes           string                  `json:"notes,omitempty"`
}

// CreateTripExpenseRequest crea un gasto del viaje con sus splits
type CreateTripExpenseRequest struct {
	CategoryID     uint                 `json:"category_id" validate:"required"`
	Amount         float64              `json:"amount" validate:"required,gt=0"`
	Description    string               `json:"description" validate:"required,min=1,max=500"`
	Date           time.Time            `json:"date" validate:"required"`
	Currency       string               `json:"currency" validate:"required,len=3"`
	ExchangeRate   float64              `json:"exchange_rate" validate:"omitempty,gt=0"`
	Location       string               `json:"location" validate:"max=200"`
	Merchant       string               `json:"merchant" validate:"max=100"`
	Notes          string               `json:"notes" validate:"max=1000"`
	ReceiptURL     string               `json:"receipt_url" validate:"omitempty,url"`
	PaidByMemberID uint                 `json:"paid_by_member_id" validate:"required"`
	Splits         []ExpenseSplitInput  `json:"splits" validate:"required,min=1,dive"`
}

// UpdateTripExpenseRequest actualiza un gasto del viaje
type UpdateTripExpenseRequest struct {
	CategoryID     *uint                `json:"category_id"`
	Amount         *float64             `json:"amount" validate:"omitempty,gt=0"`
	Description    *string              `json:"description" validate:"omitempty,min=1,max=500"`
	Date           *time.Time           `json:"date"`
	Currency       *string              `json:"currency" validate:"omitempty,len=3"`
	ExchangeRate   *float64             `json:"exchange_rate" validate:"omitempty,gt=0"`
	Location       *string              `json:"location" validate:"omitempty,max=200"`
	Merchant       *string              `json:"merchant" validate:"omitempty,max=100"`
	Notes          *string              `json:"notes" validate:"omitempty,max=1000"`
	ReceiptURL     *string              `json:"receipt_url" validate:"omitempty,url"`
	PaidByMemberID *uint                `json:"paid_by_member_id"`
	Splits         []ExpenseSplitInput  `json:"splits" validate:"omitempty,dive"`
}

// ExpenseSplitInput representa la división propuesta para un miembro
type ExpenseSplitInput struct {
	MemberID   uint    `json:"member_id" validate:"required"`
	ShareType  string  `json:"share_type" validate:"required,oneof=equal percentage exact shares"`
	ShareValue float64 `json:"share_value" validate:"gte=0"`
}

// TripExpenseResponse representa el gasto del viaje devuelto al cliente
type TripExpenseResponse struct {
	ID             uint                    `json:"id"`
	TripID         uint                    `json:"trip_id"`
	CategoryID     uint                    `json:"category_id"`
	Category       CategorySummaryResponse `json:"category"`
	Amount         float64                 `json:"amount"`
	AmountPrimary  float64                 `json:"amount_primary"`
	Currency       string                  `json:"currency"`
	ExchangeRate   float64                 `json:"exchange_rate"`
	Description    string                  `json:"description"`
	Date           time.Time               `json:"date"`
	Location       string                  `json:"location,omitempty"`
	Merchant       string                  `json:"merchant,omitempty"`
	Notes          string                  `json:"notes,omitempty"`
	ReceiptURL     string                  `json:"receipt_url,omitempty"`
	PaidByMemberID *uint                   `json:"paid_by_member_id,omitempty"`
	PaidByName     string                  `json:"paid_by_name,omitempty"`
	Splits         []TripExpenseSplit      `json:"splits"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

// TripExpenseSplit representa la división de un miembro en un gasto
type TripExpenseSplit struct {
	ID          uint    `json:"id"`
	MemberID    uint    `json:"member_id"`
	MemberName  string  `json:"member_name"`
	ShareType   string  `json:"share_type"`
	ShareValue  float64 `json:"share_value"`
	ShareAmount float64 `json:"share_amount"`
	IsPaid      bool    `json:"is_paid"`
}

// TripBalanceResponse representa el balance simplificado entre miembros
type TripBalanceResponse struct {
	TripID     uint                  `json:"trip_id"`
	Currency   string                `json:"currency"`
	NetByMember []TripMemberBalance   `json:"net_by_member"`
	Transfers  []TripBalanceTransfer `json:"transfers"`
}

// TripMemberBalance representa el saldo neto de un miembro
type TripMemberBalance struct {
	MemberID    uint    `json:"member_id"`
	MemberName  string  `json:"member_name"`
	NetAmount   float64 `json:"net_amount"`
}

// TripBalanceTransfer representa una transferencia sugerida para saldar deudas
type TripBalanceTransfer struct {
	FromMemberID uint    `json:"from_member_id"`
	FromName     string  `json:"from_name"`
	ToMemberID   uint    `json:"to_member_id"`
	ToName       string  `json:"to_name"`
	Amount       float64 `json:"amount"`
}

// CreateSettlementRequest registra un pago entre miembros
type CreateSettlementRequest struct {
	FromMemberID uint      `json:"from_member_id" validate:"required"`
	ToMemberID   uint      `json:"to_member_id" validate:"required,nefield=FromMemberID"`
	Amount       float64   `json:"amount" validate:"required,gt=0"`
	Currency     string    `json:"currency" validate:"required,len=3"`
	FxRate       float64   `json:"fx_rate" validate:"omitempty,gt=0"`
	PaidAt       time.Time `json:"paid_at"`
	Notes        string    `json:"notes" validate:"max=500"`
}

// SettlementResponse representa un pago registrado entre miembros
type SettlementResponse struct {
	ID           uint      `json:"id"`
	TripID       uint      `json:"trip_id"`
	FromMemberID uint      `json:"from_member_id"`
	FromName     string    `json:"from_name"`
	ToMemberID   uint      `json:"to_member_id"`
	ToName       string    `json:"to_name"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	FxRate       float64   `json:"fx_rate"`
	PaidAt       time.Time `json:"paid_at"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateItineraryItemRequest crea o actualiza un item del itinerario
type CreateItineraryItemRequest struct {
	Day           time.Time `json:"day" validate:"required"`
	Time          string    `json:"time" validate:"omitempty,len=5"`
	Type          string    `json:"type" validate:"required,oneof=flight hotel transport activity food other"`
	Title         string    `json:"title" validate:"required,min=1,max=200"`
	Description   string    `json:"description" validate:"max=1000"`
	Location      string    `json:"location" validate:"max=200"`
	EstimatedCost float64   `json:"estimated_cost" validate:"gte=0"`
	Currency      string    `json:"currency" validate:"required,len=3"`
}

// UpdateItineraryItemRequest permite modificar un item
type UpdateItineraryItemRequest struct {
	Day           *time.Time `json:"day"`
	Time          *string    `json:"time" validate:"omitempty,len=5"`
	Type          *string    `json:"type" validate:"omitempty,oneof=flight hotel transport activity food other"`
	Title         *string    `json:"title" validate:"omitempty,min=1,max=200"`
	Description   *string    `json:"description" validate:"omitempty,max=1000"`
	Location      *string    `json:"location" validate:"omitempty,max=200"`
	EstimatedCost *float64   `json:"estimated_cost" validate:"omitempty,gte=0"`
	Currency      *string    `json:"currency" validate:"omitempty,len=3"`
}

// LinkItineraryExpenseRequest vincula un item del itinerario con un gasto real
type LinkItineraryExpenseRequest struct {
	ExpenseID uint `json:"expense_id" validate:"required"`
}

// TripItineraryResponse representa un item del itinerario
type TripItineraryResponse struct {
	ID            uint      `json:"id"`
	Day           time.Time `json:"day"`
	Time          string    `json:"time,omitempty"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Description   string    `json:"description,omitempty"`
	Location      string    `json:"location,omitempty"`
	EstimatedCost float64   `json:"estimated_cost"`
	Currency      string    `json:"currency"`
	ExpenseID     *uint     `json:"expense_id,omitempty"`
	ActualAmount  float64   `json:"actual_amount,omitempty"`
	Variance      float64   `json:"variance,omitempty"`
}

// ImportSuggestionResponse representa un gasto candidato para asignar al viaje
type ImportSuggestionResponse struct {
	ExpenseID   uint      `json:"expense_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	CategoryID  uint      `json:"category_id"`
	Merchant    string    `json:"merchant,omitempty"`
}

// AssignImportRequest acepta gastos candidatos para asignarlos al viaje
type AssignImportRequest struct {
	ExpenseIDs []uint `json:"expense_ids" validate:"required,min=1"`
}

// TripReportResponse representa el reporte final consolidado del viaje
type TripReportResponse struct {
	Trip               TripResponse                  `json:"trip"`
	TotalsByCategory   []TripReportCategoryTotal     `json:"totals_by_category"`
	TotalsByMember     []TripReportMemberTotal       `json:"totals_by_member"`
	EstimatedVsReal    TripReportEstimateVsReal      `json:"estimated_vs_real"`
	ItineraryProgress  []TripReportItineraryProgress `json:"itinerary_progress"`
	Settlements        []SettlementResponse          `json:"settlements"`
	GeneratedAt        time.Time                     `json:"generated_at"`
}

type TripReportCategoryTotal struct {
	CategoryID      uint    `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	EstimatedAmount float64 `json:"estimated_amount"`
	SpentAmount     float64 `json:"spent_amount"`
	Variance        float64 `json:"variance"`
}

type TripReportMemberTotal struct {
	MemberID   uint    `json:"member_id"`
	MemberName string  `json:"member_name"`
	Paid       float64 `json:"paid"`
	Owed       float64 `json:"owed"`
	Net        float64 `json:"net"`
}

type TripReportEstimateVsReal struct {
	EstimatedTotal float64 `json:"estimated_total"`
	SpentTotal     float64 `json:"spent_total"`
	Variance       float64 `json:"variance"`
	OverBudget     bool    `json:"over_budget"`
}

type TripReportItineraryProgress struct {
	ItemID        uint      `json:"item_id"`
	Title         string    `json:"title"`
	Day           time.Time `json:"day"`
	EstimatedCost float64   `json:"estimated_cost"`
	ActualCost    float64   `json:"actual_cost"`
	Variance      float64   `json:"variance"`
}
