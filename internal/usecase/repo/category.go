package repo

import "github.com/nick130920/fintech-backend/internal/entity"

// CategoryRepo define la interfaz para operaciones de categoría en la base de datos
type CategoryRepo interface {
	// Operaciones básicas CRUD
	Create(category *entity.Category) error
	GetByID(id uint) (*entity.Category, error)
	GetByUserID(userID uint) ([]*entity.Category, error)
	Update(category *entity.Category) error
	Delete(id uint) error

	// Operaciones específicas de categorías
	GetDefaultCategories() ([]*entity.Category, error)
	GetUserCategories(userID uint) ([]*entity.Category, error)
	GetAllAvailableForUser(userID uint) ([]*entity.Category, error) // Sistema + Usuario

	// Búsquedas específicas
	GetByName(name string, userID *uint) (*entity.Category, error)
	GetActiveCategories(userID uint) ([]*entity.Category, error)

	// Operaciones de configuración
	SetActive(id uint, active bool) error
	UpdateSortOrder(id uint, sortOrder int) error

	// Verificaciones
	HasExpenses(categoryID uint) (bool, error)
	GetCategoryUsageStats(categoryID uint) (int64, float64, error) // count, total amount

	// Inicialización del sistema
	CreateDefaultCategories() error
	EnsureDefaultCategoriesExist() error
}
