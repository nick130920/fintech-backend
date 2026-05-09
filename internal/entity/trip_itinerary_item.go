package entity

import (
	"time"

	"gorm.io/gorm"
)

// TripItineraryType clasifica el tipo de actividad o reserva del itinerario
type TripItineraryType string

const (
	TripItineraryTypeFlight    TripItineraryType = "flight"
	TripItineraryTypeHotel     TripItineraryType = "hotel"
	TripItineraryTypeTransport TripItineraryType = "transport"
	TripItineraryTypeActivity  TripItineraryType = "activity"
	TripItineraryTypeFood      TripItineraryType = "food"
	TripItineraryTypeOther     TripItineraryType = "other"
)

// TripItineraryItem representa una entrada del itinerario del viaje. Puede
// vincularse opcionalmente a un Expense real para comparar estimado vs real.
type TripItineraryItem struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	TripID uint `json:"trip_id" gorm:"not null;index"`

	// Día del viaje (fecha exacta) y hora opcional como string HH:mm
	Day  time.Time `json:"day" gorm:"not null;index"`
	Time string    `json:"time" gorm:"type:varchar(5)" validate:"omitempty,len=5"`

	Type        TripItineraryType `json:"type" gorm:"type:varchar(20);not null;default:'activity'" validate:"oneof=flight hotel transport activity food other"`
	Title       string            `json:"title" gorm:"not null" validate:"required,min=1,max=200"`
	Description string            `json:"description" validate:"max=1000"`
	Location    string            `json:"location" validate:"max=200"`

	EstimatedCost float64 `json:"estimated_cost" gorm:"default:0;type:decimal(15,2)"`
	Currency      string  `json:"currency" gorm:"type:varchar(3);default:'USD'" validate:"len=3"`

	ExpenseID *uint `json:"expense_id" gorm:"index"`

	// Relaciones
	Trip    *Trip    `json:"trip,omitempty" gorm:"foreignKey:TripID"`
	Expense *Expense `json:"expense,omitempty" gorm:"foreignKey:ExpenseID"`
}

// HasRealExpense indica si el item ya tiene un gasto real vinculado
func (i *TripItineraryItem) HasRealExpense() bool {
	return i.ExpenseID != nil
}

// Variance retorna la diferencia entre el costo real y el estimado (positiva = sobrepasó)
func (i *TripItineraryItem) Variance(realAmount float64) float64 {
	return realAmount - i.EstimatedCost
}
