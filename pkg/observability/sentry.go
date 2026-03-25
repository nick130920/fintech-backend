package observability

import (
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// InitSentry configura el cliente (Sentry.io, GlitchTip u otro compatible con el protocolo Sentry).
// Si [dsn] está vacío, no hace nada.
func InitSentry(dsn, ginMode string) error {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil
	}

	env := strings.TrimSpace(os.Getenv("SENTRY_ENVIRONMENT"))
	if env == "" {
		env = ginMode
	}

	release := strings.TrimSpace(os.Getenv("SENTRY_RELEASE"))

	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		EnableTracing:    false,
		TracesSampleRate: 0,
		BeforeSend:       scrubSentryEvent,
	})
}

// FlushSentry envía eventos pendientes (p. ej. al apagar el proceso).
func FlushSentry() {
	sentry.Flush(2 * time.Second)
}

// CaptureGlitchTipTestEvent envía un mensaje de prueba al DSN (GlitchTip/Sentry).
// Útil cuando la UI del proveedor no ofrece "Send test event".
func CaptureGlitchTipTestEvent() {
	sentry.CaptureMessage("GlitchTip/Sentry test event from fintech-backend (debug/sentry-test)")
	sentry.Flush(2 * time.Second)
}

// SentryGinMiddleware añade el handler de Sentry/GlitchTip al router Gin.
func SentryGinMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{Repanic: false})
}

func scrubSentryEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event == nil {
		return nil
	}
	if event.Request != nil {
		if event.Request.Headers != nil {
			delete(event.Request.Headers, "Authorization")
			delete(event.Request.Headers, "Cookie")
		}
		event.Request.Cookies = ""
	}
	return event
}

// CapturePanicForRequest envía un panic recuperado a GlitchTip/Sentry (si hay hub en el contexto).
func CapturePanicForRequest(c *gin.Context, recovered any) {
	if recovered == nil || c == nil {
		return
	}
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.RecoverWithContext(c.Request.Context(), recovered)
		return
	}
	sentry.CurrentHub().RecoverWithContext(c.Request.Context(), recovered)
}
