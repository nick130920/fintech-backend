package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/entity"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/validator"
)

// BankAccountHandler maneja las peticiones HTTP relacionadas con cuentas bancarias
type BankAccountHandler struct {
	bankAccountUC *usecase.BankAccountUseCase
	validator     *validator.Validator
}

// NewBankAccountHandler crea una nueva instancia de BankAccountHandler
func NewBankAccountHandler(bankAccountUC *usecase.BankAccountUseCase) *BankAccountHandler {
	return &BankAccountHandler{
		bankAccountUC: bankAccountUC,
		validator:     validator.New(),
	}
}

// CreateBankAccount maneja la creación de una nueva cuenta bancaria
// @Summary Crear cuenta bancaria
// @Description Crea una nueva cuenta bancaria para el usuario autenticado
// @Tags bank-accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param bankAccount body dto.CreateBankAccountRequest true "Datos de la cuenta bancaria"
// @Success 201 {object} dto.BankAccountResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts [post]
func (h *BankAccountHandler) CreateBankAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	var req dto.CreateBankAccountRequest
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

	response, err := h.bankAccountUC.CreateBankAccount(userID.(uint), &req)
	if err != nil {
		if err.Error() == "bank account with this number mask already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "Conflict",
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

// GetBankAccount obtiene una cuenta bancaria por ID
// @Summary Obtener cuenta bancaria
// @Description Obtiene los detalles de una cuenta bancaria específica
// @Tags bank-accounts
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de la cuenta bancaria"
// @Success 200 {object} dto.BankAccountResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{id} [get]
func (h *BankAccountHandler) GetBankAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	response, err := h.bankAccountUC.GetBankAccount(userID.(uint), uint(bankAccountID))
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

	c.JSON(http.StatusOK, response)
}

// GetUserBankAccounts obtiene todas las cuentas bancarias del usuario
// @Summary Listar cuentas bancarias
// @Description Obtiene todas las cuentas bancarias del usuario autenticado
// @Tags bank-accounts
// @Produce json
// @Security BearerAuth
// @Param active_only query bool false "Solo cuentas activas"
// @Success 200 {array} dto.BankAccountResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts [get]
func (h *BankAccountHandler) GetUserBankAccounts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	activeOnly := c.Query("active_only") == "true"

	response, err := h.bankAccountUC.GetUserBankAccounts(userID.(uint), activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetBankAccountsByType obtiene cuentas bancarias por tipo
// @Summary Listar cuentas bancarias por tipo
// @Description Obtiene cuentas bancarias filtradas por tipo
// @Tags bank-accounts
// @Produce json
// @Security BearerAuth
// @Param type path string true "Tipo de cuenta" Enums(checking, savings, credit, debit, investment)
// @Success 200 {array} dto.BankAccountResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/type/{type} [get]
func (h *BankAccountHandler) GetBankAccountsByType(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	accountType := entity.BankAccountType(c.Param("type"))

	// Validar tipo de cuenta
	validTypes := []entity.BankAccountType{
		entity.BankAccountTypeChecking,
		entity.BankAccountTypeSavings,
		entity.BankAccountTypeCredit,
		entity.BankAccountTypeDebit,
		entity.BankAccountTypeInvestment,
	}

	isValid := false
	for _, validType := range validTypes {
		if accountType == validType {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid account type",
			Message: "Account type must be one of: checking, savings, credit, debit, investment",
		})
		return
	}

	response, err := h.bankAccountUC.GetBankAccountsByType(userID.(uint), accountType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateBankAccount actualiza una cuenta bancaria
// @Summary Actualizar cuenta bancaria
// @Description Actualiza los detalles de una cuenta bancaria
// @Tags bank-accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de la cuenta bancaria"
// @Param bankAccount body dto.UpdateBankAccountRequest true "Datos a actualizar"
// @Success 200 {object} dto.BankAccountResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{id} [put]
func (h *BankAccountHandler) UpdateBankAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	var req dto.UpdateBankAccountRequest
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

	response, err := h.bankAccountUC.UpdateBankAccount(userID.(uint), uint(bankAccountID), &req)
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

	c.JSON(http.StatusOK, response)
}

// DeleteBankAccount elimina una cuenta bancaria
// @Summary Eliminar cuenta bancaria
// @Description Elimina una cuenta bancaria (soft delete)
// @Tags bank-accounts
// @Security BearerAuth
// @Param id path int true "ID de la cuenta bancaria"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{id} [delete]
func (h *BankAccountHandler) DeleteBankAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	err = h.bankAccountUC.DeleteBankAccount(userID.(uint), uint(bankAccountID))
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

	c.Status(http.StatusNoContent)
}

// SetBankAccountActive cambia el estado activo de una cuenta bancaria
// @Summary Cambiar estado activo
// @Description Activa o desactiva una cuenta bancaria
// @Tags bank-accounts
// @Accept json
// @Security BearerAuth
// @Param id path int true "ID de la cuenta bancaria"
// @Param status body dto.SetBankAccountActiveRequest true "Estado activo"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{id}/active [patch]
func (h *BankAccountHandler) SetBankAccountActive(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	var req dto.SetBankAccountActiveRequest
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

	err = h.bankAccountUC.SetBankAccountActive(userID.(uint), uint(bankAccountID), req.IsActive)
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

	c.Status(http.StatusNoContent)
}

// UpdateBankAccountBalance actualiza el balance de una cuenta bancaria
// @Summary Actualizar balance
// @Description Actualiza el balance de una cuenta bancaria
// @Tags bank-accounts
// @Accept json
// @Security BearerAuth
// @Param id path int true "ID de la cuenta bancaria"
// @Param balance body dto.UpdateBankAccountBalanceRequest true "Nuevo balance"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/{id}/balance [patch]
func (h *BankAccountHandler) UpdateBankAccountBalance(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Invalid bank account ID",
			Message: "Bank account ID must be a valid number",
		})
		return
	}

	var req dto.UpdateBankAccountBalanceRequest
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

	err = h.bankAccountUC.UpdateBankAccountBalance(userID.(uint), uint(bankAccountID), req.Balance)
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

	c.Status(http.StatusNoContent)
}

// GetBankAccountSummary obtiene un resumen de las cuentas bancarias
// @Summary Resumen de cuentas bancarias
// @Description Obtiene un resumen de todas las cuentas bancarias del usuario
// @Tags bank-accounts
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.BankAccountSummaryResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /bank-accounts/summary [get]
func (h *BankAccountHandler) GetBankAccountSummary(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "User ID not found in context",
		})
		return
	}

	response, err := h.bankAccountUC.GetBankAccountSummary(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal server error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
