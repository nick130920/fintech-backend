package v1

import (
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// UploadHandler handles authenticated upload authorizations.
type UploadHandler struct {
	uploadUC *usecase.UploadUseCase
}

// NewUploadHandler constructs an upload HTTP handler.
func NewUploadHandler(uploadUC *usecase.UploadUseCase) *UploadHandler {
	return &UploadHandler{uploadUC: uploadUC}
}

// PresignUpload godoc
// @Summary Create a presigned upload URL
// @Description Creates a user-scoped, time-limited authorization for an object upload.
// @Tags uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param upload body dto.PresignUploadRequest true "Upload properties"
// @Success 200 {object} dto.PresignUploadResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 503 {object} dto.ErrorResponse
// @Router /api/v1/uploads/presign [post]
func (h *UploadHandler) PresignUpload(c *gin.Context) {
	if !isJSONRequest(c.GetHeader("Content-Type")) {
		handleErrorResponse(c, apperrors.NewAppError(apperrors.ErrCodeInvalidRequest, "Invalid upload request", http.StatusBadRequest))
		return
	}
	if h.uploadUC == nil {
		handleErrorResponse(c, apperrors.NewAppError(apperrors.ErrCodeObjectStorageUnavailable, "Object storage is unavailable", http.StatusServiceUnavailable))
		return
	}

	userID, err := RequireUserID(c)
	if err != nil {
		AbortWithAppError(c, apperrors.NewAppError(apperrors.ErrCodeUnauthorized, "Unauthorized", http.StatusUnauthorized))
		return
	}

	var request dto.PresignUploadRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		AbortWithAppError(c, apperrors.NewAppError(apperrors.ErrCodeInvalidRequest, "Invalid upload request", http.StatusBadRequest))
		return
	}

	result, err := h.uploadUC.PresignUpload(c.Request.Context(), userID, usecase.UploadInput{
		ContentType:   request.ContentType,
		ContentLength: request.ContentLength,
	})
	if err != nil {
		handleErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PresignUploadResponse{
		ObjectRef:        result.ObjectRef,
		UploadURL:        result.UploadURL,
		Headers:          result.Headers,
		ExpiresInSeconds: result.ExpiresInSeconds,
	})
}

func isJSONRequest(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	return err == nil && strings.EqualFold(mediaType, "application/json")
}
