package v1

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// ExpenseHandler maneja las peticiones HTTP relacionadas con gastos
type ExpenseHandler struct {
	expenseUC *usecase.ExpenseUseCase
	validator *validator.Validator
}

// NewExpenseHandler crea una nueva instancia de ExpenseHandler
func NewExpenseHandler(expenseUC *usecase.ExpenseUseCase) *ExpenseHandler {
	return &ExpenseHandler{
		expenseUC: expenseUC,
		validator: validator.New(),
	}
}

// CreateExpense godoc
// @Summary      Crear gasto manual
// @Description  Crea un nuevo gasto manual con categoría, monto y fecha
// @Tags         expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateExpenseRequest true "Datos del gasto"
// @Success      201  {object}  dto.Response{data=dto.ExpenseSummaryResponse}
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/expenses [post]
func (h *ExpenseHandler) CreateExpense(c *gin.Context) {
	var req dto.CreateExpenseRequest

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

	// Obtener ID del usuario del contexto
	userID := getUserIDFromContext(c)

	// Crear gasto con logging detallado
	log.Printf("🔍 Creando gasto para usuario %d: %+v", userID, req)

	expense, err := h.expenseUC.CreateExpense(userID, &req)
	if err != nil {
		// Log detallado del error
		log.Printf("❌ Error al crear gasto: %v | Usuario: %d | Request: %+v", err, userID, req)

		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "USER_NOT_FOUND",
				Message: "Usuario no encontrado",
			})
		case "user account is not active":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "ACCOUNT_INACTIVE",
				Message: "La cuenta de usuario no está activa",
			})
		case "category not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "CATEGORY_NOT_FOUND",
				Message: "Categoría no encontrada",
			})
		case "budget not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "BUDGET_NOT_FOUND",
				Message: "No hay presupuesto activo para registrar el gasto",
			})
		case "category not allocated in budget":
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Code:    "CATEGORY_NOT_ALLOCATED",
				Message: "La categoría no está asignada en el presupuesto actual",
			})
		case "invalid date format":
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Code:    "INVALID_DATE",
				Message: "Formato de fecha inválido. Use YYYY-MM-DD o RFC3339",
			})
		default:
			// Log de error desconocido
			log.Printf("🚨 Error no manejado en CreateExpense: %v", err)
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Error interno del servidor",
				Details: err.Error(),
			})
		}
		return
	}

	log.Printf("✅ Gasto creado exitosamente: ID=%d, Monto=%.2f", expense.ID, req.Amount)

	c.JSON(http.StatusCreated, dto.Response{
		Code:    "SUCCESS",
		Message: "Gasto registrado exitosamente",
		Data:    expense,
	})
}

// GetExpenses godoc
// @Summary      Obtener historial de gastos
// @Description  Obtiene el historial de gastos del usuario con filtros opcionales
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        category_id query int    false "ID de categoría para filtrar"
// @Param        start_date  query string false "Fecha de inicio (YYYY-MM-DD)"
// @Param        end_date    query string false "Fecha de fin (YYYY-MM-DD)"
// @Param        limit       query int    false "Número máximo de resultados (default: 50)"
// @Param        offset      query int    false "Número de resultados a omitir (default: 0)"
// @Success      200  {object}  dto.Response{data=[]dto.ExpenseSummaryResponse}
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/expenses [get]
func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener parámetros de query
	var categoryID *uint
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if id, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			categoryIDUint := uint(id)
			categoryID = &categoryIDUint
		}
	}

	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if date, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &date
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if date, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &date
		}
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	expenses, err := h.expenseUC.GetExpenses(userID, categoryID, startDate, endDate, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al obtener gastos",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Gastos obtenidos exitosamente",
		Data:    expenses,
	})
}

// GetExpensesByCategory godoc
// @Summary      Obtener gastos por categoría
// @Description  Obtiene gastos agrupados por categoría para el mes actual
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=map[string]interface{}}
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/expenses/by-category [get]
func (h *ExpenseHandler) GetExpensesByCategory(c *gin.Context) {
	userID := getUserIDFromContext(c)

	summary, err := h.expenseUC.GetExpensesByCategory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al obtener resumen por categoría",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Resumen por categoría obtenido exitosamente",
		Data:    summary,
	})
}

// GetRecentExpenses godoc
// @Summary      Obtener gastos recientes
// @Description  Obtiene los gastos más recientes del usuario
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Número de gastos a obtener (default: 10)"
// @Success      200  {object}  dto.Response{data=[]dto.ExpenseSummaryResponse}
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/expenses/recent [get]
func (h *ExpenseHandler) GetRecentExpenses(c *gin.Context) {
	userID := getUserIDFromContext(c)

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	expenses, err := h.expenseUC.GetRecentExpenses(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error al obtener gastos recientes",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Gastos recientes obtenidos exitosamente",
		Data:    expenses,
	})
}

// UpdateExpense godoc
// @Summary      Actualizar gasto
// @Description  Actualiza un gasto existente
// @Tags         expenses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      uint                        true  "ID del gasto"
// @Param        request body      dto.UpdateExpenseRequest    true  "Datos de actualización"
// @Success      200     {object}  dto.Response{data=dto.ExpenseSummaryResponse}
// @Failure      400     {object}  dto.ErrorResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/v1/expenses/{id} [put]
func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID del gasto
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de gasto inválido",
		})
		return
	}

	var req dto.UpdateExpenseRequest
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

	expense, err := h.expenseUC.UpdateExpense(userID, uint(expenseID), &req)
	if err != nil {
		switch err.Error() {
		case "expense not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "EXPENSE_NOT_FOUND",
				Message: "Gasto no encontrado",
			})
		case "expense cannot be modified":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "EXPENSE_READONLY",
				Message: "Este gasto no puede ser modificado",
			})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Error interno del servidor",
				Details: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Gasto actualizado exitosamente",
		Data:    expense,
	})
}

// DeleteExpense godoc
// @Summary      Eliminar gasto
// @Description  Elimina un gasto existente
// @Tags         expenses
// @Produce      json
// @Security     BearerAuth
// @Param        id path uint true "ID del gasto"
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/expenses/{id} [delete]
func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID del gasto
	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de gasto inválido",
		})
		return
	}

	err = h.expenseUC.DeleteExpense(userID, uint(expenseID))
	if err != nil {
		switch err.Error() {
		case "expense not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "EXPENSE_NOT_FOUND",
				Message: "Gasto no encontrado",
			})
		case "expense cannot be deleted":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "EXPENSE_READONLY",
				Message: "Este gasto no puede ser eliminado",
			})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Error interno del servidor",
				Details: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Gasto eliminado exitosamente",
		Data:    nil,
	})
}
