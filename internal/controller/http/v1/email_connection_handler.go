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
	return gmailOAuthCallbackHTML(gmailOAuthPageSuccess)
}

func gmailCallbackErrorHTML(msg string) string {
	if msg == "" {
		msg = "Error desconocido"
	}
	esc := template.HTMLEscapeString(msg)
	return gmailOAuthCallbackHTML(gmailOAuthPageError(esc))
}

type gmailOAuthPage struct {
	Title       string
	ThemeColor  string
	StatusClass string
	Headline    string
	DetailHTML  string
	CTAHref     string
	CTALabel    string
}

var gmailOAuthPageSuccess = gmailOAuthPage{
	Title:       "Gmail conectado · Money Flow",
	ThemeColor:  "#007bff",
	StatusClass: "success",
	Headline:    "Gmail conectado",
	DetailHTML:  "Tu cuenta quedó vinculada. Vuelve a <strong>Money Flow</strong> para seguir gestionando tus finanzas.",
	CTAHref:     "moneyflow://email-connected",
	CTALabel:    "Abrir Money Flow",
}

func gmailOAuthPageError(escapedDetail string) gmailOAuthPage {
	return gmailOAuthPage{
		Title:       "Error al conectar · Money Flow",
		ThemeColor:  "#ef4444",
		StatusClass: "error",
		Headline:    "No pudimos completar la conexión",
		DetailHTML:  `<p class="msg error-detail">` + escapedDetail + `</p>`,
		CTAHref:     "moneyflow://email-error",
		CTALabel:    "Volver a Money Flow",
	}
}

func gmailOAuthCallbackHTML(p gmailOAuthPage) string {
	detail := p.DetailHTML
	if p.StatusClass == "success" {
		detail = `<p class="msg">` + p.DetailHTML + `</p>`
	}
	return `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="theme-color" content="` + template.HTMLEscapeString(p.ThemeColor) + `">
<meta name="color-scheme" content="light dark">
<title>` + template.HTMLEscapeString(p.Title) + `</title>
<style>
:root {
  --mf-primary: #007bff;
  --mf-primary-hover: #0066d9;
  --mf-bg: #f6f8fa;
  --mf-surface: #ffffff;
  --mf-text: #0f172a;
  --mf-muted: #475569;
  --mf-border: rgba(0, 123, 255, 0.14);
  --mf-success: #10b981;
  --mf-error: #ef4444;
  --mf-shadow: rgba(15, 23, 42, 0.08);
}
@media (prefers-color-scheme: dark) {
  :root {
    --mf-bg: #0a0f14;
    --mf-surface: rgba(30, 41, 59, 0.72);
    --mf-text: #f1f5f9;
    --mf-muted: #94a3b8;
    --mf-border: rgba(0, 123, 255, 0.22);
    --mf-shadow: rgba(0, 0, 0, 0.45);
  }
}
*, *::before, *::after { box-sizing: border-box; }
html, body { margin: 0; min-height: 100%; }
body {
  font-family: system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: var(--mf-bg);
  color: var(--mf-text);
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  padding:
    max(24px, env(safe-area-inset-top, 0px))
    max(24px, env(safe-area-inset-right, 0px))
    max(24px, env(safe-area-inset-bottom, 0px))
    max(24px, env(safe-area-inset-left, 0px));
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
}
.shell { width: 100%; max-width: 440px; }
.card {
  background: var(--mf-surface);
  backdrop-filter: blur(12px);
  border: 1px solid var(--mf-border);
  border-radius: 16px;
  padding: clamp(20px, 5vw, 28px) clamp(20px, 5vw, 28px);
  box-shadow: 0 12px 40px var(--mf-shadow);
}
.brand {
  font-weight: 700;
  font-size: 0.8125rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--mf-primary);
  margin-bottom: 1.25rem;
}
.status-icon {
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1.25rem;
}
.status-icon svg { width: 28px; height: 28px; }
.status-icon.success {
  background: color-mix(in srgb, var(--mf-success) 18%, transparent);
  color: var(--mf-success);
}
.status-icon.error {
  background: color-mix(in srgb, var(--mf-error) 18%, transparent);
  color: var(--mf-error);
}
h1 {
  font-size: clamp(1.25rem, 4vw, 1.375rem);
  font-weight: 700;
  margin: 0 0 0.75rem;
  line-height: 1.25;
}
.msg {
  margin: 0 0 1.5rem;
  font-size: 0.9375rem;
  color: var(--mf-muted);
}
.msg strong { color: var(--mf-text); font-weight: 600; }
.error-detail { margin-bottom: 1.5rem; }
.cta {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
a.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 48px;
  padding: 0.875rem 1.25rem;
  font-size: 1rem;
  font-weight: 600;
  color: #fff !important;
  background: var(--mf-primary);
  text-decoration: none;
  border-radius: 12px;
  transition: background 0.15s ease, transform 0.08s ease;
}
a.btn:hover { background: var(--mf-primary-hover); }
a.btn:active { transform: scale(0.98); }
.hint {
  margin: 1rem 0 0;
  font-size: 0.8125rem;
  color: var(--mf-muted);
  text-align: center;
  line-height: 1.45;
}
@supports not (background: color-mix(in srgb, red, blue)) {
  .status-icon.success { background: rgba(16, 185, 129, 0.15); }
  .status-icon.error { background: rgba(239, 68, 68, 0.15); }
}
</style>
</head>
<body>
<div class="shell">
  <article class="card" aria-live="polite">
    <div class="brand">Money Flow</div>
    <div class="status-icon ` + p.StatusClass + `" aria-hidden="true">` + gmailOAuthStatusIconSVG(p.StatusClass) + `</div>
    <h1>` + template.HTMLEscapeString(p.Headline) + `</h1>
    ` + detail + `
    <div class="cta">
      <a class="btn" href="` + template.HTMLEscapeString(p.CTAHref) + `">` + template.HTMLEscapeString(p.CTALabel) + `</a>
    </div>
    <p class="hint">Si el enlace no abre la app, cierra esta pestaña y vuelve desde Money Flow.</p>
  </article>
</div>
</body>
</html>`
}

func gmailOAuthStatusIconSVG(kind string) string {
	if kind == "error" {
		return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6L6 18M6 6l12 12"/></svg>`
	}
	return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6L9 17l-5-5"/></svg>`
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
