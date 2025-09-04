package apperrors

import "errors"

// Generic Errors
var (
	ErrInternal         = errors.New("error interno del servidor")
	ErrInvalidRequest   = errors.New("datos de solicitud inválidos")
	ErrValidation       = errors.New("validación fallida")
	ErrUnauthorized     = errors.New("no autorizado")
	ErrNotFound         = errors.New("recurso no encontrado")
	ErrDuplicateRecord  = errors.New("el registro ya existe")
	ErrPermissionDenied = errors.New("permisos denegados")
)

// Auth/User Errors
var (
	ErrUserNotFound       = errors.New("usuario no encontrado")
	ErrAccountInactive    = errors.New("la cuenta del usuario no está activa")
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrEmailExists        = errors.New("el email ya existe")
)

// Budget Errors
var (
	ErrBudgetExists             = errors.New("ya existe un presupuesto para este mes")
	ErrBudgetAllocationsExceed  = errors.New("las asignaciones totales exceden el monto del presupuesto")
	ErrBudgetNotFound           = errors.New("presupuesto no encontrado")
	ErrBudgetAllocationNotFound = errors.New("asignación de presupuesto no encontrada")
	ErrInvalidTotalAmount       = errors.New("el nuevo monto total no puede ser menor al ya gastado")
)

// Category Errors
var (
	ErrCategoryNotFound = errors.New("categoría no encontrada")
	ErrInvalidCategory  = errors.New("categoría inválida")
)

// Expense Errors
var (
	ErrExpenseNotFound = errors.New("gasto no encontrado")
	ErrInvalidDate     = errors.New("formato de fecha inválido")
)
