package entity

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
)

// ExpenseSplitShareType describe cómo se distribuye el monto de un gasto
// entre los miembros del viaje.
type ExpenseSplitShareType string

const (
	ExpenseSplitShareTypeEqual      ExpenseSplitShareType = "equal"
	ExpenseSplitShareTypePercentage ExpenseSplitShareType = "percentage"
	ExpenseSplitShareTypeExact      ExpenseSplitShareType = "exact"
	ExpenseSplitShareTypeShares     ExpenseSplitShareType = "shares"
)

// ExpenseSplit representa la porción que un miembro debe asumir de un gasto
// concreto del viaje.
type ExpenseSplit struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	ExpenseID uint `json:"expense_id" gorm:"not null;index"`
	MemberID  uint `json:"member_id" gorm:"not null;index"`

	// Configuración de cómo se distribuye
	ShareType  ExpenseSplitShareType `json:"share_type" gorm:"type:varchar(20);not null;default:'equal'" validate:"oneof=equal percentage exact shares"`
	ShareValue float64               `json:"share_value" gorm:"default:0;type:decimal(15,4)"`

	// Monto final asignado a este miembro (calculado a partir del share)
	ShareAmount float64 `json:"share_amount" gorm:"not null;default:0;type:decimal(15,2)"`

	// Estado de pago de la deuda
	IsPaid bool       `json:"is_paid" gorm:"default:false;index"`
	PaidAt *time.Time `json:"paid_at"`

	// Relaciones
	Expense *Expense    `json:"expense,omitempty" gorm:"foreignKey:ExpenseID"`
	Member  *TripMember `json:"member,omitempty" gorm:"foreignKey:MemberID"`
}

// MarkPaid marca el split como pagado
func (s *ExpenseSplit) MarkPaid() {
	if s.IsPaid {
		return
	}
	now := time.Now()
	s.IsPaid = true
	s.PaidAt = &now
}

// MarkUnpaid revierte el estado de pago
func (s *ExpenseSplit) MarkUnpaid() {
	s.IsPaid = false
	s.PaidAt = nil
}

// RecalculateShares ajusta el ShareAmount de cada split en función del total
// del gasto y el ShareType. Si los splits son de tipo "equal", se reparte por
// igual; si son "percentage" se interpreta ShareValue como porcentaje (0-100);
// si son "shares" se interpreta como partes; si son "exact" se respeta el
// ShareValue como monto. Devuelve error si la suma no coincide con el total.
func RecalculateShares(splits []*ExpenseSplit, total float64) error {
	if len(splits) == 0 {
		return errors.New("no splits provided")
	}

	if total < 0 {
		return errors.New("total must be non-negative")
	}

	// Determinar tipo común
	shareType := splits[0].ShareType
	for _, s := range splits {
		if s.ShareType != shareType {
			return errors.New("mixed split types are not supported")
		}
	}

	switch shareType {
	case ExpenseSplitShareTypeEqual:
		distributeEqual(splits, total)
	case ExpenseSplitShareTypePercentage:
		if err := distributePercentage(splits, total); err != nil {
			return err
		}
	case ExpenseSplitShareTypeShares:
		if err := distributeShares(splits, total); err != nil {
			return err
		}
	case ExpenseSplitShareTypeExact:
		if err := distributeExact(splits, total); err != nil {
			return err
		}
	default:
		return errors.New("unknown share type")
	}

	// Verificar suma final con tolerancia de 1 centavo
	sum := 0.0
	for _, s := range splits {
		sum += s.ShareAmount
	}
	if math.Abs(sum-total) > 0.01 {
		// Ajuste residual al primer split para compensar redondeos
		splits[0].ShareAmount += total - sum
	}

	return nil
}

func distributeEqual(splits []*ExpenseSplit, total float64) {
	n := float64(len(splits))
	per := round2(total / n)
	for _, s := range splits {
		s.ShareAmount = per
	}
}

func distributePercentage(splits []*ExpenseSplit, total float64) error {
	totalPct := 0.0
	for _, s := range splits {
		totalPct += s.ShareValue
	}
	if math.Abs(totalPct-100) > 0.01 {
		return errors.New("percentages must sum to 100")
	}
	for _, s := range splits {
		s.ShareAmount = round2(total * s.ShareValue / 100)
	}
	return nil
}

func distributeShares(splits []*ExpenseSplit, total float64) error {
	totalShares := 0.0
	for _, s := range splits {
		totalShares += s.ShareValue
	}
	if totalShares <= 0 {
		return errors.New("total shares must be positive")
	}
	for _, s := range splits {
		s.ShareAmount = round2(total * s.ShareValue / totalShares)
	}
	return nil
}

func distributeExact(splits []*ExpenseSplit, total float64) error {
	sum := 0.0
	for _, s := range splits {
		s.ShareAmount = round2(s.ShareValue)
		sum += s.ShareAmount
	}
	if math.Abs(sum-total) > 0.01 {
		return errors.New("exact split values must sum to total")
	}
	return nil
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
