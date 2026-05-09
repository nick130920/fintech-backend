package entity

import (
	"time"

	"gorm.io/gorm"
)

// TripBudgetAllocation representa el monto estimado para una categoría dentro
// del presupuesto de un viaje (alojamiento, comida, transporte, etc.).
type TripBudgetAllocation struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	TripID     uint `json:"trip_id" gorm:"not null;index"`
	CategoryID uint `json:"category_id" gorm:"not null;index"`

	EstimatedAmount float64 `json:"estimated_amount" gorm:"not null;type:decimal(15,2)" validate:"required,gte=0"`
	SpentAmount     float64 `json:"spent_amount" gorm:"default:0;type:decimal(15,2)"`

	Currency string `json:"currency" gorm:"type:varchar(3);default:'USD'" validate:"len=3"`

	Notes string `json:"notes" validate:"max=500"`

	// Relaciones
	Trip     *Trip     `json:"trip,omitempty" gorm:"foreignKey:TripID"`
	Category *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

// RemainingAmount devuelve el saldo restante de la asignación
func (a *TripBudgetAllocation) RemainingAmount() float64 {
	return a.EstimatedAmount - a.SpentAmount
}

// ProgressPercentage retorna el porcentaje gastado
func (a *TripBudgetAllocation) ProgressPercentage() float64 {
	if a.EstimatedAmount == 0 {
		return 0
	}
	return (a.SpentAmount / a.EstimatedAmount) * 100
}

// IsOverBudget indica si se excedió el monto estimado
func (a *TripBudgetAllocation) IsOverBudget() bool {
	return a.SpentAmount > a.EstimatedAmount && a.EstimatedAmount > 0
}

// DailySuggested calcula el monto diario sugerido para el resto del viaje
func (a *TripBudgetAllocation) DailySuggested(remainingDays int) float64 {
	if remainingDays <= 0 {
		return 0
	}
	remaining := a.RemainingAmount()
	if remaining <= 0 {
		return 0
	}
	return remaining / float64(remainingDays)
}
