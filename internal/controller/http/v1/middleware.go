package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/proyecto-fintech/internal/controller/http/v1/dto"
	"github.com/nick130920/proyecto-fintech/pkg/auth"
)

// AuthMiddleware maneja la autenticación JWT
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware crea una nueva instancia de AuthMiddleware
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// RequireAuth middleware que requiere autenticación válida
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener token del header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Missing authorization header",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extraer token del header
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Invalid authorization header",
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		// Validar token
		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Invalid token",
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		// Guardar información del usuario en el contexto
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_first_name", claims.FirstName)
		c.Set("user_last_name", claims.LastName)

		c.Next()
	}
}

// OptionalAuth middleware que permite pero no requiere autenticación
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		// Guardar información del usuario si el token es válido
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_first_name", claims.FirstName)
		c.Set("user_last_name", claims.LastName)

		c.Next()
	}
}

// GetUserIDFromContext extrae el user ID del contexto de Gin
func GetUserIDFromContext(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(uint)
	return id, ok
}

// GetUserEmailFromContext extrae el email del usuario del contexto de Gin
func GetUserEmailFromContext(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}

	emailStr, ok := email.(string)
	return emailStr, ok
}

// RequireUserID helper que obtiene el user ID o devuelve error
func RequireUserID(c *gin.Context) (uint, error) {
	userID, exists := GetUserIDFromContext(c)
	if !exists {
		return 0, errors.New("user ID not found in context")
	}
	return userID, nil
}

// RateLimitMiddleware middleware simple de rate limiting (placeholder)
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implementar rate limiting real
		c.Next()
	}
}

// LoggingMiddleware middleware personalizado de logging
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}
