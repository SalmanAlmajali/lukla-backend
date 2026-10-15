package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukla/operations-engine/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestApp_CORS_PreflightAndHeaders(t *testing.T) {
	ctx := context.Background()
	application, err := app.New(ctx)
	if err != nil {
		t.Skipf("skipping test if db connection fails in test env: %v", err)
		return
	}

	handler := application.Handler()

	// 1. Test Preflight OPTIONS request
	reqOptions, err := http.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	assert.NoError(t, err)
	reqOptions.Header.Set("Origin", "http://localhost:5173")
	reqOptions.Header.Set("Access-Control-Request-Method", "POST")
	reqOptions.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	wOptions := httptest.NewRecorder()
	handler.ServeHTTP(wOptions, reqOptions)

	assert.Equal(t, http.StatusNoContent, wOptions.Code)
	assert.Equal(t, "http://localhost:5173", wOptions.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", wOptions.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, wOptions.Header().Get("Access-Control-Allow-Headers"), "authorization")

	// 2. Test simple GET request with Origin
	reqGet, err := http.NewRequest(http.MethodGet, "/health", nil)
	assert.NoError(t, err)
	reqGet.Header.Set("Origin", "http://localhost:3000")

	wGet := httptest.NewRecorder()
	handler.ServeHTTP(wGet, reqGet)

	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Equal(t, "http://localhost:3000", wGet.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", wGet.Header().Get("Access-Control-Allow-Credentials"))
}

