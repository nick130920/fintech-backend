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

func paginateAccounts(items []*entity.Account, offset, limit int) []*entity.Account {
	if offset >= len(items) {
		return []*entity.Account{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

// AccountHandler maneja las peticiones HTTP relacionadas con cuentas
type AccountHandler struct {
	accountUC *usecase.AccountUseCase
	validator *validator.Validator
}

type totalBalanceResponse struct {
	TotalBalance float64 `json:"total_balance"`
}

// NewAccountHandler crea una nueva instancia de AccountHandler
func NewAccountHandler(accountUC *usecase.AccountUseCase) *AccountHandler {
	return &AccountHandler{
		accountUC: accountUC,
		validator: validator.New(),
	}
}

// GetAccounts godoc
// @Summary Obtener cuentas
// @Description Obtiene las cuentas del usuario autenticado. Si envías page/per_page retorna respuesta paginada.
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param page query int false "Página (opcional para paginación)"
// @Param per_page query int false "Elementos por página (opcional para paginación)"
// @Success 200 {object} dto.PaginatedResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts [get]
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

	page, perPage, offset, hasPagination := ParsePaginationParams(c, 20)
	if hasPagination {
		paged := paginateAccounts(accounts, offset, perPage)
		total := int64(len(accounts))
		totalPages := int((total + int64(perPage) - 1) / int64(perPage))
		if totalPages == 0 {
			totalPages = 1
		}
		c.JSON(http.StatusOK, dto.PaginatedResponse{
			Data:       paged,
			Total:      total,
			Page:       page,
			PageSize:   perPage,
			TotalPages: totalPages,
		})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

// CreateAccount godoc
// @Summary Crear cuenta
// @Description Crea una nueva cuenta financiera para el usuario autenticado
// @Tags accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param account body dto.CreateAccountRequest true "Datos de la cuenta"
// @Success 201 {object} entity.Account
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts [post]
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

// GetAccount godoc
// @Summary Obtener cuenta
// @Description Obtiene una cuenta específica del usuario autenticado
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de cuenta"
// @Success 200 {object} entity.Account
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts/{id} [get]
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

// UpdateAccount godoc
// @Summary Actualizar cuenta
// @Description Actualiza una cuenta del usuario autenticado
// @Tags accounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID de cuenta"
// @Param account body dto.UpdateAccountRequest true "Datos a actualizar"
// @Success 200 {object} entity.Account
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts/{id} [put]
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

// DeleteAccount godoc
// @Summary Eliminar cuenta
// @Description Elimina una cuenta del usuario autenticado
// @Tags accounts
// @Security BearerAuth
// @Param id path int true "ID de cuenta"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts/{id} [delete]
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

// GetAccountSummaries godoc
// @Summary Resumen de cuentas
// @Description Obtiene un resumen compacto de cuentas del usuario autenticado
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Success 200 {array} entity.AccountSummary
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts/summaries [get]
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

// GetTotalBalance godoc
// @Summary Balance total
// @Description Obtiene el balance total consolidado de cuentas activas del usuario autenticado
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]number
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/accounts/total-balance [get]
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

	c.JSON(http.StatusOK, totalBalanceResponse{TotalBalance: total})
}
