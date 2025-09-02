package entity

import (
	"time"

	"gorm.io/gorm"
)

// Budget representa el presupuesto mensual de un usuario
type Budget struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relación con usuario
	UserID uint `json:"user_id" gorm:"not null;index"`

	// Período del presupuesto
	Year  int `json:"year" gorm:"not null;index"`
	Month int `json:"month" gorm:"not null;index"` // 1-12

	// Montos
	TotalAmount     float64 `json:"total_amount" gorm:"not null;type:decimal(15,2)" validate:"required,gt=0"`
	SpentAmount     float64 `json:"spent_amount" gorm:"default:0;type:decimal(15,2)"`
	RemainingAmount float64 `json:"remaining_amount" gorm:"type:decimal(15,2)"`

	// Estado
	IsActive bool `json:"is_active" gorm:"default:true"`

	// Configuración
	AutoCreateNext bool `json:"auto_create_next" gorm:"default:true"` // Crear automáticamente el siguiente mes

	// Relaciones
	Allocations []BudgetAllocation `json:"allocations" gorm:"foreignKey:BudgetID"`
	Expenses    []Expense          `json:"expenses" gorm:"foreignKey:BudgetID"`
}

// BudgetAllocation representa la asignación de presupuesto por categoría
type BudgetAllocation struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relaciones
	BudgetID   uint `json:"budget_id" gorm:"not null;index"`
	CategoryID uint `json:"category_id" gorm:"not null;index"`

	// Montos asignados
	AllocatedAmount float64 `json:"allocated_amount" gorm:"not null;type:decimal(15,2)" validate:"required,gte=0"`
	SpentAmount     float64 `json:"spent_amount" gorm:"default:0;type:decimal(15,2)"`
	RemainingAmount float64 `json:"remaining_amount" gorm:"type:decimal(15,2)"`

	// Configuración de límites diarios
	DailyLimit        float64    `json:"daily_limit" gorm:"type:decimal(15,2)"`         // Límite diario calculado
	CurrentDailyLimit float64    `json:"current_daily_limit" gorm:"type:decimal(15,2)"` // Límite actual con rollover
	LastCalculatedAt  *time.Time `json:"last_calculated_at"`                            // Última vez que se calculó

	// Alertas
	AlertThreshold float64 `json:"alert_threshold" gorm:"default:0.8;type:decimal(3,2)"` // % para alertar (ej: 0.8 = 80%)
	IsOverBudget   bool    `json:"is_over_budget" gorm:"default:false"`

	// Relaciones
	Budget   Budget    `json:"budget" gorm:"foreignKey:BudgetID"`
	Category Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Expenses []Expense `json:"expenses" gorm:"foreignKey:AllocationID"`
}

// CalculateRemainingAmount actualiza el monto restante
func (b *Budget) CalculateRemainingAmount() {
	b.RemainingAmount = b.TotalAmount - b.SpentAmount
}

// IsCurrentMonth verifica si es el presupuesto del mes actual
func (b *Budget) IsCurrentMonth() bool {
	now := time.Now()
	return b.Year == now.Year() && b.Month == int(now.Month())
}

// GetPeriodString retorna el período en formato string
func (b *Budget) GetPeriodString() string {
	months := []string{
		"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
		"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
	}
	return months[b.Month] + " " + string(rune(b.Year))
}

// GetProgressPercentage retorna el porcentaje gastado
func (b *Budget) GetProgressPercentage() float64 {
	if b.TotalAmount == 0 {
		return 0
	}
	return (b.SpentAmount / b.TotalAmount) * 100
}

// GetRemainingDays retorna los días restantes del mes
func (b *Budget) GetRemainingDays() int {
	now := time.Now()

	// Si no es el mes actual, retornar 0
	if !b.IsCurrentMonth() {
		return 0
	}

	// Obtener el último día del mes
	lastDay := time.Date(b.Year, time.Month(b.Month+1), 0, 0, 0, 0, 0, time.UTC).Day()

	// Días restantes incluyendo hoy
	remaining := lastDay - now.Day() + 1
	if remaining < 0 {
		return 0
	}

	return remaining
}

// CalculateRemainingAmount actualiza el monto restante de la asignación
func (ba *BudgetAllocation) CalculateRemainingAmount() {
	ba.RemainingAmount = ba.AllocatedAmount - ba.SpentAmount
}

// GetProgressPercentage retorna el porcentaje gastado de la asignación
func (ba *BudgetAllocation) GetProgressPercentage() float64 {
	if ba.AllocatedAmount == 0 {
		return 0
	}
	return (ba.SpentAmount / ba.AllocatedAmount) * 100
}

// IsOverBudgetCheck verifica si se ha excedido el presupuesto
func (ba *BudgetAllocation) IsOverBudgetCheck() bool {
	return ba.SpentAmount > ba.AllocatedAmount
}

// ShouldAlert verifica si debe enviar alerta
func (ba *BudgetAllocation) ShouldAlert() bool {
	if ba.AllocatedAmount == 0 {
		return false
	}

	percentage := ba.GetProgressPercentage() / 100
	return percentage >= ba.AlertThreshold
}

// CalculateDailyLimit calcula el límite diario basado en días restantes
func (ba *BudgetAllocation) CalculateDailyLimit(remainingDays int) {
	if remainingDays <= 0 {
		ba.DailyLimit = 0
		ba.CurrentDailyLimit = 0
		return
	}

	// Límite base = monto restante / días restantes
	ba.DailyLimit = ba.RemainingAmount / float64(remainingDays)

	// Si es negativo (ya se excedió), poner en 0
	if ba.DailyLimit < 0 {
		ba.DailyLimit = 0
	}

	// Inicializar el límite actual si no existe
	if ba.CurrentDailyLimit == 0 {
		ba.CurrentDailyLimit = ba.DailyLimit
	}
}

// AddRollover agrega saldo no gastado al límite del día siguiente
func (ba *BudgetAllocation) AddRollover(unspentAmount float64) {
	ba.CurrentDailyLimit += unspentAmount
}

// ResetDailyLimit reinicia el límite diario al calculado
func (ba *BudgetAllocation) ResetDailyLimit() {
	ba.CurrentDailyLimit = ba.DailyLimit
	now := time.Now()
	ba.LastCalculatedAt = &now
}

// CanSpend verifica si se puede gastar una cantidad
func (ba *BudgetAllocation) CanSpend(amount float64) bool {
	return ba.CurrentDailyLimit >= amount
}

// GetAllocationPercentage retorna el porcentaje de asignación del presupuesto total
func (ba *BudgetAllocation) GetAllocationPercentage(totalBudget float64) float64 {
	if totalBudget == 0 {
		return 0
	}
	return (ba.AllocatedAmount / totalBudget) * 100
}
