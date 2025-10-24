package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// WebhookHandler maneja webhooks de notificaciones bancarias
type WebhookHandler struct {
	bankNotificationUC *usecase.BankNotificationPatternUseCase
	transactionUC      *usecase.TransactionUseCase
}

// NewWebhookHandler crea una nueva instancia del handler
func NewWebhookHandler(
	bankNotificationUC *usecase.BankNotificationPatternUseCase,
	transactionUC *usecase.TransactionUseCase,
) *WebhookHandler {
	return &WebhookHandler{
		bankNotificationUC: bankNotificationUC,
		transactionUC:      transactionUC,
	}
}

// ReceiveBankNotification recibe notificaciones de bancos vía webhook
// @Summary Recibir notificación bancaria
// @Description Procesa notificaciones SMS/Push de bancos y crea transacciones automáticamente
// @Tags webhooks
// @Accept json
// @Produce json
// @Param notification body dto.BankNotificationWebhook true "Notificación bancaria"
// @Success 200 {object} dto.ProcessedNotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /webhooks/bank-notification [post]
func (h *WebhookHandler) ReceiveBankNotification(c *gin.Context) {
	requestID := getRequestID(c)

	var req dto.BankNotificationWebhook
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithFields(log.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to bind bank notification webhook")

		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Formato de notificación inválido",
		})
		return
	}

	log.WithFields(log.Fields{
		"request_id": requestID,
		"phone":      req.Phone,
		"channel":    req.Channel,
		"message":    req.Message[:min(100, len(req.Message))], // Solo primeros 100 chars
	}).Info("Received bank notification webhook")

	// Procesar la notificación
	processReq := dto.ProcessNotificationRequest{
		Message:    req.Message,
		Channel:    req.Channel,
		Phone:      req.Phone,
		ReceivedAt: req.ReceivedAt,
		BankCode:   req.BankCode,
		UserID:     req.UserID, // Puede venir del webhook o ser determinado por el teléfono
	}

	result, err := h.bankNotificationUC.ProcessNotificationWebhook(processReq)
	if err != nil {
		log.WithFields(log.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		}).Error("Failed to process bank notification")

		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.StatusCode, dto.ErrorResponse{
				Error:   string(appErr.Code),
				Message: appErr.Message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "processing_failed",
			Message: "Error al procesar la notificación",
		})
		return
	}

	// Si se creó una transacción automáticamente, registrarla
	if result.TransactionCreated && result.TransactionID > 0 {
		log.WithFields(log.Fields{
			"request_id":     requestID,
			"transaction_id": result.TransactionID,
			"amount":         result.Amount,
			"confidence":     result.Confidence,
		}).Info("Transaction created automatically from bank notification")
	}

	c.JSON(http.StatusOK, result)
}

// ReceiveSMSNotification recibe notificaciones SMS específicamente
// @Summary Recibir notificación SMS
// @Description Procesa notificaciones SMS de bancos
// @Tags webhooks
// @Accept json
// @Produce json
// @Param sms body dto.SMSNotificationWebhook true "Notificación SMS"
// @Success 200 {object} dto.ProcessedNotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /webhooks/sms [post]
func (h *WebhookHandler) ReceiveSMSNotification(c *gin.Context) {
	_ = getRequestID(c)

	var req dto.SMSNotificationWebhook
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Formato de SMS inválido",
		})
		return
	}

	// Convertir a formato estándar
	webhookReq := dto.BankNotificationWebhook{
		Message:    req.Message,
		Phone:      req.From,
		Channel:    "sms",
		ReceivedAt: req.ReceivedAt,
		UserID:     h.findUserByPhone(req.To), // Buscar usuario por teléfono destino
	}

	// Reutilizar la lógica principal
	h.processBankNotificationInternal(c, webhookReq)
}

// ProcessPendingNotifications procesa notificaciones pendientes de validación
// @Summary Procesar notificaciones pendientes
// @Description Procesa notificaciones que requieren validación manual
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body dto.ProcessPendingNotificationsRequest true "Request"
// @Success 200 {object} dto.ProcessPendingNotificationsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /webhooks/process-pending [post]
func (h *WebhookHandler) ProcessPendingNotifications(c *gin.Context) {
	requestID := getRequestID(c)

	var req dto.ProcessPendingNotificationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: "Request inválido",
		})
		return
	}

	log.WithFields(log.Fields{
		"request_id": requestID,
		"user_id":    req.UserID,
		"limit":      req.Limit,
	}).Info("Processing pending notifications")

	// Obtener notificaciones pendientes
	pendingNotifications, err := h.bankNotificationUC.GetPendingNotifications(req.UserID, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "processing_failed",
			Message: "Error al obtener notificaciones pendientes",
		})
		return
	}

	processed := 0
	failed := 0

	for _, notification := range pendingNotifications {
		processReq := dto.ProcessNotificationRequest{
			Message:    notification.RawMessage,
			Channel:    notification.Channel,
			Phone:      notification.Phone,
			ReceivedAt: notification.ReceivedAt,
			UserID:     req.UserID,
		}

		_, err := h.bankNotificationUC.ProcessNotificationWebhook(processReq)
		if err != nil {
			failed++
			log.WithFields(log.Fields{
				"request_id":      requestID,
				"notification_id": notification.ID,
				"error":           err.Error(),
			}).Error("Failed to process pending notification")
		} else {
			processed++
		}
	}

	response := dto.ProcessPendingNotificationsResponse{
		TotalFound: len(pendingNotifications),
		Processed:  processed,
		Failed:     failed,
		RequestID:  requestID,
	}

	c.JSON(http.StatusOK, response)
}

// GetNotificationStats obtiene estadísticas de notificaciones procesadas
// @Summary Estadísticas de notificaciones
// @Description Obtiene estadísticas de procesamiento de notificaciones
// @Tags webhooks
// @Produce json
// @Param user_id query int false "ID del usuario"
// @Param days query int false "Días hacia atrás (default: 30)"
// @Success 200 {object} dto.NotificationStatsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /webhooks/stats [get]
func (h *WebhookHandler) GetNotificationStats(c *gin.Context) {
	_ = getRequestID(c)

	userIDStr := c.Query("user_id")
	daysStr := c.DefaultQuery("days", "30")

	var userID *uint
	if userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			uid := uint(id)
			userID = &uid
		}
	}

	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	stats, err := h.bankNotificationUC.GetNotificationStats(userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "stats_failed",
			Message: "Error al obtener estadísticas",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// processBankNotificationInternal lógica interna para procesar notificaciones
func (h *WebhookHandler) processBankNotificationInternal(c *gin.Context, req dto.BankNotificationWebhook) {
	_ = getRequestID(c)

	processReq := dto.ProcessNotificationRequest{
		Message:    req.Message,
		Channel:    req.Channel,
		Phone:      req.Phone,
		ReceivedAt: req.ReceivedAt,
		BankCode:   req.BankCode,
		UserID:     req.UserID,
	}

	result, err := h.bankNotificationUC.ProcessNotificationWebhook(processReq)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.StatusCode, dto.ErrorResponse{
				Error:   string(appErr.Code),
				Message: appErr.Message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "processing_failed",
			Message: "Error al procesar la notificación",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// findUserByPhone busca un usuario por número de teléfono
func (h *WebhookHandler) findUserByPhone(phone string) uint {
	// TODO: Implementar búsqueda de usuario por teléfono
	// Por ahora retornamos 0, pero debería buscar en la base de datos
	// o tener un mapeo de teléfonos a usuarios
	return 0
}

// min función helper para obtener el mínimo entre dos enteros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
