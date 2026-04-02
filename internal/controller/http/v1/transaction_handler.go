package v1

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// TransactionHandler maneja las peticiones HTTP relacionadas con transacciones
type TransactionHandler struct {
	transactionUC *usecase.TransactionUseCase
	validator     *validator.Validator
	logger        zerolog.Logger
}

// NewTransactionHandler crea una nueva instancia de TransactionHandler
func NewTransactionHandler(transactionUC *usecase.TransactionUseCase, logger zerolog.Logger) *TransactionHandler {
	return &TransactionHandler{
		transactionUC: transactionUC,
		validator:     validator.New(),
		logger:        logger,
	}
}

// GetTransactions godoc
// @Summary Listar transacciones
// @Description Obtiene transacciones del usuario autenticado con filtros avanzados (fechas, montos, categoría, búsqueda y paginación)
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param account_id query int false "ID de cuenta"
// @Param type query string false "Tipo (income,expense,transfer)"
// @Param status query string false "Estado (pending,completed,cancelled)"
// @Param category_id query int false "ID de categoría"
// @Param category_ids query string false "IDs de categorías separados por coma (ej: 1,2,3)"
// @Param from_date query string false "Fecha inicio (YYYY-MM-DD)"
// @Param to_date query string false "Fecha fin (YYYY-MM-DD)"
// @Param amount_min query number false "Monto mínimo"
// @Param amount_max query number false "Monto máximo"
// @Param search query string false "Texto a buscar en descripción"
// @Param limit query int false "Límite de resultados"
// @Param offset query int false "Offset de resultados"
// @Success 200 {array} entity.TransactionSummary
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions [get]
func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	// Construir filtros desde query parameters
	filter := h.buildFilterFromQuery(c)

	transactions, err := h.transactionUC.GetByUserID(userID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get transactions",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// CreateTransaction godoc
// @Summary Crear transacción
// @Description Crea una nueva transacción para el usuario autenticado
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param transaction body dto.CreateTransactionRequest true "Datos de transacción"
// @Success 201 {object} entity.Transaction
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions [post]
func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	var req dto.CreateTransactionRequest
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

	newTransaction, err := h.transactionUC.Create(userID, &req)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Invalid account",
				Message: "Account not found or does not belong to user",
			})
			return
		}

		if err.Error() == "insufficient funds" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Insufficient funds",
				Message: "Not enough balance for this transaction",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to create transaction",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, newTransaction)
}

// GetTransaction godoc
// @Summary Obtener transacción
// @Description Obtiene una transacción específica del usuario autenticado
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de transacción"
// @Success 200 {object} entity.Transaction
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions/{id} [get]
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid transaction ID",
			Message: "Transaction ID must be a valid number",
		})
		return
	}

	transaction, err := h.transactionUC.GetByID(userID, uint(transactionID))
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Transaction not found",
				Message: "Transaction not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get transaction",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// UpdateTransaction godoc
// @Summary Actualizar transacción
// @Description Actualiza una transacción del usuario autenticado
// @Tags transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de transacción"
// @Param transaction body dto.UpdateTransactionRequest true "Datos a actualizar"
// @Success 200 {object} entity.Transaction
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions/{id} [put]
func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid transaction ID",
			Message: "Transaction ID must be a valid number",
		})
		return
	}

	var req dto.UpdateTransactionRequest
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

	updatedTransaction, err := h.transactionUC.Update(userID, uint(transactionID), &req)
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Transaction not found",
				Message: "Transaction not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update transaction",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedTransaction)
}

// DeleteTransaction godoc
// @Summary Eliminar transacción
// @Description Elimina una transacción del usuario autenticado
// @Tags transactions
// @Security BearerAuth
// @Param id path int true "ID de transacción"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions/{id} [delete]
func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid transaction ID",
			Message: "Transaction ID must be a valid number",
		})
		return
	}

	err = h.transactionUC.Delete(userID, uint(transactionID))
	if err != nil {
		if err.Error() == "transaction not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Transaction not found",
				Message: "Transaction not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to delete transaction",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// CancelTransaction godoc
// @Summary Cancelar transacción
// @Description Cancela una transacción del usuario autenticado
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de transacción"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/v1/transactions/{id}/cancel [patch]
func (h *TransactionHandler) CancelTransaction(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid transaction ID",
			Message: "Transaction ID must be a valid number",
		})
		return
	}

	err = h.transactionUC.Cancel(userID, uint(transactionID))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Failed to cancel transaction",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Transaction cancelled successfully",
	})
}

// GetRecentTransactions godoc
// @Summary Transacciones recientes
// @Description Obtiene las transacciones más recientes del usuario autenticado
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Límite de resultados"
// @Success 200 {array} entity.TransactionSummary
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions/recent [get]
func (h *TransactionHandler) GetRecentTransactions(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	transactions, err := h.transactionUC.GetRecentTransactions(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get recent transactions",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// GetTotalsByType godoc
// @Summary Totales por tipo
// @Description Obtiene totales de transacciones por tipo para el usuario autenticado
// @Tags transactions
// @Produce json
// @Security BearerAuth
// @Param from_date query string false "Fecha inicio (YYYY-MM-DD)"
// @Param to_date query string false "Fecha fin (YYYY-MM-DD)"
// @Success 200 {object} map[string]number
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/transactions/totals-by-type [get]
func (h *TransactionHandler) GetTotalsByType(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	var fromDate, toDate *time.Time

	if fromStr := c.Query("from_date"); fromStr != "" {
		if from, err := time.Parse("2006-01-02", fromStr); err == nil {
			fromDate = &from
		}
	}

	if toStr := c.Query("to_date"); toStr != "" {
		if to, err := time.Parse("2006-01-02", toStr); err == nil {
			toDate = &to
		}
	}

	totals, err := h.transactionUC.GetUserTotalsByType(userID, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get totals by type",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, totals)
}

// buildFilterFromQuery construye un filtro desde los query parameters
func (h *TransactionHandler) buildFilterFromQuery(c *gin.Context) *entity.TransactionFilter {
	filter := &entity.TransactionFilter{
		Limit:  50, // valor por defecto
		Offset: 0,  // valor por defecto
	}

	// Account ID
	if accountIDStr := c.Query("account_id"); accountIDStr != "" {
		if accountID, err := strconv.ParseUint(accountIDStr, 10, 32); err == nil {
			accountIDUint := uint(accountID)
			filter.AccountID = &accountIDUint
		}
	}

	// Type
	if typeStr := c.Query("type"); typeStr != "" {
		transType := entity.TransactionType(typeStr)
		filter.Type = &transType
	}

	// Status
	if statusStr := c.Query("status"); statusStr != "" {
		transStatus := entity.TransactionStatus(statusStr)
		filter.Status = &transStatus
	}

	// Category ID (single)
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32); err == nil {
			categoryIDUint := uint(categoryID)
			filter.CategoryID = &categoryIDUint
		}
	}

	// Category IDs (multiple)
	if categoryIDsStr := c.Query("category_ids"); categoryIDsStr != "" {
		parts := strings.Split(categoryIDsStr, ",")
		for _, part := range parts {
			v := strings.TrimSpace(part)
			if v == "" {
				continue
			}
			if id, err := strconv.ParseUint(v, 10, 32); err == nil {
				filter.CategoryIDs = append(filter.CategoryIDs, uint(id))
			}
		}
	}

	// Date range
	if fromDateStr := c.Query("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			filter.FromDate = &fromDate
		}
	}
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if fromDate, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.FromDate = &fromDate
		}
	}
	if toDateStr := c.Query("to_date"); toDateStr != "" {
		if toDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
			filter.ToDate = &toDate
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if toDate, err := time.Parse("2006-01-02", dateToStr); err == nil {
			filter.ToDate = &toDate
		}
	}

	// Amount range
	if minAmountStr := c.Query("amount_min"); minAmountStr != "" {
		if minAmount, err := strconv.ParseFloat(minAmountStr, 64); err == nil {
			filter.MinAmount = &minAmount
		}
	}
	if maxAmountStr := c.Query("amount_max"); maxAmountStr != "" {
		if maxAmount, err := strconv.ParseFloat(maxAmountStr, 64); err == nil {
			filter.MaxAmount = &maxAmount
		}
	}

	// Limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	// Offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Search
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}

	return filter
}
