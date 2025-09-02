package repository

import (
	"gorm.io/gorm"

	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase/repo"
)

// CategoryPostgres implementa CategoryRepo usando PostgreSQL
type CategoryPostgres struct {
	db *gorm.DB
}

// NewCategoryPostgres crea una nueva instancia de CategoryPostgres
func NewCategoryPostgres(db *gorm.DB) repo.CategoryRepo {
	return &CategoryPostgres{db: db}
}

// Create crea una nueva categoría
func (r *CategoryPostgres) Create(category *entity.Category) error {
	return r.db.Create(category).Error
}

// GetByID obtiene una categoría por ID
func (r *CategoryPostgres) GetByID(id uint) (*entity.Category, error) {
	var category entity.Category
	err := r.db.First(&category, id).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetByUserID obtiene todas las categorías de un usuario (incluyendo las del sistema)
func (r *CategoryPostgres) GetByUserID(userID uint) ([]*entity.Category, error) {
	var categories []*entity.Category

	// Obtener categorías del sistema (user_id IS NULL) y categorías del usuario
	err := r.db.Where("user_id IS NULL OR user_id = ?", userID).
		Where("is_active = ?", true).
		Order("is_default DESC, sort_order ASC, name ASC").
		Find(&categories).Error

	return categories, err
}

// GetDefaultCategories obtiene todas las categorías por defecto del sistema
func (r *CategoryPostgres) GetDefaultCategories() ([]*entity.Category, error) {
	var categories []*entity.Category
	err := r.db.Where("is_default = ? AND user_id IS NULL", true).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Find(&categories).Error
	return categories, err
}

// GetUserCategories obtiene solo las categorías personalizadas del usuario
func (r *CategoryPostgres) GetUserCategories(userID uint) ([]*entity.Category, error) {
	var categories []*entity.Category
	err := r.db.Where("user_id = ?", userID).
		Where("is_active = ?", true).
		Order("sort_order ASC, name ASC").
		Find(&categories).Error
	return categories, err
}

// Update actualiza una categoría
func (r *CategoryPostgres) Update(category *entity.Category) error {
	return r.db.Save(category).Error
}

// Delete elimina una categoría (solo categorías del usuario, no del sistema)
func (r *CategoryPostgres) Delete(id uint) error {
	// Solo permitir eliminar categorías de usuario, no del sistema
	return r.db.Where("user_id IS NOT NULL").Delete(&entity.Category{}, id).Error
}

// GetAllAvailableForUser obtiene todas las categorías disponibles para un usuario (sistema + usuario)
func (r *CategoryPostgres) GetAllAvailableForUser(userID uint) ([]*entity.Category, error) {
	var categories []*entity.Category

	// Categorías del sistema + categorías propias del usuario, ordenadas por tipo y luego por orden
	err := r.db.Where("(user_id IS NULL AND is_default = ?) OR user_id = ?", true, userID).
		Where("is_active = ?", true).
		Order("is_default DESC, sort_order ASC, name ASC").
		Find(&categories).Error

	return categories, err
}

// GetByName busca una categoría por nombre para un usuario específico
func (r *CategoryPostgres) GetByName(name string, userID *uint) (*entity.Category, error) {
	var category entity.Category

	if userID != nil {
		// Buscar primero en categorías del usuario
		err := r.db.Where("name = ? AND user_id = ?", name, *userID).First(&category).Error
		if err == nil {
			return &category, nil
		}
	}

	// Si no se encuentra o userID es nil, buscar en categorías del sistema
	err := r.db.Where("name = ? AND user_id IS NULL AND is_default = ?", name, true).
		First(&category).Error
	if err != nil {
		return nil, err
	}

	return &category, nil
}

// InitializeDefaultCategories inicializa las categorías por defecto del sistema
func (r *CategoryPostgres) InitializeDefaultCategories() error {
	// Verificar si ya existen categorías por defecto
	var count int64
	r.db.Model(&entity.Category{}).Where("is_default = ? AND user_id IS NULL", true).Count(&count)

	if count > 0 {
		return nil // Ya están inicializadas
	}

	// Crear categorías por defecto
	defaultCategories := entity.DefaultCategories()

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, category := range defaultCategories {
			if err := tx.Create(&category).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMostUsedByUser obtiene las categorías más usadas por un usuario
func (r *CategoryPostgres) GetMostUsedByUser(userID uint, limit int) ([]*entity.Category, error) {
	var categories []*entity.Category

	// Subconsulta para contar el uso de cada categoría
	err := r.db.Table("categories").
		Select("categories.*, COUNT(expenses.id) as usage_count").
		Joins("LEFT JOIN expenses ON categories.id = expenses.category_id AND expenses.user_id = ?", userID).
		Where("(categories.user_id IS NULL AND categories.is_default = ?) OR categories.user_id = ?", true, userID).
		Where("categories.is_active = ?", true).
		Group("categories.id").
		Order("usage_count DESC, categories.sort_order ASC").
		Limit(limit).
		Find(&categories).Error

	return categories, err
}

// UpdateBatchSortOrder actualiza el orden de clasificación de múltiples categorías del usuario
func (r *CategoryPostgres) UpdateBatchSortOrder(userID uint, categoryOrders []struct {
	ID        uint
	SortOrder int
}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, order := range categoryOrders {
			err := tx.Model(&entity.Category{}).
				Where("id = ? AND user_id = ?", order.ID, userID).
				Update("sort_order", order.SortOrder).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SetActive cambia el estado activo de una categoría
func (r *CategoryPostgres) SetActive(id uint, active bool) error {
	// Solo permitir cambiar estado en categorías de usuario, no del sistema
	return r.db.Model(&entity.Category{}).
		Where("id = ? AND user_id IS NOT NULL", id).
		Update("is_active", active).Error
}

// GetActiveCategories obtiene todas las categorías activas para un usuario
func (r *CategoryPostgres) GetActiveCategories(userID uint) ([]*entity.Category, error) {
	var categories []*entity.Category

	err := r.db.Where("(user_id IS NULL AND is_default = ?) OR user_id = ?", true, userID).
		Where("is_active = ?", true).
		Order("is_default DESC, sort_order ASC, name ASC").
		Find(&categories).Error

	return categories, err
}

// UpdateSortOrder actualiza el orden de clasificación de una categoría
func (r *CategoryPostgres) UpdateSortOrder(id uint, sortOrder int) error {
	return r.db.Model(&entity.Category{}).
		Where("id = ? AND user_id IS NOT NULL", id).
		Update("sort_order", sortOrder).Error
}

// HasExpenses verifica si una categoría tiene gastos asociados
func (r *CategoryPostgres) HasExpenses(categoryID uint) (bool, error) {
	var count int64
	err := r.db.Model(&entity.Expense{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error

	return count > 0, err
}

// GetCategoryUsageStats obtiene estadísticas de uso de una categoría
func (r *CategoryPostgres) GetCategoryUsageStats(categoryID uint) (int64, float64, error) {
	var count int64
	var totalAmount float64

	err := r.db.Model(&entity.Expense{}).
		Where("category_id = ? AND status IN (?)", categoryID, []string{"confirmed", "pending"}).
		Count(&count).Error

	if err != nil {
		return 0, 0, err
	}

	err = r.db.Model(&entity.Expense{}).
		Where("category_id = ? AND status IN (?)", categoryID, []string{"confirmed", "pending"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalAmount).Error

	return count, totalAmount, err
}

// CreateDefaultCategories crea las categorías por defecto del sistema
func (r *CategoryPostgres) CreateDefaultCategories() error {
	defaultCategories := entity.DefaultCategories()

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, category := range defaultCategories {
			if err := tx.Create(&category).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// EnsureDefaultCategoriesExist verifica y crea las categorías por defecto si no existen
func (r *CategoryPostgres) EnsureDefaultCategoriesExist() error {
	// Verificar si ya existen categorías por defecto
	var count int64
	r.db.Model(&entity.Category{}).Where("is_default = ? AND user_id IS NULL", true).Count(&count)

	if count > 0 {
		return nil // Ya están inicializadas
	}

	// Crear categorías por defecto
	return r.CreateDefaultCategories()
}

// GetCategoriesWithBudgetInfo obtiene categorías con información de presupuesto para un mes específico
func (r *CategoryPostgres) GetCategoriesWithBudgetInfo(userID uint, year, month int) ([]*entity.Category, error) {
	var categories []*entity.Category

	err := r.db.Preload("BudgetAllocations", func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN budgets ON budget_allocations.budget_id = budgets.id").
			Where("budgets.user_id = ? AND budgets.year = ? AND budgets.month = ?", userID, year, month)
	}).Where("(user_id IS NULL AND is_default = ?) OR user_id = ?", true, userID).
		Where("is_active = ?", true).
		Order("is_default DESC, sort_order ASC, name ASC").
		Find(&categories).Error

	return categories, err
}
