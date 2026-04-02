package v1

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/rs/zerolog"
)

// analyzeSMSBatchMaxDuration: 4 chunks + reintentos 429 pueden acercarse a 6 min; margen para no cortar a mitad.
const analyzeSMSBatchMaxDuration = 10 * time.Minute

// bankNotificationPatternHandler handles HTTP requests related to bank notification patterns.
type BankNotificationPatternHandler struct {
	uc     *usecase.BankNotificationPatternUseCase
	logger zerolog.Logger
}

// NewBankNotificationPatternHandler creates a new bankNotificationPatternHandler.
func NewBankNotificationPatternHandler(uc *usecase.BankNotificationPatternUseCase, logger zerolog.Logger) *BankNotificationPatternHandler {
	return &BankNotificationPatternHandler{uc: uc, logger: logger}
}

// GetUserPatterns godoc
// @Summary Listar patrones del usuario
// @Description Obtiene los patrones de notificaciones bancarias del usuario autenticado
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Success 200 {array} entity.BankNotificationPattern
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns [get]
func (h *BankNotificationPatternHandler) GetUserPatterns(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	patterns, err := h.uc.GetUserPatterns(userID.(uint))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user patterns")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user patterns"})
		return
	}

	c.JSON(http.StatusOK, patterns)
}

// CreatePattern godoc
// @Summary Crear patrón de notificación
// @Description Crea un nuevo patrón para procesar notificaciones bancarias
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateBankNotificationPatternRequest true "Datos del patrón"
// @Success 201 {object} entity.BankNotificationPattern
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns [post]
func (h *BankNotificationPatternHandler) CreatePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.CreateBankNotificationPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on create pattern")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	h.logger.Info().Str("pattern_name", req.Name).Msg("Received request to create a new pattern")

	pattern, err := h.uc.CreatePattern(userID.(uint), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create pattern"})
		return
	}

	c.JSON(http.StatusCreated, pattern)
}

// GetPatternStatistics godoc
// @Summary Estadísticas de patrones
// @Description Obtiene estadísticas agregadas de uso y efectividad de patrones
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/stats [get]
func (h *BankNotificationPatternHandler) GetPatternStatistics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	stats, err := h.uc.GetPatternStatistics(userID.(uint))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get pattern statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pattern statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ProcessNotification godoc
// @Summary Procesar notificación bancaria
// @Description Procesa una notificación bancaria y devuelve la transacción sugerida/procesada
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ProcessNotificationRequest true "Payload de notificación"
// @Success 200 {object} dto.ProcessedNotificationResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/process [post]
func (h *BankNotificationPatternHandler) ProcessNotification(c *gin.Context) {
	var req dto.ProcessNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on process notification")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Si el UserID no viene en el request, lo tomamos del contexto (usuario autenticado)
	if req.UserID == 0 {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		req.UserID = userID.(uint)
	}

	result, err := h.uc.ProcessNotificationWebhook(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to process notification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process notification"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ProcessSMSWithAI processes an SMS directly using AI (OpenRouter/Mistral)
// This is the new simplified flow that doesn't require patterns
// @Summary      Process SMS with AI
// @Description  Analyzes an SMS notification using AI to extract transaction data
// @Tags         notification-patterns
// @Accept       json
// @Produce      json
// @Param        request body dto.ProcessSMSWithAIRequest true "SMS message to process"
// @Success      200 {object} dto.ProcessedNotificationResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/process-sms [post]
func (h *BankNotificationPatternHandler) ProcessSMSWithAI(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.ProcessSMSWithAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on process SMS with AI")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.logger.Info().Str("message_preview", truncateString(req.Message, 50)).Msg("Processing SMS with AI")

	result, err := h.uc.ProcessSMSWithAI(c.Request.Context(), userID.(uint), req.Message)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to process SMS with AI")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process SMS with AI"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// AnalyzeSMSBatch analyzes multiple SMS for budget suggestions (no transactions created).
// @Summary      Analyze SMS batch for budget suggestions
// @Description  Processes multiple SMS to extract expense data and returns aggregated suggestions by category. Does not create any transactions.
// @Tags         notification-patterns
// @Accept       json
// @Produce      json
// @Param        request body dto.AnalyzeSMSBatchRequest true "Batch of SMS messages"
// @Success      200 {object} dto.AnalyzeSMSBatchResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/analyze-sms-batch [post]
func (h *BankNotificationPatternHandler) AnalyzeSMSBatch(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.AnalyzeSMSBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on analyze SMS batch")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusOK, &dto.AnalyzeSMSBatchResponse{
			Suggestions: dto.BudgetSuggestions{ByCategory: []dto.BudgetSuggestionCategory{}},
		})
		return
	}

	// Contexto propio (no el del HTTP): evita cancelación por proxy/cliente en llamadas a OpenRouter.
	// 12 min: muchos SMS ⇒ muchos chunks; el modelo gratuito + 429/reintentos puede superar 5 min.
	ctx, cancel := context.WithTimeout(context.Background(), analyzeSMSBatchMaxDuration)
	defer cancel()
	result, err := h.uc.ProcessSMSBatchForSuggestions(ctx, userID.(uint), req.Messages)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to analyze SMS batch")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze SMS batch"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ProcessSMSBatchWithAI procesa muchos SMS en pocos requests a la IA y crea transacciones (alta confianza).
// @Summary      Procesar lote de SMS con IA
// @Description  Agrupa SMS, filtra ruido, extrae movimientos por chunks y crea gastos/ingresos automáticos.
// @Tags         notification-patterns
// @Accept       json
// @Produce      json
// @Param        request body dto.ProcessSMSBatchWithAIRequest true "Lista de SMS (body + date opcional)"
// @Success      200 {object} dto.ProcessSMSBatchWithAIResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/process-sms-batch [post]
func (h *BankNotificationPatternHandler) ProcessSMSBatchWithAI(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.ProcessSMSBatchWithAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on process SMS batch with AI")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusOK, &dto.ProcessSMSBatchWithAIResponse{})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), analyzeSMSBatchMaxDuration)
	defer cancel()

	result, err := h.uc.ProcessSMSBatchWithAI(ctx, userID.(uint), req.Messages)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to process SMS batch with AI")
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// StartAnalyzeSMSBatchJob crea un job asíncrono; la app hace polling a GET .../jobs/:jobId (evita HTTP largo / proxy reset).
// @Summary      Iniciar job de análisis de SMS
// @Description  Crea un job asíncrono para analizar SMS y generar sugerencias de presupuesto
// @Tags         notification-patterns
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.AnalyzeSMSBatchRequest true "Lote de SMS"
// @Success      202 {object} dto.AnalyzeSMSBatchJobResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/analyze-sms-batch/jobs [post]
func (h *BankNotificationPatternHandler) StartAnalyzeSMSBatchJob(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req dto.AnalyzeSMSBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on start analyze SMS batch job")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	out, err := h.uc.StartSMSBatchSuggestionJob(userID.(uint), req.Messages)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to start SMS batch suggestion job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if out.JobID == "" {
		c.JSON(http.StatusOK, out)
		return
	}
	c.JSON(http.StatusAccepted, out)
}

// GetAnalyzeSMSBatchJobStatus devuelve pending|processing|completed|failed para polling.
// @Summary      Consultar estado de job de análisis
// @Description  Devuelve el estado actual del job asíncrono de análisis de SMS
// @Tags         notification-patterns
// @Produce      json
// @Security     BearerAuth
// @Param        jobId path string true "ID del job"
// @Success      200 {object} dto.AnalyzeSMSBatchJobStatusResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/analyze-sms-batch/jobs/{jobId} [get]
func (h *BankNotificationPatternHandler) GetAnalyzeSMSBatchJobStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing job id"})
		return
	}

	resp, err := h.uc.GetSMSBatchSuggestionJobStatus(userID.(uint), jobID)
	if err != nil {
		handleErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AnalyzeStatement analyzes an uploaded bank statement (PDF or image) for budget suggestions (stub: returns empty for now).
// @Summary      Analyze bank statement for budget suggestions
// @Description  Accepts a PDF or image file and returns aggregated budget suggestions. File is not stored. Stub implementation.
// @Tags         notification-patterns
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "PDF or image (JPEG/PNG)"
// @Success      200 {object} dto.AnalyzeSMSBatchResponse
// @Failure      400 {object} gin.H
// @Failure      401 {object} gin.H
// @Failure      500 {object} gin.H
// @Router       /notification-patterns/analyze-statement [post]
func (h *BankNotificationPatternHandler) AnalyzeStatement(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.logger.Error().Err(err).Msg("analyze-statement: missing file")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	const maxSize = 10 << 20 // 10 MB
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 10 MB)"})
		return
	}

	allowed := map[string]bool{
		".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".txt": true, ".csv": true,
	}
	lower := strings.ToLower(file.Filename)
	ok := false
	for ext := range allowed {
		if strings.HasSuffix(lower, ext) {
			ok = true
			break
		}
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format (use PDF, PNG, JPEG, TXT or CSV)"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}

	// Paso intermedio: soportamos análisis real para texto/csv.
	if strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".csv") {
		resp, err := h.uc.AnalyzeStatementText(c.Request.Context(), userID.(uint), string(content))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze statement"})
			return
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": "image/pdf analysis not enabled yet; please upload TXT/CSV as intermediate format",
	})
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetPattern godoc
// @Summary Obtener patrón
// @Description Obtiene un patrón específico por ID
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Success 200 {object} entity.BankNotificationPattern
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /notification-patterns/{id} [get]
func (h *BankNotificationPatternHandler) GetPattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	pattern, err := h.uc.GetPattern(userID.(uint), uint(patternID))
	if err != nil {
		h.logger.Error().Err(err).Uint64("pattern_id", patternID).Msg("failed to get pattern")
		c.JSON(http.StatusNotFound, gin.H{"error": "pattern not found"})
		return
	}

	c.JSON(http.StatusOK, pattern)
}
// UpdatePattern godoc
// @Summary Actualizar patrón
// @Description Actualiza un patrón existente de notificación bancaria
// @Tags notification-patterns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Param request body dto.UpdateBankNotificationPatternRequest true "Datos a actualizar"
// @Success 200 {object} entity.BankNotificationPattern
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/{id} [put]
func (h *BankNotificationPatternHandler) UpdatePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	var req dto.UpdateBankNotificationPatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on update pattern")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pattern, err := h.uc.UpdatePattern(userID.(uint), uint(patternID), &req)
	if err != nil {
		h.logger.Error().Err(err).Uint64("pattern_id", patternID).Msg("failed to update pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pattern"})
		return
	}

	c.JSON(http.StatusOK, pattern)
}
// DeletePattern godoc
// @Summary Eliminar patrón
// @Description Elimina un patrón de notificación bancaria del usuario
// @Tags notification-patterns
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Success 204 "No Content"
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/{id} [delete]
func (h *BankNotificationPatternHandler) DeletePattern(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	if err := h.uc.DeletePattern(userID.(uint), uint(patternID)); err != nil {
		h.logger.Error().Err(err).Uint64("pattern_id", patternID).Msg("failed to delete pattern")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete pattern"})
		return
	}

	c.Status(http.StatusNoContent)
}
// SetPatternStatus godoc
// @Summary Cambiar estado del patrón
// @Description Activa o desactiva un patrón de notificación bancaria
// @Tags notification-patterns
// @Accept json
// @Security BearerAuth
// @Param id path int true "ID del patrón"
// @Param request body dto.SetPatternStatusRequest true "Nuevo estado"
// @Success 204 "No Content"
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/{id}/status [patch]
func (h *BankNotificationPatternHandler) SetPatternStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	patternID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pattern ID"})
		return
	}

	var req dto.SetPatternStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error().Err(err).Msg("bad request on set pattern status")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.uc.SetPatternStatus(userID.(uint), uint(patternID), req.Status); err != nil {
		h.logger.Error().Err(err).Uint64("pattern_id", patternID).Msg("failed to set pattern status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set pattern status"})
		return
	}

	c.Status(http.StatusNoContent)
}
// GetBankAccountPatterns godoc
// @Summary Patrones por cuenta bancaria
// @Description Obtiene patrones asociados a una cuenta bancaria del usuario
// @Tags notification-patterns
// @Produce json
// @Security BearerAuth
// @Param bank_account_id path int true "ID de cuenta bancaria"
// @Param active_only query bool false "Filtrar solo activos"
// @Success 200 {array} entity.BankNotificationPattern
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /notification-patterns/bank-account/{bank_account_id} [get]
func (h *BankNotificationPatternHandler) GetBankAccountPatterns(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	bankAccountID, err := strconv.ParseUint(c.Param("bank_account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bank account ID"})
		return
	}

	activeOnly, _ := strconv.ParseBool(c.Query("active_only"))

	patterns, err := h.uc.GetBankAccountPatterns(userID.(uint), uint(bankAccountID), activeOnly)
	if err != nil {
		h.logger.Error().Err(err).Uint64("bank_account_id", bankAccountID).Msg("failed to get bank account patterns")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get bank account patterns"})
		return
	}

	c.JSON(http.StatusOK, patterns)
}
