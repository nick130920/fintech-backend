package v1

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	log "github.com/sirupsen/logrus"
)

// ErrorResponse representa la estructura de respuesta de error estándar
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ErrorHandlerMiddleware maneja errores de forma centralizada
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Verificar si hay errores
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, err)
		}
	}
}

// RecoveryMiddleware maneja panics de forma más robusta
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Obtener stack trace
				stack := make([]byte, 4096)
				length := runtime.Stack(stack, false)

				// Log del panic con detalles completos
				requestID := getRequestID(c)
				log.WithFields(log.Fields{
					"request_id": requestID,
					"method":     c.Request.Method,
					"uri":        c.Request.RequestURI,
					"user_agent": c.Request.UserAgent(),
					"ip":         c.ClientIP(),
					"panic":      err,
					"stack":      string(stack[:length]),
				}).Error("🚨 PANIC RECOVERED")

				// Responder al cliente
				appErr := apperrors.ErrInternal.WithDetails("Se produjo un error inesperado")
				handleError(c, appErr)
			}
		}()
		c.Next()
	}
}

// handleError procesa y responde errores de forma consistente
func handleError(c *gin.Context, err error) {
	requestID := getRequestID(c)

	// Verificar si es un AppError
	if appErr, ok := apperrors.IsAppError(err); ok {
		// Log estructurado del error
		logFields := log.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"uri":        c.Request.RequestURI,
			"error_code": appErr.Code,
			"status":     appErr.StatusCode,
		}

		// Añadir error interno si existe
		if appErr.Internal != nil {
			logFields["internal_error"] = appErr.Internal.Error()
		}

		// Log apropiado según severidad
		if appErr.StatusCode >= 500 {
			log.WithFields(logFields).Error("❌ SERVER ERROR")
		} else if appErr.StatusCode >= 400 {
			log.WithFields(logFields).Warn("⚠️  CLIENT ERROR")
		}

		// Respuesta estructurada
		response := ErrorResponse{
			Error:     "error",
			Code:      string(appErr.Code),
			Message:   appErr.Message,
			Details:   appErr.Details,
			Fields:    appErr.Fields,
			Timestamp: time.Now(),
			RequestID: requestID,
		}

		c.JSON(appErr.StatusCode, response)
		return
	}

	// Error genérico no estructurado
	log.WithFields(log.Fields{
		"request_id": requestID,
		"method":     c.Request.Method,
		"uri":        c.Request.RequestURI,
		"error":      err.Error(),
	}).Error("❌ UNHANDLED ERROR")

	// Respuesta genérica para errores no estructurados
	response := ErrorResponse{
		Error:     "error",
		Code:      string(apperrors.ErrCodeInternal),
		Message:   "Error interno del servidor",
		Timestamp: time.Now(),
		RequestID: requestID,
	}

	c.JSON(http.StatusInternalServerError, response)
}

// AbortWithError facilita el envío de errores desde handlers
func AbortWithError(c *gin.Context, err error) {
	c.Error(err)
	c.Abort()
}

// AbortWithAppError facilita el envío de AppErrors específicos
func AbortWithAppError(c *gin.Context, appErr *apperrors.AppError) {
	c.Error(appErr)
	c.Abort()
}

// ValidationError convierte errores de validación en AppError
func ValidationError(details string, fields map[string]interface{}) *apperrors.AppError {
	return apperrors.ErrValidation.
		WithDetails(details).
		WithField("validation_errors", fields)
}

// getRequestID obtiene o genera un ID de request
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), c.ClientIP())
}
