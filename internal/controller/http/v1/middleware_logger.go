package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorLoggerMiddleware middleware para logging detallado de errores
func ErrorLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log detallado para errores 4xx y 5xx
		if param.StatusCode >= 400 {
			log.Printf(
				"❌ ERROR [%d] %s %s | IP: %s | Latencia: %v | User-Agent: %s | Error: %s",
				param.StatusCode,
				param.Method,
				param.Path,
				param.ClientIP,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		}

		// Log normal para requests exitosos
		return ""
	})
}

// RequestLoggerMiddleware middleware para logging de requests detallados
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Capturar el body del request para debugging
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		// Log del request entrante
		authHeader := c.Request.Header.Get("Authorization")
		authPreview := "N/A"
		if len(authHeader) > 20 {
			authPreview = authHeader[:20] + "..."
		} else if len(authHeader) > 0 {
			authPreview = authHeader
		}

		log.Printf(
			"[DEBUG] 📤 REQUEST: %s %s | IP: %s | Auth: %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			authPreview,
		)

		// Log del body si es POST/PUT/PATCH
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if len(body) > 0 {
				var prettyJSON bytes.Buffer
				if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
					log.Printf("[DEBUG] 📦 REQUEST BODY:\n%s", prettyJSON.String())
				} else {
					log.Printf("📦 REQUEST BODY (raw): %s", string(body))
				}
			}
		}

		// Procesar request
		c.Next()

		// Log de la respuesta
		latency := time.Since(start)
		status := c.Writer.Status()

		if status >= 500 {
			log.Printf(
				"[ERROR] 📥 RESPONSE ERROR [%d] | Latencia: %v | Errors: %v",
				status,
				latency,
				c.Errors,
			)
		} else if status >= 400 {
			log.Printf(
				"[WARN] 📥 RESPONSE ERROR [%d] | Latencia: %v | Errors: %v",
				status,
				latency,
				c.Errors,
			)
		} else {
			log.Printf(
				"[INFO] 📥 RESPONSE SUCCESS [%d] | Latencia: %v",
				status,
				latency,
			)
		}
	}
}

// RecoveryMiddleware middleware personalizado para recovery con logging
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf(
			"[ERROR] 🚨 PANIC RECOVERED: %v | Method: %s | Path: %s | IP: %s",
			recovered,
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
		)

		c.JSON(500, gin.H{
			"code":    "INTERNAL_PANIC",
			"message": "Error interno del servidor",
			"details": "Se ha producido un error inesperado",
		})
	})
}
