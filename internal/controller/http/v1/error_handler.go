package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/pkg/apperrors"
)

func handleErrorResponse(c *gin.Context, err error) {
	var code string
	var message string
	var status int

	switch {
	// 400 Bad Request
	case errors.Is(err, apperrors.ErrInvalidRequest),
		errors.Is(err, apperrors.ErrValidation),
		errors.Is(err, apperrors.ErrBudgetAllocationsExceed),
		errors.Is(err, apperrors.ErrInvalidTotalAmount),
		errors.Is(err, apperrors.ErrInvalidCategory),
		errors.Is(err, apperrors.ErrInvalidDate):
		status = http.StatusBadRequest
		code = "BAD_REQUEST"
		message = err.Error()

	// 401 Unauthorized
	case errors.Is(err, apperrors.ErrUnauthorized),
		errors.Is(err, apperrors.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		code = "UNAUTHORIZED"
		message = "No autorizado"

	// 403 Forbidden
	case errors.Is(err, apperrors.ErrPermissionDenied),
		errors.Is(err, apperrors.ErrAccountInactive):
		status = http.StatusForbidden
		code = "FORBIDDEN"
		message = "Acceso denegado"

	// 404 Not Found
	case errors.Is(err, apperrors.ErrNotFound),
		errors.Is(err, apperrors.ErrUserNotFound),
		errors.Is(err, apperrors.ErrBudgetNotFound),
		errors.Is(err, apperrors.ErrCategoryNotFound),
		errors.Is(err, apperrors.ErrExpenseNotFound):
		status = http.StatusNotFound
		code = "NOT_FOUND"
		message = "Recurso no encontrado"

	// 409 Conflict
	case errors.Is(err, apperrors.ErrDuplicateRecord),
		errors.Is(err, apperrors.ErrBudgetExists),
		errors.Is(err, apperrors.ErrEmailExists):
		status = http.StatusConflict
		code = "CONFLICT"
		message = err.Error()

	// 500 Internal Server Error
	default:
		status = http.StatusInternalServerError
		code = "INTERNAL_ERROR"
		message = "Error interno del servidor"
	}

	c.JSON(status, dto.ErrorResponse{
		Code:    code,
		Message: message,
		Details: err.Error(),
	})
}
