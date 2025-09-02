package entity

import (
	"time"

	"gorm.io/gorm"
)

// Category representa una categoría de gasto
type Category struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Información básica
	Name        string `json:"name" gorm:"not null" validate:"required,min=1,max=50"`
	Description string `json:"description" validate:"max=200"`
	Icon        string `json:"icon" validate:"max=50"`
	Color       string `json:"color" gorm:"default:'#007bff'" validate:"hexcolor"`

	// Configuración
	IsActive  bool `json:"is_active" gorm:"default:true"`
	IsDefault bool `json:"is_default" gorm:"default:false"` // Categorías predefinidas del sistema
	SortOrder int  `json:"sort_order" gorm:"default:0"`

	// Relación con usuario (null para categorías del sistema)
	UserID *uint `json:"user_id" gorm:"index"`
}

// CategoryType define tipos predefinidos de categorías
type CategoryType string

const (
	CategoryTypeFood          CategoryType = "food"          // Alimentación
	CategoryTypeTransport     CategoryType = "transport"     // Transporte
	CategoryTypeEntertainment CategoryType = "entertainment" // Ocio
	CategoryTypeHealth        CategoryType = "health"        // Salud
	CategoryTypeEducation     CategoryType = "education"     // Educación
	CategoryTypeUtilities     CategoryType = "utilities"     // Servicios
	CategoryTypeShopping      CategoryType = "shopping"      // Compras
	CategoryTypeOther         CategoryType = "other"         // Otros
)

// DefaultCategories retorna las categorías predefinidas del sistema
func DefaultCategories() []Category {
	return []Category{
		{Name: "Alimentación", Description: "Comida, supermercado, restaurantes", Icon: "🍽️", Color: "#FF6B35", IsDefault: true, SortOrder: 1},
		{Name: "Transporte", Description: "Gasolina, transporte público, Uber", Icon: "🚗", Color: "#4ECDC4", IsDefault: true, SortOrder: 2},
		{Name: "Ocio", Description: "Entretenimiento, cine, salidas", Icon: "🎭", Color: "#45B7D1", IsDefault: true, SortOrder: 3},
		{Name: "Servicios", Description: "Luz, agua, internet, teléfono", Icon: "🏠", Color: "#96CEB4", IsDefault: true, SortOrder: 4},
		{Name: "Salud", Description: "Médico, medicinas, seguros", Icon: "⚕️", Color: "#FFEAA7", IsDefault: true, SortOrder: 5},
		{Name: "Compras", Description: "Ropa, electrónicos, compras varias", Icon: "🛍️", Color: "#DDA0DD", IsDefault: true, SortOrder: 6},
		{Name: "Educación", Description: "Cursos, libros, capacitación", Icon: "📚", Color: "#74B9FF", IsDefault: true, SortOrder: 7},
		{Name: "Otros", Description: "Gastos varios no clasificados", Icon: "💼", Color: "#FDCB6E", IsDefault: true, SortOrder: 8},
	}
}

// IsSystemCategory verifica si es una categoría del sistema
func (c *Category) IsSystemCategory() bool {
	return c.UserID == nil && c.IsDefault
}

// IsUserCategory verifica si es una categoría personalizada del usuario
func (c *Category) IsUserCategory() bool {
	return c.UserID != nil
}

// CanBeDeleted verifica si la categoría puede ser eliminada
func (c *Category) CanBeDeleted() bool {
	// Las categorías del sistema no se pueden eliminar
	return !c.IsSystemCategory()
}

// GetDisplayName retorna el nombre con emoji si tiene icono
func (c *Category) GetDisplayName() string {
	if c.Icon != "" {
		return c.Icon + " " + c.Name
	}
	return c.Name
}
