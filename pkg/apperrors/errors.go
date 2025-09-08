package apperrors

import (
	"fmt"
	"net/http"
)

// ErrorCode representa códigos de error estructurados
type ErrorCode string

const (
	// Códigos de error genéricos
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"

	// Códigos específicos de dominio
	ErrCodeUserNotFound     ErrorCode = "USER_NOT_FOUND"
	ErrCodeInvalidAuth      ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	ErrCodeBudgetNotFound   ErrorCode = "BUDGET_NOT_FOUND"
	ErrCodeExpenseNotFound  ErrorCode = "EXPENSE_NOT_FOUND"
	ErrCodeCategoryNotFound ErrorCode = "CATEGORY_NOT_FOUND"
)

// AppError representa un error estructurado de la aplicación
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
	Internal   error                  `json:"-"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError crea un nuevo error de aplicación
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WithDetails añade detalles adicionales al error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithInternal añade el error interno para logging
func (e *AppError) WithInternal(err error) *AppError {
	e.Internal = err
	return e
}

// WithField añade un campo específico al error
func (e *AppError) WithField(key string, value interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// Errores predefinidos más estructurados
var (
	ErrInternal         = NewAppError(ErrCodeInternal, "Error interno del servidor", http.StatusInternalServerError)
	ErrInvalidRequest   = NewAppError(ErrCodeInvalidRequest, "Datos de solicitud inválidos", http.StatusBadRequest)
	ErrValidation       = NewAppError(ErrCodeValidation, "Error de validación", http.StatusBadRequest)
	ErrUnauthorized     = NewAppError(ErrCodeUnauthorized, "No autorizado", http.StatusUnauthorized)
	ErrForbidden        = NewAppError(ErrCodeForbidden, "Acceso denegado", http.StatusForbidden)
	ErrNotFound         = NewAppError(ErrCodeNotFound, "Recurso no encontrado", http.StatusNotFound)
	ErrConflict         = NewAppError(ErrCodeConflict, "Conflicto: el recurso ya existe", http.StatusConflict)
	ErrRateLimit        = NewAppError(ErrCodeRateLimit, "Límite de solicitudes excedido", http.StatusTooManyRequests)
	ErrTimeout          = NewAppError(ErrCodeTimeout, "Tiempo de espera agotado", http.StatusRequestTimeout)
	ErrPermissionDenied = NewAppError(ErrCodeForbidden, "Permisos denegados", http.StatusForbidden)
)

// Errores específicos de dominio más estructurados
var (
	// Auth/User Errors
	ErrUserNotFound       = NewAppError(ErrCodeUserNotFound, "Usuario no encontrado", http.StatusNotFound)
	ErrAccountInactive    = NewAppError(ErrCodeForbidden, "La cuenta del usuario no está activa", http.StatusForbidden)
	ErrInvalidCredentials = NewAppError(ErrCodeInvalidAuth, "Credenciales inválidas", http.StatusUnauthorized)
	ErrTokenExpired       = NewAppError(ErrCodeTokenExpired, "Token expirado", http.StatusUnauthorized)
	ErrEmailExists        = NewAppError(ErrCodeConflict, "El email ya existe", http.StatusConflict)

	// Budget Errors
	ErrBudgetExists             = NewAppError(ErrCodeConflict, "Ya existe un presupuesto para este mes", http.StatusConflict)
	ErrBudgetAllocationsExceed  = NewAppError(ErrCodeValidation, "Las asignaciones totales exceden el monto del presupuesto", http.StatusBadRequest)
	ErrBudgetNotFound           = NewAppError(ErrCodeBudgetNotFound, "Presupuesto no encontrado", http.StatusNotFound)
	ErrBudgetAllocationNotFound = NewAppError(ErrCodeNotFound, "Asignación de presupuesto no encontrada", http.StatusNotFound)
	ErrInvalidTotalAmount       = NewAppError(ErrCodeValidation, "El nuevo monto total no puede ser menor al ya gastado", http.StatusBadRequest)

	// Category Errors
	ErrCategoryNotFound = NewAppError(ErrCodeCategoryNotFound, "Categoría no encontrada", http.StatusNotFound)
	ErrInvalidCategory  = NewAppError(ErrCodeValidation, "Categoría inválida", http.StatusBadRequest)

	// Expense Errors
	ErrExpenseNotFound = NewAppError(ErrCodeExpenseNotFound, "Gasto no encontrado", http.StatusNotFound)
	ErrInvalidDate     = NewAppError(ErrCodeValidation, "Formato de fecha inválido", http.StatusBadRequest)
)

// IsAppError verifica si un error es del tipo AppError
func IsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// WrapError convierte un error genérico en AppError
func WrapError(err error, code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Internal:   err,
	}
}
