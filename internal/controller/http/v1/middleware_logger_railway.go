package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// RailwayLoggerMiddleware proporciona logging optimizado para Railway
func RailwayLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Generar request ID corto para Railway
		requestID := generateShortID()
		c.Set("request_id", requestID)

		// Capturar método y path
		method := c.Request.Method
		path := c.Request.URL.Path
		
		// Solo capturar body para métodos que modifican datos y en debug
		var bodyPreview string
		if gin.Mode() == gin.DebugMode && shouldLogBody(method) {
			bodyPreview = captureBodyPreview(c)
		}

		// Extraer info de auth
		hasAuth := c.Request.Header.Get("Authorization") != ""
		userID := extractUserIDFromContext(c)

		// Log compacto de inicio (solo en debug)
		if gin.Mode() == gin.DebugMode {
			authStatus := "❌"
			if hasAuth {
				authStatus = "✅"
			}
			log.Infof("🚀 [%s] %s %s %s | IP: %s", 
				requestID, method, path, authStatus, c.ClientIP())
			
			if bodyPreview != "" {
				log.Infof("📦 [%s] Body: %s", requestID, bodyPreview)
			}
		}

		// Response writer wrapper
		wrapper := &responseWriterWrapper{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = wrapper

		// Procesar request
		c.Next()

		// Calcular métricas
		latency := time.Since(start)
		status := c.Writer.Status()
		responseSize := wrapper.size

		// Log compacto de finalización
		logRequestResult(requestID, method, path, status, latency, 
			c.ClientIP(), userID, hasAuth, responseSize, c.Errors)
	}
}

// logRequestResult registra el resultado del request de forma compacta
func logRequestResult(requestID, method, path string, status int, latency time.Duration, 
	ip, userID string, hasAuth bool, responseSize int, errors []error) {
	
	// Emoji y color según status
	emoji, level := getStatusEmoji(status)
	
	// Formatear latencia
	latencyStr := formatLatency(latency)
	
	// Info de usuario
	userInfo := "anon"
	if userID != "" {
		userInfo = fmt.Sprintf("user:%s", userID)
	} else if hasAuth {
		userInfo = "auth"
	}
	
	// Tamaño de respuesta
	sizeStr := formatSize(responseSize)
	
	// Mensaje base compacto
	message := fmt.Sprintf("%s [%s] %s %s → %d | %s | %s | %s | IP: %s",
		emoji, requestID, method, path, status, latencyStr, sizeStr, userInfo, ip)
	
	// Añadir errores si existen
	if len(errors) > 0 {
		message += fmt.Sprintf(" | Errors: %d", len(errors))
	}

	// Log según nivel
	switch level {
	case "error":
		log.Error(message)
		// Log detalles de errores en líneas separadas
		for _, err := range errors {
			log.Errorf("   ❌ [%s] Error: %s", requestID, err.Error())
		}
	case "warn":
		log.Warn(message)
	default:
		log.Info(message)
	}
	
	// Log especial para requests lentos
	if latency > 2*time.Second {
		log.Warnf("🐌 [%s] SLOW REQUEST: %s %s took %s", requestID, method, path, latency)
	}
	
	// Log especial para responses grandes
	if responseSize > 1024*1024 { // > 1MB
		log.Warnf("📦 [%s] LARGE RESPONSE: %s", requestID, formatSize(responseSize))
	}
}

// generateShortID genera un ID corto para Railway
func generateShortID() string {
	id := uuid.New().String()
	// Tomar solo los primeros 8 caracteres para logs más compactos
	return id[:8]
}

// captureBodyPreview captura una vista previa del body
func captureBodyPreview(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	
	// Restaurar el body
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
	if len(body) == 0 {
		return ""
	}
	
	// Intentar parsear como JSON para vista previa más limpia
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// Crear vista previa compacta
		preview := make(map[string]interface{})
		count := 0
		for key, value := range jsonData {
			if count >= 3 { // Solo mostrar primeros 3 campos
				preview["..."] = fmt.Sprintf("and %d more fields", len(jsonData)-3)
				break
			}
			
			// Sanitizar campos sensibles
			if isSensitiveField(key) {
				preview[key] = "***"
			} else {
				// Truncar valores largos
				if str, ok := value.(string); ok && len(str) > 20 {
					preview[key] = str[:20] + "..."
				} else {
					preview[key] = value
				}
			}
			count++
		}
		
		// Convertir a string compacto
		if previewBytes, err := json.Marshal(preview); err == nil {
			return string(previewBytes)
		}
	}
	
	// Fallback: mostrar primeros 50 caracteres
	bodyStr := string(body)
	if len(bodyStr) > 50 {
		return bodyStr[:50] + "..."
	}
	return bodyStr
}

// getStatusEmoji devuelve emoji y nivel según status code
func getStatusEmoji(status int) (string, string) {
	switch {
	case status >= 500:
		return "💥", "error"
	case status >= 400:
		return "⚠️", "warn"
	case status >= 300:
		return "🔄", "info"
	case status >= 200:
		return "✅", "info"
	default:
		return "❓", "info"
	}
}

// formatLatency formatea la latencia de forma compacta
func formatLatency(latency time.Duration) string {
	if latency < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(latency.Nanoseconds())/1000)
	}
	if latency < time.Second {
		return fmt.Sprintf("%.0fms", float64(latency.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", latency.Seconds())
}

// formatSize formatea el tamaño de forma compacta
func formatSize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
}

// isSensitiveField verifica si un campo es sensible
func isSensitiveField(field string) bool {
	sensitive := []string{"password", "token", "secret", "key", "authorization", "auth"}
	fieldLower := strings.ToLower(field)
	
	for _, s := range sensitive {
		if strings.Contains(fieldLower, s) {
			return true
		}
	}
	return false
}

// SimpleRecoveryMiddleware recovery simple para Railway
func SimpleRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := getRequestID(c)
				
				// Log compacto del panic
				log.Errorf("💥 [%s] PANIC: %s %s | %v", 
					requestID, c.Request.Method, c.Request.URL.Path, err)
				
				// Respuesta de error
				c.JSON(500, gin.H{
					"error":      "internal_error",
					"message":    "Error interno del servidor",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

