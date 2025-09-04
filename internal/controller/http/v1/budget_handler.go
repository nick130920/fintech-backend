package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// BudgetHandler maneja las peticiones HTTP relacionadas con presupuestos
type BudgetHandler struct {
	budgetUC  *usecase.BudgetUseCase
	validator *validator.Validator
}

// NewBudgetHandler crea una nueva instancia de BudgetHandler
func NewBudgetHandler(budgetUC *usecase.BudgetUseCase) *BudgetHandler {
	return &BudgetHandler{
		budgetUC:  budgetUC,
		validator: validator.New(),
	}
}

// CreateBudget godoc
// @Summary      Crear presupuesto mensual
// @Description  Crea un nuevo presupuesto mensual con asignaciones por categoría
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateBudgetRequest true "Datos del presupuesto"
// @Success      201  {object}  dto.Response{data=dto.BudgetSummaryResponse}
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/budgets [post]
func (h *BudgetHandler) CreateBudget(c *gin.Context) {
	var req dto.CreateBudgetRequest

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

	// Crear presupuesto
	budget, err := h.budgetUC.CreateBudget(userID, &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Code:    "SUCCESS",
		Message: "Presupuesto creado exitosamente",
		Data:    budget,
	})
}

// GetCurrentBudget godoc
// @Summary      Obtener presupuesto actual
// @Description  Obtiene el presupuesto del mes actual del usuario
// @Tags         budgets
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.BudgetSummaryResponse}
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/current [get]
func (h *BudgetHandler) GetCurrentBudget(c *gin.Context) {
	userID := getUserIDFromContext(c)

	budget, err := h.budgetUC.GetCurrentBudget(userID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Presupuesto obtenido exitosamente",
		Data:    budget,
	})
}

// GetBudgetByMonth godoc
// @Summary      Obtener presupuesto por mes
// @Description  Obtiene el presupuesto de un mes específico
// @Tags         budgets
// @Produce      json
// @Security     BearerAuth
// @Param        year  query     int  true  "Año (ej: 2024)"
// @Param        month query     int  true  "Mes (1-12)"
// @Success      200  {object}  dto.Response{data=dto.BudgetSummaryResponse}
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/month [get]
func (h *BudgetHandler) GetBudgetByMonth(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener parámetros de query
	yearStr := c.Query("year")
	monthStr := c.Query("month")

	if yearStr == "" || monthStr == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "MISSING_PARAMETERS",
			Message: "Se requieren los parámetros year y month",
		})
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2020 || year > 2030 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_YEAR",
			Message: "Año inválido. Debe ser un número entre 2020 y 2030",
		})
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_MONTH",
			Message: "Mes inválido. Debe ser un número entre 1 y 12",
		})
		return
	}

	budget, err := h.budgetUC.GetBudgetByMonth(userID, year, month)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Presupuesto obtenido exitosamente",
		Data:    budget,
	})
}

// UpdateBudget godoc
// @Summary      Actualizar presupuesto
// @Description  Actualiza un presupuesto existente
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      uint                        true  "ID del presupuesto"
// @Param        request body      dto.UpdateBudgetRequest     true  "Datos de actualización"
// @Success      200     {object}  dto.Response{data=dto.BudgetSummaryResponse}
// @Failure      400     {object}  dto.ErrorResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/{id} [put]
func (h *BudgetHandler) UpdateBudget(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID del presupuesto
	budgetID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de presupuesto inválido",
		})
		return
	}

	var req dto.UpdateBudgetRequest
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

	budget, err := h.budgetUC.UpdateBudget(userID, uint(budgetID), &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Presupuesto actualizado exitosamente",
		Data:    budget,
	})
}

// GetDashboard godoc
// @Summary      Obtener dashboard principal
// @Description  Obtiene el dashboard con información del presupuesto actual, gastos y alertas
// @Tags         budgets
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response{data=dto.BudgetDashboardResponse}
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/dashboard [get]
func (h *BudgetHandler) GetDashboard(c *gin.Context) {
	userID := getUserIDFromContext(c)

	dashboard, err := h.budgetUC.GetDashboard(userID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Dashboard obtenido exitosamente",
		Data:    dashboard,
	})
}

// ProcessDailyRollover godoc
// @Summary      Procesar rollover diario
// @Description  Procesa el rollover diario de saldos no gastados (endpoint interno/cron)
// @Tags         budgets
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.Response
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/rollover [post]
func (h *BudgetHandler) ProcessDailyRollover(c *gin.Context) {
	userID := getUserIDFromContext(c)

	err := h.budgetUC.ProcessDailyRollover(userID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Rollover diario procesado exitosamente",
		Data:    nil,
	})
}

// UpdateAllocation godoc
// @Summary      Actualizar asignación de presupuesto
// @Description  Actualiza una asignación individual de presupuesto por categoría
// @Tags         budgets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      uint                               true  "ID de la asignación"
// @Param        request body      dto.UpdateSingleAllocationRequest true  "Datos de actualización"
// @Success      200     {object}  dto.Response{data=dto.AllocationSummaryResponse}
// @Failure      400     {object}  dto.ErrorResponse
// @Failure      401     {object}  dto.ErrorResponse
// @Failure      404     {object}  dto.ErrorResponse
// @Failure      500     {object}  dto.ErrorResponse
// @Router       /api/v1/budgets/allocations/{id} [put]
func (h *BudgetHandler) UpdateAllocation(c *gin.Context) {
	userID := getUserIDFromContext(c)

	// Obtener ID de la asignación
	allocationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de asignación inválido",
		})
		return
	}

	var req dto.UpdateSingleAllocationRequest
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

	allocation, err := h.budgetUC.UpdateAllocation(userID, uint(allocationID), &req)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Asignación actualizada exitosamente",
		Data:    allocation,
	})
}

// Helper function para obtener el ID del usuario del contexto JWT
func getUserIDFromContext(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		// En caso de que no exista (no debería pasar con el middleware correcto)
		return 0
	}

	// Convertir a uint
	if id, ok := userID.(uint); ok {
		return id
	}

	// Si viene como float64 (típico de JWT claims)
	if id, ok := userID.(float64); ok {
		return uint(id)
	}

	return 0
}
