package v1

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
)

// IncomeHandler maneja las peticiones HTTP relacionadas con ingresos
type IncomeHandler struct {
	incomeUC *usecase.IncomeUseCase
}

// NewIncomeHandler crea un nuevo IncomeHandler
func NewIncomeHandler(incomeUC *usecase.IncomeUseCase) *IncomeHandler {
	return &IncomeHandler{
		incomeUC: incomeUC,
	}
}

// CreateIncome godoc
// @Summary Crear un nuevo ingreso
// @Description Crea un nuevo ingreso para el usuario autenticado
// @Tags incomes
// @Accept json
// @Produce json
// @Param income body dto.CreateIncomeRequest true "Datos del ingreso"
// @Success 201 {object} dto.Response{data=dto.IncomeResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes [post]
func (h *IncomeHandler) CreateIncome(c *gin.Context) {
	userID := getUserID(c)

	var req dto.CreateIncomeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[WARN] ❌ Error al parsear request CreateIncome: %v | UserID: %d", err, userID)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Datos de ingreso inválidos: " + err.Error(),
		})
		return
	}

	log.Printf("[DEBUG] 📥 CreateIncome Request: UserID=%d, Monto=%.2f, Fuente=%s", userID, req.Amount, req.Source)

	income, err := h.incomeUC.CreateIncome(userID, &req)
	if err != nil {
		log.Printf("[WARN] ❌ Error al crear ingreso: %v | Usuario: %d | Request: %+v", err, userID, req)

		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "USER_NOT_FOUND",
				Message: "Usuario no encontrado",
			})
		case "user account is not active":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "ACCOUNT_INACTIVE",
				Message: "Cuenta de usuario inactiva",
			})
		default:
			if contains(err.Error(), "invalid date format") {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{
					Code:    "INVALID_DATE",
					Message: "Formato de fecha inválido",
				})
			} else {
				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Code:    "INTERNAL_ERROR",
					Message: "Error interno del servidor",
				})
			}
		}
		return
	}

	log.Printf("[INFO] ✅ Ingreso creado exitosamente: ID=%d, UserID=%d, Monto=%.2f", income.ID, userID, income.Amount)

	c.JSON(http.StatusCreated, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingreso creado exitosamente",
		Data:    income,
	})
}

// GetIncomes godoc
// @Summary Obtener ingresos del usuario
// @Description Obtiene la lista de ingresos del usuario con filtros opcionales
// @Tags incomes
// @Accept json
// @Produce json
// @Param start_date query string false "Fecha de inicio (YYYY-MM-DD)"
// @Param end_date query string false "Fecha de fin (YYYY-MM-DD)"
// @Param source query string false "Fuente de ingreso" Enums(salary, freelance, investment, business, rental, bonus, gift, other)
// @Param limit query int false "Límite de resultados" default(10)
// @Param offset query int false "Offset de resultados" default(0)
// @Success 200 {object} dto.Response{data=[]dto.IncomeSummaryResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes [get]
func (h *IncomeHandler) GetIncomes(c *gin.Context) {
	userID := getUserID(c)

	// Parsear parámetros de consulta
	var startDate, endDate *time.Time
	var source *entity.IncomeSource

	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &parsed
		}
	}

	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = &parsed
		}
	}

	if sourceStr := c.Query("source"); sourceStr != "" {
		s := entity.IncomeSource(sourceStr)
		source = &s
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	incomes, err := h.incomeUC.GetIncomes(userID, startDate, endDate, source, limit, offset)
	if err != nil {
		log.Printf("❌ Error al obtener ingresos: %v | UserID: %d", err, userID)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error interno del servidor",
		})
		return
	}

	log.Printf("✅ Ingresos obtenidos: %d resultados | UserID: %d", len(incomes), userID)

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingresos obtenidos exitosamente",
		Data:    incomes,
	})
}

// GetIncome godoc
// @Summary Obtener ingreso por ID
// @Description Obtiene los detalles de un ingreso específico
// @Tags incomes
// @Accept json
// @Produce json
// @Param id path int true "ID del ingreso"
// @Success 200 {object} dto.Response{data=dto.IncomeResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/{id} [get]
func (h *IncomeHandler) GetIncome(c *gin.Context) {
	userID := getUserID(c)

	incomeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de ingreso inválido",
		})
		return
	}

	income, err := h.incomeUC.GetIncomeByID(userID, uint(incomeID))
	if err != nil {
		log.Printf("❌ Error al obtener ingreso: %v | UserID: %d | IncomeID: %d", err, userID, incomeID)

		if err.Error() == "income not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "INCOME_NOT_FOUND",
				Message: "Ingreso no encontrado",
			})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Error interno del servidor",
			})
		}
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingreso obtenido exitosamente",
		Data:    income,
	})
}

// UpdateIncome godoc
// @Summary Actualizar un ingreso
// @Description Actualiza los datos de un ingreso existente
// @Tags incomes
// @Accept json
// @Produce json
// @Param id path int true "ID del ingreso"
// @Param income body dto.UpdateIncomeRequest true "Datos actualizados del ingreso"
// @Success 200 {object} dto.Response{data=dto.IncomeResponse}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/{id} [put]
func (h *IncomeHandler) UpdateIncome(c *gin.Context) {
	userID := getUserID(c)

	incomeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de ingreso inválido",
		})
		return
	}

	var req dto.UpdateIncomeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Error al parsear request UpdateIncome: %v | UserID: %d", err, userID)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Datos de actualización inválidos: " + err.Error(),
		})
		return
	}

	income, err := h.incomeUC.UpdateIncome(userID, uint(incomeID), &req)
	if err != nil {
		log.Printf("❌ Error al actualizar ingreso: %v | UserID: %d | IncomeID: %d", err, userID, incomeID)

		switch err.Error() {
		case "income not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "INCOME_NOT_FOUND",
				Message: "Ingreso no encontrado",
			})
		case "income cannot be modified":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "CANNOT_MODIFY",
				Message: "El ingreso no puede ser modificado",
			})
		default:
			if contains(err.Error(), "invalid date format") {
				c.JSON(http.StatusBadRequest, dto.ErrorResponse{
					Code:    "INVALID_DATE",
					Message: "Formato de fecha inválido",
				})
			} else {
				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
					Code:    "INTERNAL_ERROR",
					Message: "Error interno del servidor",
				})
			}
		}
		return
	}

	log.Printf("✅ Ingreso actualizado exitosamente: ID=%d, UserID=%d", incomeID, userID)

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingreso actualizado exitosamente",
		Data:    income,
	})
}

// DeleteIncome godoc
// @Summary Eliminar un ingreso
// @Description Elimina un ingreso existente
// @Tags incomes
// @Accept json
// @Produce json
// @Param id path int true "ID del ingreso"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/{id} [delete]
func (h *IncomeHandler) DeleteIncome(c *gin.Context) {
	userID := getUserID(c)

	incomeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:    "INVALID_ID",
			Message: "ID de ingreso inválido",
		})
		return
	}

	err = h.incomeUC.DeleteIncome(userID, uint(incomeID))
	if err != nil {
		log.Printf("❌ Error al eliminar ingreso: %v | UserID: %d | IncomeID: %d", err, userID, incomeID)

		switch err.Error() {
		case "income not found":
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Code:    "INCOME_NOT_FOUND",
				Message: "Ingreso no encontrado",
			})
		case "income cannot be deleted":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Code:    "CANNOT_DELETE",
				Message: "El ingreso no puede ser eliminado",
			})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Code:    "INTERNAL_ERROR",
				Message: "Error interno del servidor",
			})
		}
		return
	}

	log.Printf("✅ Ingreso eliminado exitosamente: ID=%d, UserID=%d", incomeID, userID)

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingreso eliminado exitosamente",
	})
}

// GetIncomeStats godoc
// @Summary Obtener estadísticas de ingresos
// @Description Obtiene estadísticas detalladas de los ingresos del usuario
// @Tags incomes
// @Accept json
// @Produce json
// @Param year query int false "Año para las estadísticas" default(current year)
// @Success 200 {object} dto.Response{data=dto.IncomeStatsResponse}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/stats [get]
func (h *IncomeHandler) GetIncomeStats(c *gin.Context) {
	userID := getUserID(c)

	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		if parsed, err := strconv.Atoi(yearStr); err == nil {
			year = &parsed
		}
	}

	stats, err := h.incomeUC.GetIncomeStats(userID, year)
	if err != nil {
		log.Printf("❌ Error al obtener estadísticas de ingresos: %v | UserID: %d", err, userID)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error interno del servidor",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Estadísticas obtenidas exitosamente",
		Data:    stats,
	})
}

// GetRecentIncomes godoc
// @Summary Obtener ingresos recientes
// @Description Obtiene los ingresos más recientes del usuario
// @Tags incomes
// @Accept json
// @Produce json
// @Param limit query int false "Límite de resultados" default(10)
// @Success 200 {object} dto.Response{data=[]dto.IncomeSummaryResponse}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/recent [get]
func (h *IncomeHandler) GetRecentIncomes(c *gin.Context) {
	userID := getUserID(c)

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	incomes, err := h.incomeUC.GetRecentIncomes(userID, limit)
	if err != nil {
		log.Printf("❌ Error al obtener ingresos recientes: %v | UserID: %d", err, userID)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error interno del servidor",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Ingresos recientes obtenidos exitosamente",
		Data:    incomes,
	})
}

// ProcessRecurringIncomes godoc
// @Summary Procesar ingresos recurrentes
// @Description Procesa y genera ingresos recurrentes pendientes
// @Tags incomes
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.RecurringIncomeProcessResponse}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security Bearer
// @Router /api/v1/incomes/process-recurring [post]
func (h *IncomeHandler) ProcessRecurringIncomes(c *gin.Context) {
	userID := getUserID(c)

	result, err := h.incomeUC.ProcessRecurringIncomes(userID)
	if err != nil {
		log.Printf("❌ Error al procesar ingresos recurrentes: %v | UserID: %d", err, userID)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "Error interno del servidor",
		})
		return
	}

	log.Printf("✅ Ingresos recurrentes procesados: %d | UserID: %d", result.ProcessedCount, userID)

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: result.Message,
		Data:    result,
	})
}

// Helper functions

func getUserID(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}

	switch v := userID.(type) {
	case uint:
		return v
	case int:
		return uint(v)
	case float64:
		return uint(v)
	default:
		return 0
	}
}

func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 &&
		(str == substr || (len(str) >= len(substr) &&
			str[0:len(substr)] == substr))
}
