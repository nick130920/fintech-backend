package v1

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/rs/zerolog"
)

// EmailConnectionHandler rutas correo OAuth (Gmail).
type EmailConnectionHandler struct {
	uc     *usecase.EmailGmailUseCase
	logger zerolog.Logger
}

type emailStatusFallbackResponse struct {
	Connected bool `json:"connected"`
}

// NewEmailConnectionHandler constructor.
func NewEmailConnectionHandler(uc *usecase.EmailGmailUseCase, logger zerolog.Logger) *EmailConnectionHandler {
	return &EmailConnectionHandler{uc: uc, logger: logger}
}

// CSP del middleware global (default-src 'self') bloquea <style> inline y fuentes de Google Fonts.
// Esta política solo aplica a las respuestas HTML del callback OAuth de Gmail.
const gmailOAuthCallbackCSP = "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; img-src data:; style-src 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com"

func setGmailOAuthCallbackHTMLHeaders(c *gin.Context) {
	c.Header("Content-Security-Policy", gmailOAuthCallbackCSP)
}

// GmailAuthorize godoc
// @Summary Iniciar autorización Gmail
// @Description Genera la URL OAuth para conectar Gmail con el usuario autenticado
// @Tags email
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /email/gmail/authorize [get]
func (h *EmailConnectionHandler) GmailAuthorize(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "service_unavailable", Message: "gmail oauth no configurado"})
		return
	}
	out, err := h.uc.BuildGmailAuthorizeURL(userID.(uint))
	if err != nil {
		h.logger.Error().Err(err).Msg("gmail authorize url")
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GmailOAuthCallback godoc
// @Summary Callback OAuth de Gmail
// @Description Endpoint público al que Google redirige tras autorización; completa el vínculo y muestra HTML
// @Tags email
// @Produce html
// @Param code query string false "Código OAuth"
// @Param state query string false "Estado OAuth"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 503 {string} string
// @Router /email/gmail/callback [get]
func (h *EmailConnectionHandler) GmailOAuthCallback(c *gin.Context) {
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.String(http.StatusServiceUnavailable, "Gmail no configurado en el servidor")
		return
	}
	if c.Query("error") != "" {
		setGmailOAuthCallbackHTMLHeaders(c)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML(c.Query("error_description"))))
		return
	}
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		setGmailOAuthCallbackHTMLHeaders(c)
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML("Faltan code o state")))
		return
	}
	if err := h.uc.HandleGmailCallback(c.Request.Context(), state, code); err != nil {
		h.logger.Error().Err(err).Msg("gmail oauth callback")
		setGmailOAuthCallbackHTMLHeaders(c)
		c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(gmailCallbackErrorHTML("No se pudo completar la conexión")))
		return
	}
	setGmailOAuthCallbackHTMLHeaders(c)
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
	Headline:    "Listo: Gmail vinculado",
	DetailHTML:  "Las notificaciones de tu banco en el correo podrán usarse en <strong>Money Flow</strong>. Vuelve a la app para continuar.",
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
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&amp;display=swap" rel="stylesheet">
<style>
/* Alineado con AppColors / dashboard_money_flow_blue: primary #007bff, glass 5%, bordes primary 10% */
:root {
  --mf-primary: #007bff;
  --mf-primary-dim: rgba(0, 123, 255, 0.12);
  --mf-primary-glow: rgba(0, 123, 255, 0.35);
  --mf-primary-hover: #0066d9;
  --mf-bg-top: #f6f8fa;
  --mf-bg-bot: #f1f5f9;
  --mf-glass: rgba(255, 255, 255, 0.78);
  --mf-glass-border: rgba(0, 123, 255, 0.12);
  --mf-glass-inner: rgba(0, 123, 255, 0.06);
  --mf-text: #0f172a;
  --mf-muted: #64748b;
  --mf-success: #10b981;
  --mf-error: #ef4444;
  --mf-chip-bg: rgba(0, 123, 255, 0.08);
  --mf-chip-border: rgba(0, 123, 255, 0.18);
}
@media (prefers-color-scheme: dark) {
  :root {
    --mf-bg-top: #0a0f14;
    --mf-bg-bot: #0d1419;
    --mf-glass: rgba(30, 41, 59, 0.55);
    --mf-glass-border: rgba(0, 123, 255, 0.22);
    --mf-glass-inner: rgba(255, 255, 255, 0.04);
    --mf-text: #f8fafc;
    --mf-muted: #94a3b8;
    --mf-chip-bg: rgba(0, 123, 255, 0.12);
    --mf-chip-border: rgba(0, 123, 255, 0.28);
    --mf-primary-glow: rgba(0, 123, 255, 0.45);
  }
}
*, *::before, *::after { box-sizing: border-box; }
html, body { margin: 0; min-height: 100%; }
body {
  font-family: "Inter", system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
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
  position: relative;
  overflow-x: hidden;
  background: linear-gradient(165deg, var(--mf-bg-top) 0%, var(--mf-bg-bot) 55%, var(--mf-bg-top) 100%);
}
.ambient {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
}
.orb {
  position: absolute;
  border-radius: 50%;
  filter: blur(72px);
  opacity: 0.55;
}
.orb-1 {
  width: min(320px, 85vw);
  height: min(320px, 85vw);
  background: var(--mf-primary-glow);
  top: -12%;
  right: -18%;
}
.orb-2 {
  width: min(280px, 75vw);
  height: min(280px, 75vw);
  background: var(--mf-primary);
  bottom: -20%;
  left: -22%;
  opacity: 0.22;
}
.grid-noise {
  position: fixed;
  inset: 0;
  z-index: 0;
  opacity: 0.04;
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}
.shell {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 420px;
}
.card {
  position: relative;
  background: var(--mf-glass);
  -webkit-backdrop-filter: blur(20px) saturate(160%);
  backdrop-filter: blur(20px) saturate(160%);
  border: 1px solid var(--mf-glass-border);
  border-radius: 20px;
  padding: clamp(24px, 6vw, 32px);
  box-shadow:
    0 0 0 1px var(--mf-glass-inner) inset,
    0 24px 48px rgba(15, 23, 42, 0.1),
    0 8px 24px rgba(0, 123, 255, 0.06);
}
@media (prefers-color-scheme: dark) {
  .card {
    box-shadow:
      0 0 0 1px rgba(255, 255, 255, 0.06) inset,
      0 24px 56px rgba(0, 0, 0, 0.55),
      0 8px 32px rgba(0, 123, 255, 0.12);
  }
}
.card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 16px;
  right: 16px;
  height: 1px;
  border-radius: 1px;
  background: linear-gradient(90deg, transparent, rgba(0, 123, 255, 0.35), transparent);
  opacity: 0.7;
}
.brand-row {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 24px;
}
.brand-mark {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 123, 255, 0.12);
  border: 1px solid rgba(0, 123, 255, 0.22);
  color: var(--mf-primary);
}
.brand-mark svg { width: 26px; height: 26px; }
.brand-text .name {
  font-weight: 700;
  font-size: 1.125rem;
  letter-spacing: -0.02em;
  color: var(--mf-text);
  line-height: 1.2;
}
.brand-text .tag {
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--mf-primary);
  margin-top: 4px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--mf-primary);
  background: var(--mf-chip-bg);
  border: 1px solid var(--mf-chip-border);
  margin-bottom: 16px;
}
.chip svg { width: 14px; height: 14px; flex-shrink: 0; }
.chip.error {
  color: var(--mf-error);
  background: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.28);
}
.status-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}
.status-icon svg { width: 32px; height: 32px; }
.status-icon.success {
  background: color-mix(in srgb, var(--mf-success) 20%, transparent);
  color: var(--mf-success);
  border: 1px solid color-mix(in srgb, var(--mf-success) 35%, transparent);
}
.status-icon.error {
  background: color-mix(in srgb, var(--mf-error) 20%, transparent);
  color: var(--mf-error);
  border: 1px solid color-mix(in srgb, var(--mf-error) 35%, transparent);
}
h1 {
  font-size: clamp(1.375rem, 4.5vw, 1.5rem);
  font-weight: 700;
  margin: 0 0 12px;
  line-height: 1.2;
  letter-spacing: -0.02em;
}
.msg {
  margin: 0 0 24px;
  font-size: 0.9375rem;
  color: var(--mf-muted);
  line-height: 1.55;
}
.msg strong { color: var(--mf-text); font-weight: 600; }
.error-detail { margin-bottom: 24px; }
.cta { margin-top: 4px; }
a.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 50px;
  padding: 14px 20px;
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: #fff !important;
  background: linear-gradient(180deg, #1a8cff 0%, var(--mf-primary) 45%, #0066d9 100%);
  text-decoration: none;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.15) inset,
    0 8px 24px rgba(0, 123, 255, 0.35);
  transition: transform 0.1s ease, box-shadow 0.15s ease, filter 0.15s ease;
}
a.btn:hover {
  filter: brightness(1.05);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.2) inset,
    0 12px 32px rgba(0, 123, 255, 0.4);
}
a.btn:active { transform: scale(0.98); }
.hint {
  margin: 18px 0 0;
  padding-top: 18px;
  border-top: 1px solid var(--mf-primary-dim);
  font-size: 0.8125rem;
  color: var(--mf-muted);
  text-align: center;
  line-height: 1.45;
}
@supports not (background: color-mix(in srgb, red, blue)) {
  .status-icon.success {
    background: rgba(16, 185, 129, 0.18);
    border-color: rgba(16, 185, 129, 0.35);
  }
  .status-icon.error {
    background: rgba(239, 68, 68, 0.18);
    border-color: rgba(239, 68, 68, 0.35);
  }
}
</style>
</head>
<body>
<div class="ambient" aria-hidden="true"><div class="orb orb-1"></div><div class="orb orb-2"></div></div>
<div class="grid-noise" aria-hidden="true"></div>
<div class="shell">
  <article class="card" aria-live="polite">
    <header class="brand-row">
      <div class="brand-mark" aria-hidden="true">` + gmailOAuthBrandMarkSVG() + `</div>
      <div class="brand-text">
        <div class="name">Money Flow</div>
        <div class="tag">Avisos bancarios · solo lectura</div>
      </div>
    </header>
    <div class="chip ` + p.StatusClass + `">` + gmailOAuthChipInner(p.StatusClass) + `</div>
    <div class="status-icon ` + p.StatusClass + `" aria-hidden="true">` + gmailOAuthStatusIconSVG(p.StatusClass) + `</div>
    <h1>` + template.HTMLEscapeString(p.Headline) + `</h1>
    ` + detail + `
    <div class="cta">
      <a class="btn" href="` + template.HTMLEscapeString(p.CTAHref) + `">` + template.HTMLEscapeString(p.CTALabel) + `</a>
    </div>
    <p class="hint">Si el botón no abre la app, vuelve a Money Flow manualmente; en algunos navegadores hay que permitir abrir enlaces externos.</p>
  </article>
</div>
</body>
</html>`
}

func gmailOAuthBrandMarkSVG() string {
	return `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
<path d="M12 2L3 7v10l9 5 9-5V7l-9-5z" stroke="currentColor" stroke-width="1.75" stroke-linejoin="round"/>
<path d="M3 7l9 5 9-5M12 12v9" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
</svg>`
}

func gmailOAuthChipInner(statusClass string) string {
	if statusClass == "error" {
		return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"/></svg><span>Conexión de correo</span>`
	}
	return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75"/></svg><span>Gmail · Money Flow</span>`
}

func gmailOAuthStatusIconSVG(kind string) string {
	if kind == "error" {
		return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M18 6L6 18M6 6l12 12"/></svg>`
	}
	return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6L9 17l-5-5"/></svg>`
}

// GetEmailConnectionStatus godoc
// @Summary Estado de conexión de correo
// @Description Obtiene el estado de conexión Gmail para el usuario autenticado
// @Tags email
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /email/status [get]
func (h *EmailConnectionHandler) GetEmailConnectionStatus(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}
	if h.uc == nil {
		c.JSON(http.StatusOK, emailStatusFallbackResponse{Connected: false})
		return
	}
	st, err := h.uc.GetEmailStatus(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, st)
}

// GmailDisconnect godoc
// @Summary Desconectar Gmail
// @Description Revoca la conexión Gmail del usuario autenticado
// @Tags email
// @Security BearerAuth
// @Success 204 "No Content"
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /email/gmail/disconnect [delete]
func (h *EmailConnectionHandler) GmailDisconnect(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}
	if h.uc == nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}
	if err := h.uc.DisconnectGmail(c.Request.Context(), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GmailSync godoc
// @Summary Sincronizar Gmail
// @Description Ejecuta una sincronización manual del buzón Gmail conectado
// @Tags email
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /email/gmail/sync [post]
func (h *EmailConnectionHandler) GmailSync(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}
	if h.uc == nil || !h.uc.IsGmailConfigured() {
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{Error: "service_unavailable", Message: "gmail no disponible"})
		return
	}
	out, err := h.uc.SyncGmailForUser(c.Request.Context(), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
