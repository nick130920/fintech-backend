package v1

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/rs/zerolog"
)

// EmailConnectionHandler rutas correo OAuth (Gmail).
type EmailConnectionHandler struct {
	uc     *usecase.EmailGmailUseCase
	logger zerolog.Logger
}

// NewEmailConnectionHandler constructor.
func NewEmailConnectionHandler(uc *usecase.EmailGmailUseCase, logger zerolog.Logger) *EmailConnectionHandler {
	return &EmailConnectionHandler{uc: uc, logger: logger}
}

// GmailAuthorize GET devuelve URL para abrir en navegador.
func (h *EmailConnectionHandler) GmailAuthorize(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gmail oauth no configurado"})
		return
	}
	out, err := h.uc.BuildGmailAuthorizeURL(userID.(uint))
	if err != nil {
		h.logger.Error().Err(err).Msg("gmail authorize url")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GmailOAuthCallback GET público: Google redirige aquí con ?code=&state=.
func (h *EmailConnectionHandler) GmailOAuthCallback(c *gin.Context) {
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.String(http.StatusServiceUnavailable, "Gmail no configurado en el servidor")
		return
	}
	if c.Query("error") != "" {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML(c.Query("error_description"))))
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML("Faltan code o state")))
		return
	}
	if err := h.uc.HandleGmailCallback(c.Request.Context(), state, code); err != nil {
		h.logger.Error().Err(err).Msg("gmail oauth callback")
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML("No se pudo completar la conexión")))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(gmailCallbackSuccessHTML()))
}

func gmailCallbackSuccessHTML() string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Gmail conectado</title></head><body style="font-family:system-ui;padding:24px;">
<p>Gmail conectado correctamente. Puedes volver a la app Money Flow.</p>
<p><a href="moneyflow://email-connected">Abrir app</a></p>
</body></html>`
}

func gmailCallbackErrorHTML(msg string) string {
	if msg == "" {
		msg = "Error desconocido"
	}
	esc := template.HTMLEscapeString(msg)
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Error</title></head><body style="font-family:system-ui;padding:24px;">
<p>` + esc + `</p>
<p><a href="moneyflow://email-error">Volver a la app</a></p>
</body></html>`
}

// GetEmailConnectionStatus GET estado vinculación.
func (h *EmailConnectionHandler) GetEmailConnectionStatus(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.uc == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}
	st, err := h.uc.GetEmailStatus(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, st)
}

// GmailDisconnect DELETE revoca conexión.
func (h *EmailConnectionHandler) GmailDisconnect(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.uc == nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}
	if err := h.uc.DisconnectGmail(c.Request.Context(), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GmailSync POST sincroniza buzón ahora.
func (h *EmailConnectionHandler) GmailSync(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gmail no disponible"})
		return
	}
	out, err := h.uc.SyncGmailForUser(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
