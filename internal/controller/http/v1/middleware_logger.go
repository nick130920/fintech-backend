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

// RequestMetrics contiene métricas de request
type RequestMetrics struct {
	RequestID    string
	Method       string
	Path         string
	Status       int
	Latency      time.Duration
	RequestSize  int64
	ResponseSize int64
	UserAgent    string
	IP           string
	UserID       string
	HasAuth      bool
	Errors       []string
}

// EnhancedLoggerMiddleware proporciona logging avanzado con métricas
func EnhancedLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generar request ID único
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Capturar body del request (solo para debugging en desarrollo)
		var requestBody []byte
		var requestSize int64

		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			requestSize = int64(len(requestBody))
		}

		// Extraer información de autenticación
		authHeader := c.Request.Header.Get("Authorization")
		hasAuth := authHeader != ""
		userID := extractUserIDFromContext(c)

		// Log de request entrante
		logRequestStart(c, requestID, requestBody, hasAuth)

		// Response writer wrapper para capturar métricas
		wrapper := &responseWriterWrapper{
			ResponseWriter: c.Writer,
			size:           0,
		}
		c.Writer = wrapper

		// Procesar request
		c.Next()

		// Calcular métricas
		metrics := RequestMetrics{
			RequestID:    requestID,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Status:       c.Writer.Status(),
			Latency:      time.Since(start),
			RequestSize:  requestSize,
			ResponseSize: int64(wrapper.size),
			UserAgent:    c.Request.UserAgent(),
			IP:           c.ClientIP(),
			UserID:       userID,
			HasAuth:      hasAuth,
			Errors:       extractErrors(c),
		}

		// Log de respuesta con métricas completas
		logRequestComplete(metrics)
	}
}

// logRequestStart registra el inicio del request
func logRequestStart(c *gin.Context, requestID string, body []byte, hasAuth bool) {
	fields := log.Fields{
		"request_id": requestID,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"query":      c.Request.URL.RawQuery,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
		"has_auth":   hasAuth,
		"headers":    sanitizeHeaders(c.Request.Header),
	}

	// Solo loggear body en desarrollo y para métodos que modifican datos
	if gin.Mode() == gin.DebugMode && shouldLogBody(c.Request.Method) && len(body) > 0 {
		fields["request_body"] = sanitizeRequestBody(body)
	}

	log.WithFields(fields).Info("📤 REQUEST START")
}

// logRequestComplete registra la finalización del request con métricas
func logRequestComplete(metrics RequestMetrics) {
	fields := log.Fields{
		"request_id":    metrics.RequestID,
		"method":        metrics.Method,
		"path":          metrics.Path,
		"status":        metrics.Status,
		"latency_ms":    metrics.Latency.Milliseconds(),
		"latency_str":   metrics.Latency.String(),
		"request_size":  metrics.RequestSize,
		"response_size": metrics.ResponseSize,
		"ip":            metrics.IP,
		"user_agent":    metrics.UserAgent,
		"has_auth":      metrics.HasAuth,
	}

	if metrics.UserID != "" {
		fields["user_id"] = metrics.UserID
	}

	if len(metrics.Errors) > 0 {
		fields["errors"] = metrics.Errors
	}

	// Determinar nivel de log y emoji según status
	emoji, level := getLogLevelAndEmoji(metrics.Status)

	switch level {
	case "error":
		log.WithFields(fields).Error(fmt.Sprintf("%s REQUEST COMPLETE [%d]", emoji, metrics.Status))
	case "warn":
		log.WithFields(fields).Warn(fmt.Sprintf("%s REQUEST COMPLETE [%d]", emoji, metrics.Status))
	default:
		log.WithFields(fields).Info(fmt.Sprintf("%s REQUEST COMPLETE [%d]", emoji, metrics.Status))
	}

	// Log de métricas adicionales para requests lentos
	if metrics.Latency > 1*time.Second {
		log.WithFields(fields).Warn("🐌 SLOW REQUEST DETECTED")
	}

	// Log de requests grandes
	if metrics.RequestSize > 1024*1024 { // > 1MB
		log.WithFields(fields).Warn("📦 LARGE REQUEST DETECTED")
	}
}

// responseWriterWrapper envuelve el ResponseWriter para capturar métricas
type responseWriterWrapper struct {
	gin.ResponseWriter
	size int
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	size, err := w.ResponseWriter.Write(data)
	w.size += size
	return size, err
}

// Funciones de utilidad
func extractUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("%v", userID)
	}
	return ""
}

func extractErrors(c *gin.Context) []string {
	var errors []string
	for _, err := range c.Errors {
		errors = append(errors, err.Error())
	}
	return errors
}

func getLogLevelAndEmoji(status int) (string, string) {
	switch {
	case status >= 500:
		return "❌", "error"
	case status >= 400:
		return "⚠️", "warn"
	case status >= 300:
		return "🔄", "info"
	default:
		return "✅", "info"
	}
}

func shouldLogBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

func sanitizeHeaders(headers map[string][]string) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Ocultar headers sensibles
		if lowerKey == "authorization" || lowerKey == "cookie" || lowerKey == "x-api-key" {
			if len(values) > 0 && len(values[0]) > 10 {
				sanitized[key] = values[0][:10] + "..."
			} else {
				sanitized[key] = "[HIDDEN]"
			}
		} else {
			sanitized[key] = values
		}
	}

	return sanitized
}

func sanitizeRequestBody(body []byte) interface{} {
	// Limitar tamaño del body en logs
	const maxBodySize = 1024 // 1KB
	if len(body) > maxBodySize {
		return fmt.Sprintf("%s... [TRUNCATED %d bytes]", string(body[:maxBodySize]), len(body))
	}

	// Intentar parsear como JSON para mejor formato
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// Sanitizar campos sensibles en JSON
		if jsonMap, ok := jsonData.(map[string]interface{}); ok {
			sanitizeJSONFields(jsonMap)
		}
		return jsonData
	}

	return string(body)
}

func sanitizeJSONFields(data map[string]interface{}) {
	sensitiveFields := []string{"password", "token", "secret", "key", "authorization"}

	for key, value := range data {
		lowerKey := strings.ToLower(key)

		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				if str, ok := value.(string); ok && len(str) > 4 {
					data[key] = str[:4] + "***"
				} else {
					data[key] = "[HIDDEN]"
				}
				break
			}
		}
	}
}
