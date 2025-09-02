package apperrors

import "errors"

// Generic Errors
var (
	ErrInternal         = errors.New("internal server error")
	ErrInvalidRequest   = errors.New("invalid request data")
	ErrValidation       = errors.New("validation failed")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotFound         = errors.New("resource not found")
	ErrDuplicateRecord  = errors.New("record already exists")
	ErrPermissionDenied = errors.New("permission denied")
)

// Auth/User Errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrAccountInactive    = errors.New("user account is not active")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists")
)

// Budget Errors
var (
	ErrBudgetExists            = errors.New("budget already exists for this month")
	ErrBudgetAllocationsExceed = errors.New("total allocations exceed budget amount")
	ErrBudgetNotFound          = errors.New("budget not found")
	ErrInvalidTotalAmount      = errors.New("new total amount cannot be less than amount already spent")
)

// Category Errors
var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrInvalidCategory  = errors.New("invalid category")
)

// Expense Errors
var (
	ErrExpenseNotFound = errors.New("expense not found")
	ErrInvalidDate     = errors.New("invalid date format")
)
