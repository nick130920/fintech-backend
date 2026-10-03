package usecase

import (
	"context"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/storage"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

// OpaqueReferenceGenerator creates a UUID reference for an upload object.
type OpaqueReferenceGenerator func() (string, error)

// UploadInput contains only client-controlled upload properties.
type UploadInput struct {
	ContentType   string
	ContentLength int64
}

// UploadPresign contains the authorization needed to upload one object.
type UploadPresign struct {
	ObjectRef        string
	UploadURL        string
	Headers          map[string]string
	ExpiresInSeconds int
}

// UploadUseCase creates user-scoped upload authorizations.
type UploadUseCase struct {
	store     storage.ObjectStore
	config    configs.UploadConfig
	generate  OpaqueReferenceGenerator
	mimeTypes map[string]struct{}
}

// NewUploadUseCase constructs an upload use case. A nil store represents optional
// object storage that is not configured for this deployment.
func NewUploadUseCase(store storage.ObjectStore, config configs.UploadConfig, generator OpaqueReferenceGenerator) *UploadUseCase {
	if generator == nil {
		generator = newOpaqueReference
	}

	return &UploadUseCase{
		store:     store,
		config:    config,
		generate:  generator,
		mimeTypes: allowedMediaTypes(config.AllowedTypes),
	}
}

// PresignUpload validates an authenticated user's upload and creates a PUT authorization.
func (u *UploadUseCase) PresignUpload(ctx context.Context, userID uint, input UploadInput) (UploadPresign, error) {
	contentType, ok := normalizedMediaType(input.ContentType)
	if userID == 0 || !ok || input.ContentLength <= 0 || input.ContentLength > u.config.MaxSize || !u.allowed(contentType) {
		return UploadPresign{}, invalidUploadRequestError()
	}
	if u.store == nil {
		return UploadPresign{}, objectStorageUnavailableError()
	}

	objectRef, err := u.generate()
	if err != nil || !validOpaqueReference(objectRef) {
		return UploadPresign{}, objectStorageUnavailableError()
	}

	presigned, err := u.store.PresignUpload(ctx, storage.PresignUploadRequest{
		Key:           "users/" + strconv.FormatUint(uint64(userID), 10) + "/" + objectRef,
		ContentType:   contentType,
		ContentLength: input.ContentLength,
		ExpiresIn:     time.Duration(u.config.SignedURLTTLSeconds) * time.Second,
	})
	if err != nil {
		return UploadPresign{}, objectStorageUnavailableError()
	}

	return UploadPresign{
		ObjectRef:        objectRef,
		UploadURL:        presigned.URL,
		Headers:          presigned.Headers,
		ExpiresInSeconds: u.config.SignedURLTTLSeconds,
	}, nil
}

func newOpaqueReference() (string, error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func allowedMediaTypes(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if mediaType, ok := normalizedMediaType(value); ok {
			result[mediaType] = struct{}{}
		}
	}
	return result
}

func normalizedMediaType(value string) (string, bool) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil || mediaType == "" {
		return "", false
	}
	return strings.ToLower(mediaType), true
}

func (u *UploadUseCase) allowed(contentType string) bool {
	_, ok := u.mimeTypes[contentType]
	return ok
}

func validOpaqueReference(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func invalidUploadRequestError() *apperrors.AppError {
	return apperrors.NewAppError(apperrors.ErrCodeInvalidRequest, "Invalid upload request", 400)
}

func objectStorageUnavailableError() *apperrors.AppError {
	return apperrors.NewAppError(apperrors.ErrCodeObjectStorageUnavailable, "Object storage is unavailable", 503)
}
