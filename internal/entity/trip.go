package entity

import (
	"time"

	"gorm.io/gorm"
)

// TripStatus define el estado de un viaje
type TripStatus string

const (
	TripStatusPlanning  TripStatus = "planning"
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
	TripStatusCancelled TripStatus = "cancelled"
)

// Trip representa un viaje de turismo donde se lleva presupuesto y gastos compartidos
type Trip struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Propietario del viaje
	OwnerUserID uint `json:"owner_user_id" gorm:"not null;index"`

	// Información básica
	Name        string `json:"name" gorm:"not null" validate:"required,min=1,max=120"`
	Destination string `json:"destination" gorm:"not null" validate:"required,min=1,max=200"`
	CountryCode string `json:"country_code" gorm:"type:varchar(3)" validate:"omitempty,len=2"`

	// Fechas del viaje
	StartDate time.Time `json:"start_date" gorm:"not null;index"`
	EndDate   time.Time `json:"end_date" gorm:"not null;index"`

	// Moneda principal del viaje (donde se efectúa la mayoría del gasto)
	PrimaryCurrency string `json:"primary_currency" gorm:"type:varchar(3);not null;default:'USD'" validate:"len=3"`

	// Snapshot de FX al crear el viaje (para reportes históricos estables)
	BaseCurrencyAtCreation string  `json:"base_currency_at_creation" gorm:"type:varchar(3);default:'USD'"`
	FxRateAtCreation       float64 `json:"fx_rate_at_creation" gorm:"default:1;type:decimal(18,8)"`

	// Estado del viaje
	Status TripStatus `json:"status" gorm:"type:varchar(20);default:'planning';index" validate:"oneof=planning active completed cancelled"`

	// Información visual
	CoverImageURL string `json:"cover_image_url" validate:"omitempty,url"`

	// Totales agregados (denormalizados para listas rápidas)
	EstimatedTotal float64 `json:"estimated_total" gorm:"default:0;type:decimal(15,2)"`
	SpentTotal     float64 `json:"spent_total" gorm:"default:0;type:decimal(15,2)"`

	// Notas libres
	Notes string `json:"notes" validate:"max=2000"`

	// Relaciones
	Members     []TripMember             `json:"members,omitempty" gorm:"foreignKey:TripID"`
	Allocations []TripBudgetAllocation   `json:"allocations,omitempty" gorm:"foreignKey:TripID"`
	Expenses    []Expense                `json:"expenses,omitempty" gorm:"foreignKey:TripID"`
	Settlements []Settlement             `json:"settlements,omitempty" gorm:"foreignKey:TripID"`
	Itinerary   []TripItineraryItem      `json:"itinerary,omitempty" gorm:"foreignKey:TripID"`
	Invitations []TripInvitation         `json:"invitations,omitempty" gorm:"foreignKey:TripID"`
}

// IsActiveNow verifica si el viaje está actualmente en curso
func (t *Trip) IsActiveNow() bool {
	now := time.Now()
	return t.Status == TripStatusActive && !now.Before(t.StartDate) && !now.After(t.EndDate)
}

// DaysTotal retorna la duración total del viaje en días (mínimo 1)
func (t *Trip) DaysTotal() int {
	days := int(t.EndDate.Sub(t.StartDate).Hours()/24) + 1
	if days < 1 {
		return 1
	}
	return days
}

// DaysRemaining retorna los días restantes del viaje desde hoy
func (t *Trip) DaysRemaining() int {
	now := time.Now()
	if now.After(t.EndDate) {
		return 0
	}
	if now.Before(t.StartDate) {
		return t.DaysTotal()
	}
	remaining := int(t.EndDate.Sub(now).Hours()/24) + 1
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ProgressPercentage retorna el porcentaje gastado del estimado total
func (t *Trip) ProgressPercentage() float64 {
	if t.EstimatedTotal == 0 {
		return 0
	}
	return (t.SpentTotal / t.EstimatedTotal) * 100
}

// RemainingAmount retorna el monto restante del presupuesto estimado
func (t *Trip) RemainingAmount() float64 {
	return t.EstimatedTotal - t.SpentTotal
}

// IsOverBudget verifica si el gasto supera al estimado
func (t *Trip) IsOverBudget() bool {
	return t.SpentTotal > t.EstimatedTotal && t.EstimatedTotal > 0
}

// CanBeStarted verifica si el viaje puede pasar a estado activo
func (t *Trip) CanBeStarted() bool {
	return t.Status == TripStatusPlanning
}

// CanBeCompleted verifica si el viaje puede marcarse como completado
func (t *Trip) CanBeCompleted() bool {
	return t.Status == TripStatusActive || t.Status == TripStatusPlanning
}

// CanBeCancelled verifica si el viaje puede cancelarse
func (t *Trip) CanBeCancelled() bool {
	return t.Status != TripStatusCompleted && t.Status != TripStatusCancelled
}
