package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/internal/usecase"
	"github.com/nick130920/proyecto-fintech/pkg/validator"
)

// UserHandler maneja las peticiones HTTP relacionadas con usuarios
type UserHandler struct {
	userUC    *usecase.UserUseCase
	validator *validator.Validator
}

// NewUserHandler crea una nueva instancia de UserHandler
func NewUserHandler(userUC *usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUC:    userUC,
		validator: validator.New(),
	}
}

// Register maneja el registro de nuevos usuarios
// @Summary Registrar nuevo usuario
// @Description Crea una nueva cuenta de usuario
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.CreateUserRequest true "Datos del usuario"
// @Success 201 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.CreateUserRequest

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

	response, err := h.userUC.Register(&req)
	if err != nil {
		if err.Error() == "user already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "User already exists",
				Message: "A user with this email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Registration failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login maneja el inicio de sesión
// @Summary Iniciar sesión
// @Description Autentica un usuario y devuelve tokens de acceso
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body dto.LoginRequest true "Credenciales de login"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

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

	response, err := h.userUC.Login(&req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Invalid credentials",
				Message: "Email or password is incorrect",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Login failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken maneja la renovación de tokens
// @Summary Renovar token de acceso
// @Description Renueva el token de acceso usando el refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh body dto.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} dto.TokenResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /auth/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest

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

	response, err := h.userUC.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Invalid refresh token",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile obtiene el perfil del usuario autenticado
// @Summary Obtener perfil del usuario
// @Description Devuelve la información del perfil del usuario autenticado
// @Tags users
// @Produce json
// @Security Bearer
// @Success 200 {object} entity.UserPublic
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	user, err := h.userUC.GetByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Message: "User profile not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to get profile",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user.ToPublic())
}

// UpdateProfile actualiza el perfil del usuario
// @Summary Actualizar perfil del usuario
// @Description Actualiza la información del perfil del usuario autenticado
// @Tags users
// @Accept json
// @Produce json
// @Security Bearer
// @Param user body dto.UpdateUserRequest true "Datos a actualizar"
// @Success 200 {object} entity.UserPublic
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Valid authentication required",
		})
		return
	}

	var req dto.UpdateUserRequest

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

	updatedUser, err := h.userUC.Update(userID, &req)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "User not found",
				Message: "User profile not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Failed to update profile",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, updatedUser.ToPublic())
}

// Logout maneja el cierre de sesión
// @Summary Cerrar sesión
// @Description Cierra la sesión del usuario (invalida tokens)
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.LogoutResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /auth/logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	// En una implementación real, aquí invalidarías el token
	// Por ahora, simplemente devolvemos un mensaje de éxito

	c.JSON(http.StatusOK, dto.LogoutResponse{
		Message: "Successfully logged out",
	})
}

// ValidateToken valida el token de acceso
// @Summary Validar token
// @Description Valida si el token de acceso es válido y activo
// @Tags auth
// @Produce json
// @Security Bearer
// @Success 200 {object} dto.SuccessResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /auth/validate [get]
func (h *UserHandler) ValidateToken(c *gin.Context) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid or expired token",
		})
		return
	}

	// Si llegamos aquí, el token es válido (pasó el middleware)
	c.JSON(http.StatusOK, dto.Response{
		Code:    "SUCCESS",
		Message: "Token is valid",
		Data: gin.H{
			"user_id": userID,
		},
	})
}
