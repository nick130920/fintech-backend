package dto

import "github.com/nick130920/proyecto-fintech/internal/entity"

// CreateUserRequest representa la estructura para registro de usuario
type CreateUserRequest struct {
	FirstName   string `json:"first_name" validate:"required,min=2,max=50"`
	LastName    string `json:"last_name" validate:"required,min=2,max=50"`
	Email       string `json:"email" validate:"required,email"`
	Phone       string `json:"phone" validate:"omitempty,min=10,max=20"`
	Password    string `json:"password" validate:"required,min=8,max=50"`
	DateOfBirth string `json:"date_of_birth" validate:"omitempty"` // Format: YYYY-MM-DD
	Locale      string `json:"locale" validate:"omitempty,len=5"`  // Format: es-MX
	Timezone    string `json:"timezone" validate:"omitempty"`      // Format: America/Mexico_City
}

// LoginRequest representa la estructura para login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

// UpdateUserRequest representa la estructura para actualización de usuario
type UpdateUserRequest struct {
	FirstName   string `json:"first_name" validate:"omitempty,min=2,max=50"`
	LastName    string `json:"last_name" validate:"omitempty,min=2,max=50"`
	Phone       string `json:"phone" validate:"omitempty,min=10,max=20"`
	DateOfBirth string `json:"date_of_birth" validate:"omitempty"` // Format: YYYY-MM-DD
	Locale      string `json:"locale" validate:"omitempty,len=5"`  // Format: es-MX
	Timezone    string `json:"timezone" validate:"omitempty"`      // Format: America/Mexico_City
}

// ChangePasswordRequest representa la estructura para cambio de contraseña
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=1"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=50"`
}

// RefreshTokenRequest representa la estructura para refresh token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LoginResponse representa la respuesta de login exitoso
type LoginResponse struct {
	User         *entity.UserPublic `json:"user"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	TokenType    string             `json:"token_type"` // "Bearer"
	ExpiresIn    int                `json:"expires_in"` // segundos
}

// TokenResponse representa la respuesta de refresh token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"` // "Bearer"
	ExpiresIn    int    `json:"expires_in"` // segundos
}

// UserPublicResponse representa información pública del usuario
type UserPublicResponse struct {
	ID          uint   `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"date_of_birth,omitempty"` // Format: YYYY-MM-DD
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
	IsActive    bool   `json:"is_active"`
	IsVerified  bool   `json:"is_verified"`
	CreatedAt   string `json:"created_at"`              // ISO format
	LastLoginAt string `json:"last_login_at,omitempty"` // ISO format
}

// UserStatsResponse representa estadísticas del usuario
type UserStatsResponse struct {
	TotalBudgets        int     `json:"total_budgets"`
	CurrentMonthBudget  float64 `json:"current_month_budget"`
	TotalSpentThisMonth float64 `json:"total_spent_this_month"`
	TotalCategories     int     `json:"total_categories"`
	TotalExpenses       int     `json:"total_expenses"`
	AccountCreatedDays  int     `json:"account_created_days"`
}

// UserProfileResponse representa el perfil completo del usuario
type UserProfileResponse struct {
	User  *UserPublicResponse `json:"user"`
	Stats *UserStatsResponse  `json:"stats"`
}

// LogoutResponse representa la respuesta de logout exitoso
type LogoutResponse struct {
	Message string `json:"message"`
}

// RegisterResponse representa la respuesta de registro exitoso
type RegisterResponse struct {
	Message string              `json:"message"`
	User    *UserPublicResponse `json:"user"`
}

// MapUserToPublicResponse convierte una entidad User a UserPublicResponse
func MapUserToPublicResponse(user *entity.User) *UserPublicResponse {
	response := &UserPublicResponse{
		ID:         user.ID,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Email:      user.Email,
		Phone:      user.Phone,
		Locale:     user.Locale,
		Timezone:   user.Timezone,
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Fecha de nacimiento
	if user.DateOfBirth != nil {
		response.DateOfBirth = user.DateOfBirth.Format("2006-01-02")
	}

	// Último login
	if user.LastLoginAt != nil {
		response.LastLoginAt = user.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return response
}

// ValidationErrorResponse representa errores de validación
type ValidationErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}
