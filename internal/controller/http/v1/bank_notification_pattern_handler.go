package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"go.uber.org/zap"
)

// bankNotificationPatternHandler handles HTTP requests related to bank notification patterns.
type BankNotificationPatternHandler struct {
	uc     *usecase.BankNotificationPatternUseCase
	logger *zap.Logger
}

// NewBankNotificationPatternHandler creates a new bankNotificationPatternHandler.
func NewBankNotificationPatternHandler(uc *usecase.BankNotificationPatternUseCase, logger *zap.Logger) *BankNotificationPatternHandler {
	return &BankNotificationPatternHandler{uc: uc, logger: logger}
}

// @Summary      Generate a notification pattern from a message using AI
// @Description  Takes a notification message and uses an AI model to suggest regex patterns.
// @Tags         notification-patterns
// @Accept       json
// @Produce      json
// @Param        request body dto.GeneratePatternRequest true "Message and bank account ID"
// @Success      200 {object} dto.GeneratePatternResponse
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /notification-patterns/generate-from-message [post]
func (h *BankNotificationPatternHandler) GeneratePatternFromMessage(c *gin.Context) {
	var req dto.GeneratePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("bad request on generate pattern", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.uc.GeneratePatternFromMessage(c.Request.Context(), req.Message)
	if err != nil {
		h.logger.Error("failed to generate pattern with AI", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate pattern"})
		return
	}

	response := dto.GeneratePatternResponse{
		AmountRegex:      result.AmountRegex,
		DateRegex:        result.DateRegex,
		DescriptionRegex: result.DescriptionRegex,
		MerchantRegex:    result.MerchantRegex,
		KeywordsTrigger:  result.KeywordsTrigger,
	}

	c.JSON(http.StatusOK, response)
}

// --- Stubs for other methods to avoid compile errors ---

func (h *BankNotificationPatternHandler) GetUserPatterns(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) CreatePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) GetPatternStatistics(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) ProcessNotification(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) GetPattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) UpdatePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) DeletePattern(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) SetPatternStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
func (h *BankNotificationPatternHandler) GetBankAccountPatterns(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
