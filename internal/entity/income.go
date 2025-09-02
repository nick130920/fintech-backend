package entity

import (
	"time"

	"gorm.io/gorm"
)

// IncomeSource define los tipos de fuente de ingreso
type IncomeSource string

const (
	IncomeSourceSalary     IncomeSource = "salary"     // Salario
	IncomeSourceFreelance  IncomeSource = "freelance"  // Freelance
	IncomeSourceInvestment IncomeSource = "investment" // Inversiones
	IncomeSourceBusiness   IncomeSource = "business"   // Negocio
	IncomeSourceRental     IncomeSource = "rental"     // Renta
	IncomeSourceBonus      IncomeSource = "bonus"      // Bonos
	IncomeSourceGift       IncomeSource = "gift"       // Regalo
	IncomeSourceOther      IncomeSource = "other"      // Otros
)

// IncomeFrequency define la frecuencia de ingresos recurrentes
type IncomeFrequency string

const (
	IncomeFrequencyOnce      IncomeFrequency = "once"      // Una vez
	IncomeFrequencyWeekly    IncomeFrequency = "weekly"    // Semanal
	IncomeFrequencyBiweekly  IncomeFrequency = "biweekly"  // Quincenal
	IncomeFrequencyMonthly   IncomeFrequency = "monthly"   // Mensual
	IncomeFrequencyQuarterly IncomeFrequency = "quarterly" // Trimestral
	IncomeFrequencyYearly    IncomeFrequency = "yearly"    // Anual
)

// Income representa un ingreso del usuario
type Income struct {
	gorm.Model
	UserID      uint         `json:"user_id" gorm:"not null;index"`
	Amount      float64      `json:"amount" gorm:"not null" validate:"gt=0"`
	Description string       `json:"description" gorm:"not null" validate:"required,max=255"`
	Source      IncomeSource `json:"source" gorm:"not null" validate:"required"`
	Date        time.Time    `json:"date" gorm:"not null"`
	Notes       string       `json:"notes,omitempty" gorm:"type:text"`
	Currency    string       `json:"currency" gorm:"size:3;default:'USD'" validate:"len=3"`

	// Campos para ingresos recurrentes
	IsRecurring    bool             `json:"is_recurring" gorm:"default:false"`
	Frequency      *IncomeFrequency `json:"frequency,omitempty" gorm:"type:varchar(20)"`
	NextDate       *time.Time       `json:"next_date,omitempty"`
	EndDate        *time.Time       `json:"end_date,omitempty"`
	RecurringUntil *time.Time       `json:"recurring_until,omitempty"`

	// Metadatos
	TaxDeducted float64 `json:"tax_deducted" gorm:"default:0"`
	NetAmount   float64 `json:"net_amount" gorm:"default:0"` // amount - tax_deducted

	// Relaciones
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// BeforeCreate se ejecuta antes de crear el ingreso
func (i *Income) BeforeCreate(tx *gorm.DB) error {
	// Calcular el monto neto
	i.NetAmount = i.Amount - i.TaxDeducted

	// Si es recurrente y no tiene next_date, calcularlo
	if i.IsRecurring && i.NextDate == nil {
		nextDate := i.CalculateNextDate()
		i.NextDate = &nextDate
	}

	return nil
}

// BeforeUpdate se ejecuta antes de actualizar el ingreso
func (i *Income) BeforeUpdate(tx *gorm.DB) error {
	// Recalcular el monto neto
	i.NetAmount = i.Amount - i.TaxDeducted
	return nil
}

// CalculateNextDate calcula la próxima fecha basada en la frecuencia
func (i *Income) CalculateNextDate() time.Time {
	if !i.IsRecurring || i.Frequency == nil {
		return i.Date
	}

	switch *i.Frequency {
	case IncomeFrequencyWeekly:
		return i.Date.AddDate(0, 0, 7)
	case IncomeFrequencyBiweekly:
		return i.Date.AddDate(0, 0, 14)
	case IncomeFrequencyMonthly:
		return i.Date.AddDate(0, 1, 0)
	case IncomeFrequencyQuarterly:
		return i.Date.AddDate(0, 3, 0)
	case IncomeFrequencyYearly:
		return i.Date.AddDate(1, 0, 0)
	default:
		return i.Date
	}
}

// IsActive verifica si el ingreso recurrente sigue activo
func (i *Income) IsActive() bool {
	if !i.IsRecurring {
		return true
	}

	now := time.Now()
	if i.EndDate != nil && now.After(*i.EndDate) {
		return false
	}

	if i.RecurringUntil != nil && now.After(*i.RecurringUntil) {
		return false
	}

	return true
}

// GetSourceDisplayName retorna el nombre para mostrar de la fuente
func (i *Income) GetSourceDisplayName() string {
	switch i.Source {
	case IncomeSourceSalary:
		return "Salario"
	case IncomeSourceFreelance:
		return "Freelance"
	case IncomeSourceInvestment:
		return "Inversiones"
	case IncomeSourceBusiness:
		return "Negocio"
	case IncomeSourceRental:
		return "Renta"
	case IncomeSourceBonus:
		return "Bono"
	case IncomeSourceGift:
		return "Regalo"
	case IncomeSourceOther:
		return "Otros"
	default:
		return string(i.Source)
	}
}

// GetFrequencyDisplayName retorna el nombre para mostrar de la frecuencia
func (i *Income) GetFrequencyDisplayName() string {
	if i.Frequency == nil {
		return "Una vez"
	}

	switch *i.Frequency {
	case IncomeFrequencyWeekly:
		return "Semanal"
	case IncomeFrequencyBiweekly:
		return "Quincenal"
	case IncomeFrequencyMonthly:
		return "Mensual"
	case IncomeFrequencyQuarterly:
		return "Trimestral"
	case IncomeFrequencyYearly:
		return "Anual"
	default:
		return "Una vez"
	}
}

// GetFormattedAmount retorna el monto formateado con moneda
func (i *Income) GetFormattedAmount() string {
	return FormatCurrency(i.Amount, i.Currency)
}

// GetFormattedNetAmount retorna el monto neto formateado con moneda
func (i *Income) GetFormattedNetAmount() string {
	return FormatCurrency(i.NetAmount, i.Currency)
}

// CanBeModified verifica si el ingreso puede ser modificado
func (i *Income) CanBeModified() bool {
	// Los ingresos pueden ser modificados dentro de 30 días
	return time.Since(i.Date).Hours() < 24*30
}

// CanBeDeleted verifica si el ingreso puede ser eliminado
func (i *Income) CanBeDeleted() bool {
	// Los ingresos pueden ser eliminados dentro de 7 días
	return time.Since(i.Date).Hours() < 24*7
}

// IsFutureIncome verifica si es un ingreso futuro
func (i *Income) IsFutureIncome() bool {
	return i.Date.After(time.Now())
}

// TableName especifica el nombre de la tabla
func (Income) TableName() string {
	return "incomes"
}
