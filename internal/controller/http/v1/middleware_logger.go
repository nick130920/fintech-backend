package v1

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nick130920/fintech-backend/pkg/logger"
)

// LoggerMiddleware creates a gin middleware for structured logging with zerolog.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Get logger instance
		log := logger.Get()

		// Log request start
		log.Info().
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("Request started")

		// Create a response writer proxy to capture status and size
		writer := &responseWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = writer

		c.Next()

		// Log request completion
		latency := time.Since(start)
		status := c.Writer.Status()

		logEvent := log.Info() // Default to info level
		if status >= http.StatusInternalServerError {
			logEvent = log.Error()
		} else if status >= http.StatusBadRequest {
			logEvent = log.Warn()
		}

		// Add all relevant fields
		logEvent.
			Str("request_id", requestID).
			Int("status", status).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("latency", latency).
			Int("response_size", writer.body.Len()).
			Msg("Request completed")

		// Log slow requests
		if latency > 2*time.Second {
			log.Warn().
				Str("request_id", requestID).
				Dur("latency", latency).
				Msg("Slow request detected")
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
