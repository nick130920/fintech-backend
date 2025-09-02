package dto

// CreateCategoryRequest representa la estructura para crear una categoría personalizada
type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=50"`
	Description string `json:"description" validate:"max=200"`
	Icon        string `json:"icon" validate:"max=50"`
	Color       string `json:"color" validate:"omitempty,hexcolor"`
}

// UpdateCategoryRequest representa la estructura para actualizar una categoría
type UpdateCategoryRequest struct {
	Name        string `json:"name" validate:"omitempty,min=1,max=50"`
	Description string `json:"description" validate:"max=200"`
	Icon        string `json:"icon" validate:"max=50"`
	Color       string `json:"color" validate:"omitempty,hexcolor"`
	IsActive    *bool  `json:"is_active"`
	SortOrder   *int   `json:"sort_order"`
}

// CategorySummaryResponse representa el resumen de una categoría
type CategorySummaryResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	Color          string `json:"color"`
	DisplayName    string `json:"display_name"`
	IsActive       bool   `json:"is_active"`
	IsDefault      bool   `json:"is_default"`
	IsUserCategory bool   `json:"is_user_category"`
	SortOrder      int    `json:"sort_order"`
	CanBeDeleted   bool   `json:"can_be_deleted"`
}

// CategoriesResponse representa la respuesta con todas las categorías disponibles
type CategoriesResponse struct {
	DefaultCategories []CategorySummaryResponse `json:"default_categories"`
	UserCategories    []CategorySummaryResponse `json:"user_categories"`
	TotalCount        int                       `json:"total_count"`
}
