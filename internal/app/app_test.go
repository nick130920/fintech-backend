package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nick130920/fintech-backend/configs"
)

type testHandler struct{}

func (*testHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func TestNewHTTPServerAppliesServerConfig(t *testing.T) {
	handler := &testHandler{}
	serverConfig := configs.ServerConfig{
		Port:              "9090",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	server := newHTTPServer(handler, serverConfig)

	if server.Addr != ":9090" {
		t.Errorf("Addr = %q; want %q", server.Addr, ":9090")
	}
	if server.Handler != handler {
		t.Error("Handler was not preserved")
	}
	if server.ReadHeaderTimeout != serverConfig.ReadHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %s; want %s", server.ReadHeaderTimeout, serverConfig.ReadHeaderTimeout)
	}
	if server.ReadTimeout != serverConfig.ReadTimeout {
		t.Errorf("ReadTimeout = %s; want %s", server.ReadTimeout, serverConfig.ReadTimeout)
	}
	if server.WriteTimeout != serverConfig.WriteTimeout {
		t.Errorf("WriteTimeout = %s; want %s", server.WriteTimeout, serverConfig.WriteTimeout)
	}
	if server.IdleTimeout != serverConfig.IdleTimeout {
		t.Errorf("IdleTimeout = %s; want %s", server.IdleTimeout, serverConfig.IdleTimeout)
	}
	if server.MaxHeaderBytes != serverConfig.MaxHeaderBytes {
		t.Errorf("MaxHeaderBytes = %d; want %d", server.MaxHeaderBytes, serverConfig.MaxHeaderBytes)
	}

	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
}
