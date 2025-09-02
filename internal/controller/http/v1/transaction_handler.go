package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// TransactionHandler maneja las peticiones HTTP relacionadas con transacciones
type TransactionHandler struct {
	transactionUC *usecase.TransactionUseCase
	validator     *validator.Validator
}

// NewTransactionHandler crea una nueva instancia de TransactionHandler
func NewTransactionHandler(transactionUC *usecase.TransactionUseCase) *TransactionHandler {
	return &TransactionHandler{
		transactionUC: transactionUC,
		validator:     validator.New(),
	}
}

// GetTransactions obtiene las transacciones del usuario con filtros
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

// CreateTransaction crea una nueva transacción
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

// GetTransaction obtiene una transacción específica
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

// UpdateTransaction actualiza una transacción
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

// DeleteTransaction elimina una transacción
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

// CancelTransaction cancela una transacción
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

// GetRecentTransactions obtiene transacciones recientes
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

// GetTotalsByType obtiene totales por tipo de transacción
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
