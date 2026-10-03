package usecase

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/storage"
	"github.com/nick130920/fintech-backend/pkg/apperrors"
)

func TestUploadUseCasePresignUploadUsesCanonicalContentTypeAndServerOwnedKey(t *testing.T) {
	store := &storage.FakeObjectStore{
		UploadResult: storage.PresignedUpload{
			URL:     "https://objects.example.test/upload",
			Headers: map[string]string{"Content-Type": "application/pdf"},
		},
	}
	useCase := NewUploadUseCase(store, configs.UploadConfig{
		MaxSize:             1024,
		AllowedTypes:        []string{" image/jpeg ", "application/pdf"},
		SignedURLTTLSeconds: 300,
	}, func() (string, error) {
		return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil
	})

	result, err := useCase.PresignUpload(context.Background(), 42, UploadInput{
		ContentType:   "Application/PDF; charset=binary",
		ContentLength: 512,
	})
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}
	if result.ObjectRef != "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d" {
		t.Errorf("ObjectRef = %q", result.ObjectRef)
	}
	if result.UploadURL != "https://objects.example.test/upload" {
		t.Errorf("UploadURL = %q", result.UploadURL)
	}
	if result.ExpiresInSeconds != 300 {
		t.Errorf("ExpiresInSeconds = %d, want 300", result.ExpiresInSeconds)
	}
	if len(store.UploadRequests) != 1 {
		t.Fatalf("PresignUpload calls = %d, want 1", len(store.UploadRequests))
	}
	request := store.UploadRequests[0]
	if request.Key != "users/42/018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d" {
		t.Errorf("Key = %q, want server-owned user-scoped key", request.Key)
	}
	if request.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want canonical media type", request.ContentType)
	}
	if request.ContentLength != 512 {
		t.Errorf("ContentLength = %d, want 512", request.ContentLength)
	}
	if request.ExpiresIn != 300*time.Second {
		t.Errorf("ExpiresIn = %s, want %s", request.ExpiresIn, 300*time.Second)
	}
}

func TestUploadUseCasePresignUploadRejectsInvalidInputWithoutCallingStorage(t *testing.T) {
	tests := []struct {
		name   string
		userID uint
		input  UploadInput
	}{
		{name: "zero user ID", userID: 0, input: UploadInput{ContentType: "image/jpeg", ContentLength: 1}},
		{name: "blank content type", userID: 1, input: UploadInput{ContentType: " ", ContentLength: 1}},
		{name: "unsupported content type", userID: 1, input: UploadInput{ContentType: "text/plain", ContentLength: 1}},
		{name: "nonpositive length", userID: 1, input: UploadInput{ContentType: "image/jpeg", ContentLength: 0}},
		{name: "length above maximum", userID: 1, input: UploadInput{ContentType: "image/jpeg", ContentLength: 101}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &storage.FakeObjectStore{}
			useCase := NewUploadUseCase(store, configs.UploadConfig{
				MaxSize:             100,
				AllowedTypes:        []string{"image/jpeg"},
				SignedURLTTLSeconds: 60,
			}, func() (string, error) { return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil })

			_, err := useCase.PresignUpload(context.Background(), tt.userID, tt.input)
			appErr, ok := apperrors.IsAppError(err)
			if !ok || appErr.StatusCode != http.StatusBadRequest || appErr.Code != apperrors.ErrCodeInvalidRequest {
				t.Fatalf("PresignUpload() error = %#v, want stable 400 invalid request AppError", err)
			}
			if len(store.UploadRequests) != 0 {
				t.Fatalf("PresignUpload calls = %d, want 0", len(store.UploadRequests))
			}
		})
	}
}

func TestUploadUseCasePresignUploadMapsUnavailableStorageAndFailures(t *testing.T) {
	tests := []struct {
		name      string
		store     storage.ObjectStore
		generator OpaqueReferenceGenerator
	}{
		{name: "absent storage", store: nil, generator: func() (string, error) { return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil }},
		{name: "presign failure", store: &storage.FakeObjectStore{UploadErr: errors.New("provider secret failure")}, generator: func() (string, error) { return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil }},
		{name: "invalid generated reference", store: &storage.FakeObjectStore{}, generator: func() (string, error) { return "not-a-uuid", nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase := NewUploadUseCase(tt.store, configs.UploadConfig{
				MaxSize:             100,
				AllowedTypes:        []string{"image/jpeg"},
				SignedURLTTLSeconds: 60,
			}, tt.generator)

			_, err := useCase.PresignUpload(context.Background(), 1, UploadInput{ContentType: "image/jpeg", ContentLength: 1})
			appErr, ok := apperrors.IsAppError(err)
			if !ok || appErr.StatusCode != http.StatusServiceUnavailable || appErr.Code != apperrors.ErrCodeObjectStorageUnavailable {
				t.Fatalf("PresignUpload() error = %#v, want stable 503 object storage AppError", err)
			}
			if appErr.Details != "" || appErr.Internal != nil {
				t.Fatalf("PresignUpload() exposed internal failure: %#v", appErr)
			}
		})
	}
}
