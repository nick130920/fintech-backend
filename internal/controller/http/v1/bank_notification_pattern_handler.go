package v1

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/rs/zerolog"
)

// bankNotificationPatternHandler handles HTTP requests related to bank notification patterns.
type BankNotificationPatternHandler struct {
	uc     *usecase.BankNotificationPatternUseCase
	logger zerolog.Logger
}

// NewBankNotificationPatternHandler creates a new bankNotificationPatternHandler.
func NewBankNotificationPatternHandler(uc *usecase.BankNotificationPatternUseCase, logger zerolog.Logger) *BankNotificationPatternHandler {
	return &BankNotificationPatternHandler{uc: uc, logger: logger}
}

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

	result, err := h.uc.ProcessSMSBatchForSuggestions(c.Request.Context(), userID.(uint), req.Messages)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to analyze SMS batch")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze SMS batch"})
		return
	}

	c.JSON(http.StatusOK, result)
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
	_, exists := c.Get("user_id")
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
		".pdf": true, ".png": true, ".jpg": true, ".jpeg": true,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format (use PDF, PNG or JPEG)"})
		return
	}

	// Stub: do not process file yet; return empty suggestions. TODO: OCR + extraction.
	c.JSON(http.StatusOK, &dto.AnalyzeSMSBatchResponse{
		Suggestions: dto.BudgetSuggestions{
			TotalExpense3m: 0,
			ByCategory:     []dto.BudgetSuggestionCategory{},
		},
	})
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
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
