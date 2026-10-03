package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/configs"
	"github.com/nick130920/fintech-backend/internal/storage"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/pkg/auth"
)

func TestUploadHandlerPresignUploadRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authMiddleware := NewAuthMiddleware(auth.NewJWTManager("test-secret", time.Hour))
	router.Use(authMiddleware.RequireAuth())
	router.POST("/api/v1/uploads/presign", NewUploadHandler(nil).PresignUpload)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":1}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestRouterRegistersPresignUploadAsProtectedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewRouter(
		router,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		usecase.NewUploadUseCase(nil, configs.UploadConfig{MaxSize: 100, AllowedTypes: []string{"image/jpeg"}, SignedURLTTLSeconds: 60}, nil),
		nil, nil, auth.NewJWTManager("test-secret", time.Hour), zerolog.Nop(),
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":1}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want protected-route status %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestUploadHandlerPresignUploadUsesAuthenticatedUserScopedKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &storage.FakeObjectStore{UploadResult: storage.PresignedUpload{URL: "https://objects.example.test/upload", Headers: map[string]string{"Content-Type": "image/jpeg"}}}
	uploadUC := usecase.NewUploadUseCase(store, configs.UploadConfig{
		MaxSize:             100,
		AllowedTypes:        []string{"image/jpeg"},
		SignedURLTTLSeconds: 60,
	}, func() (string, error) { return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil })
	handler := NewUploadHandler(uploadUC)

	router := gin.New()
	router.POST("/api/v1/uploads/presign", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		handler.PresignUpload(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":10,"key":"attacker-key","bucket":"attacker-bucket"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if len(store.UploadRequests) != 1 {
		t.Fatalf("PresignUpload calls = %d, want 1", len(store.UploadRequests))
	}
	if got := store.UploadRequests[0].Key; got != "users/7/018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d" {
		t.Errorf("Key = %q, want server-owned user-scoped key", got)
	}
	if strings.Contains(recorder.Body.String(), "attacker-key") || strings.Contains(recorder.Body.String(), "attacker-bucket") {
		t.Errorf("response includes client-supplied storage location: %s", recorder.Body.String())
	}
}

func TestUploadHandlerPresignUploadRejectsNonJSONContentTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &storage.FakeObjectStore{}
	handler := NewUploadHandler(usecase.NewUploadUseCase(store, configs.UploadConfig{
		MaxSize:             100,
		AllowedTypes:        []string{"image/jpeg"},
		SignedURLTTLSeconds: 60,
	}, nil))

	router := gin.New()
	router.POST("/api/v1/uploads/presign", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		handler.PresignUpload(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":10}`))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=not-a-file")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if len(store.UploadRequests) != 0 {
		t.Fatalf("PresignUpload calls = %d, want 0", len(store.UploadRequests))
	}
}

func TestUploadHandlerPresignUploadReturnsUnavailableWhenStorageIsAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUploadHandler(usecase.NewUploadUseCase(nil, configs.UploadConfig{
		MaxSize:             100,
		AllowedTypes:        []string{"image/jpeg"},
		SignedURLTTLSeconds: 60,
	}, nil))

	router := gin.New()
	router.POST("/api/v1/uploads/presign", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		handler.PresignUpload(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":10}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(recorder.Body.String(), "OBJECT_STORAGE_UNAVAILABLE") {
		t.Errorf("response = %s, want stable unavailable code", recorder.Body.String())
	}
}

func TestUploadHandlerPresignUploadDoesNotExposeProviderFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewUploadHandler(usecase.NewUploadUseCase(&storage.FakeObjectStore{UploadErr: errProviderFailure{}}, configs.UploadConfig{
		MaxSize:             100,
		AllowedTypes:        []string{"image/jpeg"},
		SignedURLTTLSeconds: 60,
	}, func() (string, error) { return "018f2d4a-6b9d-7cc8-9aaf-789c1a2b3c4d", nil }))

	router := gin.New()
	router.POST("/api/v1/uploads/presign", func(c *gin.Context) {
		c.Set("user_id", uint(7))
		handler.PresignUpload(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/presign", strings.NewReader(`{"content_type":"image/jpeg","content_length":10}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(recorder.Body.String(), "provider secret failure") {
		t.Errorf("provider failure leaked: %s", recorder.Body.String())
	}
}

type errProviderFailure struct{}

func (errProviderFailure) Error() string { return "provider secret failure" }
