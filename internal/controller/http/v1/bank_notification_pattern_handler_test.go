package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nick130920/fintech-backend/internal/controller/http/v1/dto"
	"github.com/nick130920/fintech-backend/internal/usecase"
	"github.com/nick130920/fintech-backend/internal/usecase/webapi"
	"github.com/rs/zerolog"
)

func TestAIEndpoints_ReturnServiceUnavailableWhenNoProviderIsConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	aiService, err := webapi.NewAIServiceWithFallback()
	if err != nil {
		t.Fatalf("NewAIServiceWithFallback() error = %v", err)
	}
	uc := usecase.NewBankNotificationPatternUseCase(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, aiService, nil)
	handler := NewBankNotificationPatternHandler(uc, zerolog.Nop())

	endpoints := []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{name: "process notification", handler: handler.ProcessNotification},
		{name: "process SMS", handler: handler.ProcessSMSWithAI},
		{name: "process SMS batch", handler: handler.ProcessSMSBatchWithAI},
		{name: "analyze SMS batch", handler: handler.AnalyzeSMSBatch},
		{name: "create SMS batch job", handler: handler.StartAnalyzeSMSBatchJob},
		{name: "analyze statement", handler: handler.AnalyzeStatement},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
			ctx.Set("user_id", uint(1))

			endpoint.handler(ctx)

			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d; want %d", recorder.Code, http.StatusServiceUnavailable)
			}
			var response dto.ErrorResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Code != "AI_SERVICE_UNAVAILABLE" {
				t.Fatalf("error code = %q; want %q", response.Code, "AI_SERVICE_UNAVAILABLE")
			}
		})
	}
}
