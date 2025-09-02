package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/entity"
	"github.com/nick130920/proyecto-fintech/internal/usecase/repo"
	"github.com/nick130920/proyecto-fintech/pkg/validator"
)

// CategoryHandler maneja las peticiones HTTP relacionadas con categorías
type CategoryHandler struct {
	categoryRepo repo.CategoryRepo
	validator    *validator.Validator
}

// NewCategoryHandler crea una nueva instancia de CategoryHandler
func NewCategoryHandler(categoryRepo repo.CategoryRepo) *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: categoryRepo,
		validator:    validator.New(),
	}
}

// GetCategories godoc
// @Summary      Obtener categorías
// @Description  Obtiene todas las categorías disponibles para el usuario (sistema + personalizadas)
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.CategoriesResponse}
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/categories [get]
func (h *CategoryHandler) GetCategories(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener categorías por defecto del sistema
	defaultCategories, err := h.categoryRepo.GetDefaultCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al obtener categorías del sistema",
			Details: err.Error(),
		})
		return
	}

	// Obtener categorías personalizadas del usuario
	userCategories, err := h.categoryRepo.GetUserCategories(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al obtener categorías del usuario",
			Details: err.Error(),
		})
		return
	}

	// Mapear a DTOs
	defaultCategoriesDTO := make([]dto.CategorySummaryResponse, len(defaultCategories))
	for i, cat := range defaultCategories {
		defaultCategoriesDTO[i] = mapCategoryToSummaryResponse(cat)
	}

	userCategoriesDTO := make([]dto.CategorySummaryResponse, len(userCategories))
	for i, cat := range userCategories {
		userCategoriesDTO[i] = mapCategoryToSummaryResponse(cat)
	}

	response := dto.CategoriesResponse{
		DefaultCategories: defaultCategoriesDTO,
		UserCategories:    userCategoriesDTO,
		TotalCount:        len(defaultCategoriesDTO) + len(userCategoriesDTO),
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Categorías obtenidas exitosamente",
		Data:    response,
	})
}

// CreateCategory godoc
// @Summary      Crear categoría personalizada
// @Description  Crea una nueva categoría personalizada para el usuario
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateCategoryRequest true "Datos de la categoría"
// @Success      201  {object}  dto.Response{data=dto.CategorySummaryResponse}
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Datos de entrada inválidos",
			Details: err.Error(),
		})
		return
	}

	// Validar datos
	if err := h.validator.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "Error de validación",
			Details: err.Error(),
		})
		return
	}

	// Verificar que no existe una categoría con el mismo nombre
	existingCategory, _ := h.categoryRepo.GetByName(req.Name, &userID)
	if existingCategory != nil {
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Code:    "CATEGORY_EXISTS",
			Message: "Ya existe una categoría con este nombre",
		})
		return
	}

	// Crear la categoría
	category := &entity.Category{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		IsActive:    true,
		IsDefault:   false,
		UserID:      &userID,
		SortOrder:   0,
	}

	if err := h.categoryRepo.Create(category); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al crear la categoría",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Code:    "SUCCESS",
		Message: "Categoría creada exitosamente",
		Data:    mapCategoryToSummaryResponse(category),
	})
}

// UpdateCategory godoc
// @Summary      Actualizar categoría personalizada
// @Description  Actualiza una categoría personalizada del usuario
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      uint                          true  "ID de la categoría"
// @Param        request body      dto.UpdateCategoryRequest     true  "Datos de actualización"
// @Success      200     {object}  dto.Response{data=dto.CategorySummaryResponse}
// @Failure      400     {object}  dto.ErrorResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/v1/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID de la categoría
	categoryID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de categoría inválido",
		})
		return
	}

	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Datos de entrada inválidos",
			Details: err.Error(),
		})
		return
	}

	// Validar datos
	if err := h.validator.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "Error de validación",
			Details: err.Error(),
		})
		return
	}

	// Obtener la categoría
	category, err := h.categoryRepo.GetByID(uint(categoryID))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Code:    "CATEGORY_NOT_FOUND",
			Message: "Categoría no encontrada",
		})
		return
	}

	// Verificar que pertenece al usuario y no es del sistema
	if category.UserID == nil || *category.UserID != userID {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: "No tienes permisos para modificar esta categoría",
		})
		return
	}

	// Actualizar campos
	if req.Name != "" {
		category.Name = req.Name
	}
	if req.Description != "" {
		category.Description = req.Description
	}
	if req.Icon != "" {
		category.Icon = req.Icon
	}
	if req.Color != "" {
		category.Color = req.Color
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}

	if err := h.categoryRepo.Update(category); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al actualizar la categoría",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Categoría actualizada exitosamente",
		Data:    mapCategoryToSummaryResponse(category),
	})
}

// DeleteCategory godoc
// @Summary      Eliminar categoría personalizada
// @Description  Elimina una categoría personalizada del usuario
// @Tags         categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path uint true "ID de la categoría"
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID de la categoría
	categoryID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de categoría inválido",
		})
		return
	}

	// Obtener la categoría
	category, err := h.categoryRepo.GetByID(uint(categoryID))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Code:    "CATEGORY_NOT_FOUND",
			Message: "Categoría no encontrada",
		})
		return
	}

	// Verificar que pertenece al usuario y no es del sistema
	if category.UserID == nil || *category.UserID != userID || category.IsDefault {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Code:    "FORBIDDEN",
			Message: "No puedes eliminar esta categoría",
		})
		return
	}

	// Verificar que no tiene gastos asociados
	hasExpenses, err := h.categoryRepo.HasExpenses(uint(categoryID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al verificar gastos asociados",
			Details: err.Error(),
		})
		return
	}

	if hasExpenses {
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Code:    "CATEGORY_HAS_EXPENSES",
			Message: "No se puede eliminar una categoría que tiene gastos asociados",
		})
		return
	}

	if err := h.categoryRepo.Delete(uint(categoryID)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al eliminar la categoría",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Categoría eliminada exitosamente",
		Data:    nil,
	})
}

// Helper function para mapear entidad a DTO
func mapCategoryToSummaryResponse(category *entity.Category) dto.CategorySummaryResponse {
	return dto.CategorySummaryResponse{
		ID:             category.ID,
		Name:           category.Name,
		Description:    category.Description,
		Icon:           category.Icon,
		Color:          category.Color,
		DisplayName:    category.GetDisplayName(),
		IsActive:       category.IsActive,
		IsDefault:      category.IsDefault,
		IsUserCategory: category.IsUserCategory(),
		SortOrder:      category.SortOrder,
		CanBeDeleted:   category.CanBeDeleted(),
	}
}
