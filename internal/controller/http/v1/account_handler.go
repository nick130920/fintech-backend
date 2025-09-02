package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/usecase"
	"github.com/nick130920/proyecto-fintech/pkg/validator"
)

// AccountHandler maneja las peticiones HTTP relacionadas con cuentas
type AccountHandler struct {
	accountUC *usecase.AccountUseCase
	validator *validator.Validator
}

// NewAccountHandler crea una nueva instancia de AccountHandler
func NewAccountHandler(accountUC *usecase.AccountUseCase) *AccountHandler {
	return &AccountHandler{
		accountUC: accountUC,
		validator: validator.New(),
	}
}

// GetAccounts obtiene todas las cuentas del usuario
func (h *AccountHandler) GetAccounts(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	accounts, err := h.accountUC.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get accounts",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

// CreateAccount crea una nueva cuenta
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	var req dto.CreateAccountRequest
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

	newAccount, err := h.accountUC.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to create account",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, newAccount)
}

// GetAccount obtiene una cuenta específica
func (h *AccountHandler) GetAccount(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid number",
		})
		return
	}

	account, err := h.accountUC.GetByID(userID, uint(accountID))
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Account not found",
				Message: "Account not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get account",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// UpdateAccount actualiza una cuenta
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid number",
		})
		return
	}

	var req dto.UpdateAccountRequest
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

	updatedAccount, err := h.accountUC.Update(userID, uint(accountID), &req)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Account not found",
				Message: "Account not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update account",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedAccount)
}

// DeleteAccount elimina una cuenta
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid number",
		})
		return
	}

	err = h.accountUC.Delete(userID, uint(accountID))
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "Account not found",
				Message: "Account not found",
			})
			return
		}

		if err.Error() == "cannot delete account with existing transactions" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "Cannot delete account",
				Message: "Account cannot be deleted because it has associated transactions",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to delete account",
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAccountSummaries obtiene resúmenes de cuentas
func (h *AccountHandler) GetAccountSummaries(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	summaries, err := h.accountUC.GetAccountSummaries(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get account summaries",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// GetTotalBalance obtiene el balance total
func (h *AccountHandler) GetTotalBalance(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	total, err := h.accountUC.GetTotalBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get total balance",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_balance": total})
}
