package entity

import (
	"time"

	"gorm.io/gorm"
)

// Settlement representa un pago realizado entre dos miembros del viaje para
// saldar una deuda parcial o total.
type Settlement struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	TripID         uint `json:"trip_id" gorm:"not null;index"`
	FromMemberID   uint `json:"from_member_id" gorm:"not null;index"`
	ToMemberID     uint `json:"to_member_id" gorm:"not null;index"`
	RecordedByUser uint `json:"recorded_by_user" gorm:"not null;index"`

	Amount   float64 `json:"amount" gorm:"not null;type:decimal(15,2)" validate:"required,gt=0"`
	Currency string  `json:"currency" gorm:"type:varchar(3);default:'USD'" validate:"len=3"`
	FxRate   float64 `json:"fx_rate" gorm:"default:1;type:decimal(18,8)"`

	PaidAt time.Time `json:"paid_at" gorm:"not null;index"`
	Notes  string    `json:"notes" validate:"max=500"`

	// Relaciones
	Trip       *Trip       `json:"trip,omitempty" gorm:"foreignKey:TripID"`
	FromMember *TripMember `json:"from_member,omitempty" gorm:"foreignKey:FromMemberID"`
	ToMember   *TripMember `json:"to_member,omitempty" gorm:"foreignKey:ToMemberID"`
}

// AmountInPrimaryCurrency convierte el monto a la moneda primaria del viaje
// utilizando el FxRate registrado.
func (s *Settlement) AmountInPrimaryCurrency() float64 {
	if s.FxRate == 0 {
		return s.Amount
	}
	return s.Amount * s.FxRate
}
