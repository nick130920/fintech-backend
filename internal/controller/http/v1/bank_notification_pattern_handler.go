package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// BankNotificationPatternHandler maneja las peticiones HTTP relacionadas con patrones de notificación bancaria
type BankNotificationPatternHandler struct {
	patternUC *usecase.BankNotificationPatternUseCase
	validator *validator.Validator
}

// NewBankNotificationPatternHandler crea una nueva instancia de BankNotificationPatternHandler
func NewBankNotificationPatternHandler(patternUC *usecase.BankNotificationPatternUseCase) *BankNotificationPatternHandler {
	return &BankNotificationPatternHandler{
		patternUC: patternUC,
		validator: validator.New(),
	}
}

// CreatePattern maneja la creación de un nuevo patrón de notificación
// @Summary Crear patrón de notificación
// @Description Crea un nuevo patrón de notificación bancaria
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param pattern body dto.CreateBankNotificationPatternRequest true "Datos del patrón"
// @Success 201 {object} dto.BankNotificationPatternResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns [post]
func (h *BankNotificationPatternHandler) CreatePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	var req dto.CreateBankNotificationPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	response, err := h.patternUC.CreatePattern(userID.(uint), &req)
	if err != nil {
		if err.Error() == "bank account not found" || err.Error() == "unauthorized access to bank account" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetPattern obtiene un patrón de notificación por ID
// @Summary Obtener patrón de notificación
// @Description Obtiene los detalles de un patrón de notificación específico
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Success 200 {object} dto.BankNotificationPatternResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/{id} [get]
func (h *BankNotificationPatternHandler) GetPattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid pattern ID",
			Message: "Pattern ID must be a valid number",
		})
		return
	}

	response, err := h.patternUC.GetPattern(userID.(uint), uint(patternID))
	if err != nil {
		if err.Error() == "pattern not found" || err.Error() == "unauthorized access to pattern" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserPatterns obtiene todos los patrones del usuario
// @Summary Listar patrones de notificación
// @Description Obtiene todos los patrones de notificación del usuario autenticado
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.BankNotificationPatternResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns [get]
func (h *BankNotificationPatternHandler) GetUserPatterns(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	response, err := h.patternUC.GetUserPatterns(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetBankAccountPatterns obtiene patrones de una cuenta bancaria específica
// @Summary Listar patrones por cuenta bancaria
// @Description Obtiene todos los patrones de notificación de una cuenta bancaria
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Param bank_account_id path int true "ID de la cuenta bancaria"
// @Param active_only query bool false "Solo patrones activos"
// @Success 200 {array} dto.BankNotificationPatternResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{bank_account_id}/notification-patterns [get]
func (h *BankNotificationPatternHandler) GetBankAccountPatterns(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("bank_account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	activeOnly := c.Query("active_only") == "true"

	response, err := h.patternUC.GetBankAccountPatterns(userID.(uint), uint(bankAccountID), activeOnly)
	if err != nil {
		if err.Error() == "unauthorized access to bank account" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePattern actualiza un patrón de notificación
// @Summary Actualizar patrón de notificación
// @Description Actualiza los detalles de un patrón de notificación
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Param pattern body dto.UpdateBankNotificationPatternRequest true "Datos a actualizar"
// @Success 200 {object} dto.BankNotificationPatternResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/{id} [put]
func (h *BankNotificationPatternHandler) UpdatePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid pattern ID",
			Message: "Pattern ID must be a valid number",
		})
		return
	}

	var req dto.UpdateBankNotificationPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	response, err := h.patternUC.UpdatePattern(userID.(uint), uint(patternID), &req)
	if err != nil {
		if err.Error() == "pattern not found" || err.Error() == "unauthorized access to pattern" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeletePattern elimina un patrón de notificación
// @Summary Eliminar patrón de notificación
// @Description Elimina un patrón de notificación (soft delete)
// @Tags notification-patterns
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/{id} [delete]
func (h *BankNotificationPatternHandler) DeletePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid pattern ID",
			Message: "Pattern ID must be a valid number",
		})
		return
	}

	err = h.patternUC.DeletePattern(userID.(uint), uint(patternID))
	if err != nil {
		if err.Error() == "pattern not found" || err.Error() == "unauthorized access to pattern" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// SetPatternStatus cambia el estado de un patrón
// @Summary Cambiar estado del patrón
// @Description Cambia el estado de un patrón de notificación (active, inactive, learning)
// @Tags notification-patterns
// @Accept json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Param status body dto.SetPatternStatusRequest true "Nuevo estado"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/{id}/status [patch]
func (h *BankNotificationPatternHandler) SetPatternStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid pattern ID",
			Message: "Pattern ID must be a valid number",
		})
		return
	}

	var req dto.SetPatternStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	err = h.patternUC.SetPatternStatus(userID.(uint), uint(patternID), req.Status)
	if err != nil {
		if err.Error() == "pattern not found" || err.Error() == "unauthorized access to pattern" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ProcessNotification procesa una notificación bancaria
// @Summary Procesar notificación bancaria
// @Description Procesa una notificación bancaria usando patrones de reconocimiento
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param notification body dto.ProcessNotificationRequest true "Datos de la notificación"
// @Success 200 {object} dto.ProcessedNotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/process [post]
func (h *BankNotificationPatternHandler) ProcessNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	var req dto.ProcessNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	if err := h.validator.Validate(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
		})
		return
	}

	response, err := h.patternUC.ProcessNotification(userID.(uint), req.BankAccountID, req.Channel, req.Message)
	if err != nil {
		if err.Error() == "unauthorized access to bank account" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Not found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPatternStatistics obtiene estadísticas de patrones
// @Summary Estadísticas de patrones
// @Description Obtiene estadísticas generales de los patrones de notificación del usuario
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.PatternStatisticsResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /notification-patterns/statistics [get]
func (h *BankNotificationPatternHandler) GetPatternStatistics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	response, err := h.patternUC.GetPatternStatistics(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
