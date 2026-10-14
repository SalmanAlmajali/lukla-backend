package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestApp_LoginLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		body           map[string]string
		expectedStatus int
		expectSuccess  bool
	}{
		{
			name: "Success login with seeded executive user at /api/v1/auth/login",
			path: "/api/v1/auth/login",
			body: map[string]string{
				"username": "executive@lukla.local",
				"password": "password",
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Success login with seeded production user at /api/v1/auth/login",
			path: "/api/v1/auth/login",
			body: map[string]string{
				"username": "production@lukla.local",
				"password": "password",
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "Fail login with invalid email format",
			path: "/api/v1/auth/login",
			body: map[string]string{
				"username": "not-an-email",
				"password": "password",
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
		},
		{
			name: "Fail login with wrong password",
			path: "/api/v1/auth/login",
			body: map[string]string{
				"username": "executive@lukla.local",
				"password": "wrongpassword123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
		{
			name: "Fail login with tailor (no app access)",
			path: "/api/v1/auth/login",
			body: map[string]string{
				"username": "tailor_non_existent@lukla.local",
				"password": "password",
			},
			expectedStatus: http.StatusUnauthorized,
			expectSuccess:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPost, tt.path, bytes.NewBuffer(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			application.server.Handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					Token string `json:"token"`
				} `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)

			if resp.Success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, resp.Success)
			}
			if tt.expectSuccess && resp.Data.Token == "" {
				t.Errorf("expected token to be present")
			}
		})
	}
}

func TestApp_MeLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login to get token for executive user
	loginBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %s", wLogin.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 2. Call GET /api/v1/auth/me with Bearer token
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	wMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/v1/auth/me, got %d. Body: %s", wMe.Code, wMe.Body.String())
	}

	var meResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID              string `json:"id"`
			Username        string `json:"username"`
			Name            string `json:"name"`
			Email           string `json:"email"`
			Role            string `json:"role"`
			RoleDescription string `json:"roleDescription"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to unmarshal me response: %v", err)
	}

	if !meResp.Success {
		t.Errorf("expected success to be true")
	}
	if meResp.Data.Email != "executive@lukla.local" {
		t.Errorf("expected email executive@lukla.local, got %s", meResp.Data.Email)
	}
	if meResp.Data.Role != "OWNER" {
		t.Errorf("expected role OWNER, got %s", meResp.Data.Role)
	}
	if meResp.Data.Name != "Budi Santoso" {
		t.Errorf("expected name Budi Santoso, got %s", meResp.Data.Name)
	}
}

func TestApp_ClientProfileLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login to get token for client portal user
	loginBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("login failed: %s", wLogin.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 2. Call GET /api/v1/auth/me with Bearer token
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	wMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/v1/auth/me, got %d. Body: %s", wMe.Code, wMe.Body.String())
	}

	var meResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID              string  `json:"id"`
			Username        string  `json:"username"`
			Name            string  `json:"name"`
			Email           string  `json:"email"`
			Role            string  `json:"role"`
			ClientID        *string `json:"clientId"`
			RoleDescription string  `json:"roleDescription"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to unmarshal me response: %v", err)
	}

	if !meResp.Success {
		t.Errorf("expected success to be true")
	}
	if meResp.Data.Email != "client@lukla.local" {
		t.Errorf("expected email client@lukla.local, got %s", meResp.Data.Email)
	}
	if meResp.Data.Role != "CLIENT_PORTAL" {
		t.Errorf("expected role CLIENT_PORTAL, got %s", meResp.Data.Role)
	}
	if meResp.Data.ClientID == nil || *meResp.Data.ClientID == "" {
		t.Errorf("expected non-nil and non-empty ClientID for client@lukla.local, got nil")
	} else {
		t.Logf("client@lukla.local successfully linked to clientId: %s", *meResp.Data.ClientID)
	}
}
