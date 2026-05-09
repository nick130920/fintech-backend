package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/nick130920/fintech-backend/pkg/validator"
)

// newTripHandlerForTest crea un handler con dependencias nulas para validar
// las rutas de validación que no llegan a invocar use cases.
func newTripHandlerForTest() *TripHandler {
	return &TripHandler{
		validator: validator.New(),
		logger:    zerolog.Nop(),
	}
}

func TestTripHandler_CreateTrip_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTripHandlerForTest()

	router := gin.New()
	router.POST("/trips", handler.CreateTrip)

	req := httptest.NewRequest(http.MethodPost, "/trips", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTripHandler_CreateTrip_ValidationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTripHandlerForTest()

	router := gin.New()
	router.POST("/trips", handler.CreateTrip)

	body, _ := json.Marshal(map[string]any{
		"name": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if resp["code"] == nil {
		t.Fatalf("expected error code in response: %v", resp)
	}
}

func TestTripHandler_ParseUintParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newTripHandlerForTest()

	router := gin.New()
	router.GET("/items/:id", func(c *gin.Context) {
		_, ok := handler.parseUintParam(c, "id")
		if ok {
			c.Status(http.StatusOK)
			return
		}
	})

	tests := []struct {
		path string
		code int
	}{
		{path: "/items/42", code: http.StatusOK},
		{path: "/items/notanumber", code: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.code {
				t.Fatalf("expected %d, got %d", tt.code, rec.Code)
			}
		})
	}
}
