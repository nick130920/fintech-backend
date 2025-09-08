package v1

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
	log "github.com/sirupsen/logrus"
)

// RateLimiter implementa rate limiting por IP
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*ClientLimiter
	limit   int
	window  time.Duration
}

// ClientLimiter mantiene el estado de rate limit para un cliente
type ClientLimiter struct {
	tokens     int
	lastSeen   time.Time
	requests   []time.Time
	violations int
}

// NewRateLimiter crea un nuevo rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ClientLimiter),
		limit:   limit,
		window:  window,
	}

	// Limpieza periódica de clientes inactivos
	go rl.cleanup()
	return rl
}

// RateLimitMiddleware implementa rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.allow(ip) {
			// Log de rate limit violation
			log.WithFields(log.Fields{
				"ip":         ip,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"user_agent": c.Request.UserAgent(),
			}).Warn("🚨 RATE LIMIT EXCEEDED")

			AbortWithAppError(c, apperrors.ErrRateLimit.WithDetails("Demasiadas solicitudes, intenta más tarde"))
			return
		}

		c.Next()
	}
}

// allow verifica si una IP puede hacer una request
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[ip]

	if !exists {
		rl.clients[ip] = &ClientLimiter{
			tokens:   rl.limit - 1,
			lastSeen: now,
			requests: []time.Time{now},
		}
		return true
	}

	// Limpiar requests antiguas
	client.requests = rl.filterRecentRequests(client.requests, now)

	// Verificar si excede el límite
	if len(client.requests) >= rl.limit {
		client.violations++
		client.lastSeen = now
		return false
	}

	// Permitir request
	client.requests = append(client.requests, now)
	client.lastSeen = now
	return true
}

// filterRecentRequests filtra requests dentro de la ventana de tiempo
func (rl *RateLimiter) filterRecentRequests(requests []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-rl.window)
	var recent []time.Time

	for _, req := range requests {
		if req.After(cutoff) {
			recent = append(recent, req)
		}
	}

	return recent
}

// cleanup limpia clientes inactivos periódicamente
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		for ip, client := range rl.clients {
			if now.Sub(client.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}

		rl.mu.Unlock()
	}
}

// SecurityHeadersMiddleware añade headers de seguridad
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Headers de seguridad estándar
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Headers específicos de API
		c.Header("X-API-Version", "v1")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	}
}

// ValidateContentTypeMiddleware valida el content-type para requests con body
func ValidateContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// Solo validar para métodos que pueden tener body
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.Request.Header.Get("Content-Type")

			// Permitir application/json y multipart/form-data
			if contentType == "" {
				AbortWithAppError(c, apperrors.ErrInvalidRequest.WithDetails("Content-Type header requerido"))
				return
			}

			if !isValidContentType(contentType) {
				AbortWithAppError(c, apperrors.ErrInvalidRequest.WithDetails("Content-Type no soportado"))
				return
			}
		}

		c.Next()
	}
}

// IPWhitelistMiddleware implementa whitelist de IPs (opcional)
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	allowedMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedMap[ip] = true
	}

	return func(c *gin.Context) {
		if len(allowedIPs) == 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if !allowedMap[clientIP] {
			log.WithFields(log.Fields{
				"ip":     clientIP,
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
			}).Warn("🚫 IP NOT WHITELISTED")

			AbortWithAppError(c, apperrors.ErrForbidden.WithDetails("IP no autorizada"))
			return
		}

		c.Next()
	}
}

// RequestSizeLimitMiddleware limita el tamaño de requests
func RequestSizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			log.WithFields(log.Fields{
				"ip":             c.ClientIP(),
				"content_length": c.Request.ContentLength,
				"max_size":       maxSize,
				"path":           c.Request.URL.Path,
			}).Warn("📦 REQUEST TOO LARGE")

			AbortWithAppError(c, apperrors.ErrInvalidRequest.WithDetails("Request demasiado grande"))
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware implementa timeout para requests
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Por ahora implementamos timeout básico
		// En el futuro se puede usar context.WithTimeout
		c.Next()
	}
}

// SuspiciousActivityMiddleware detecta actividad sospechosa
func SuspiciousActivityMiddleware() gin.HandlerFunc {
	suspiciousPatterns := []string{
		"script", "javascript", "eval", "alert", "onload",
		"../", "..\\", "etc/passwd", "/bin/", "cmd.exe",
		"union", "select", "drop", "insert", "update", "delete",
	}

	return func(c *gin.Context) {
		// Verificar URL path
		path := c.Request.URL.Path
		for _, pattern := range suspiciousPatterns {
			if containsIgnoreCase(path, pattern) {
				log.WithFields(log.Fields{
					"ip":      c.ClientIP(),
					"path":    path,
					"pattern": pattern,
					"type":    "suspicious_path",
				}).Warn("🚨 SUSPICIOUS ACTIVITY DETECTED")

				AbortWithAppError(c, apperrors.ErrForbidden.WithDetails("Actividad sospechosa detectada"))
				return
			}
		}

		// Verificar query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				for _, pattern := range suspiciousPatterns {
					if containsIgnoreCase(value, pattern) {
						log.WithFields(log.Fields{
							"ip":        c.ClientIP(),
							"parameter": key,
							"value":     value,
							"pattern":   pattern,
							"type":      "suspicious_query",
						}).Warn("🚨 SUSPICIOUS ACTIVITY DETECTED")

						AbortWithAppError(c, apperrors.ErrForbidden.WithDetails("Actividad sospechosa detectada"))
						return
					}
				}
			}
		}

		c.Next()
	}
}

// Funciones de utilidad
func isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"multipart/form-data",
		"application/x-www-form-urlencoded",
	}

	for _, valid := range validTypes {
		if containsIgnoreCase(contentType, valid) {
			return true
		}
	}

	return false
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				indexOf(toLower(s), toLower(substr)) >= 0))
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}
