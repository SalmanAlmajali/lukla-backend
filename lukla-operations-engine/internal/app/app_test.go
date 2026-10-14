package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

func TestApp_LogoutLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login first to get JWT token
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

	// 2. Call POST /api/v1/auth/logout with Bearer token
	reqLogout, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.Header.Set("Authorization", "Bearer "+token)
	wLogout := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogout, reqLogout)

	if wLogout.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/v1/auth/logout, got %d. Body: %s", wLogout.Code, wLogout.Body.String())
	}

	var logoutResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(wLogout.Body.Bytes(), &logoutResp); err != nil {
		t.Fatalf("failed to unmarshal logout response: %v", err)
	}

	if !logoutResp.Success {
		t.Errorf("expected success to be true")
	}
	if logoutResp.Message != "Berhasil logout." {
		t.Errorf("expected message 'Berhasil logout.', got %s", logoutResp.Message)
	}
}

func TestApp_LogoutUnauthorizedLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	reqLogout, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	wLogout := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogout, reqLogout)

	if wLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing token on logout, got %d. Body: %s", wLogout.Code, wLogout.Body.String())
	}
}

func TestApp_ListClientsLiveDB_Pagination(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 0. Login to get token
	loginBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogin, reqLogin)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 1. Test Page 1 with limit 10
	reqP1, _ := http.NewRequest(http.MethodGet, "/api/v1/clients?page=1&limit=10", nil)
	reqP1.Header.Set("Authorization", "Bearer "+token)
	wP1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wP1, reqP1)

	if wP1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for page 1, got %d. Body: %s", wP1.Code, wP1.Body.String())
	}

	var respP1 struct {
		Success bool `json:"success"`
		Data    []struct {
			ID         string `json:"id"`
			Code       string `json:"code"`
			Name       string `json:"name"`
			ClientType string `json:"clientType"`
		} `json:"data"`
		Meta struct {
			Page       int   `json:"page"`
			Limit      int   `json:"limit"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"totalPages"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(wP1.Body.Bytes(), &respP1); err != nil {
		t.Fatalf("failed to decode page 1 response: %v", err)
	}

	if !respP1.Success {
		t.Errorf("expected success true")
	}
	if respP1.Meta.Total < 12 {
		t.Errorf("expected total at least 12 clients, got %d", respP1.Meta.Total)
	}
	if respP1.Meta.Page != 1 {
		t.Errorf("expected page 1, got %d", respP1.Meta.Page)
	}
	if respP1.Meta.Limit != 10 {
		t.Errorf("expected limit 10, got %d", respP1.Meta.Limit)
	}
	if len(respP1.Data) != 10 {
		t.Errorf("expected 10 items on page 1, got %d", len(respP1.Data))
	}
	if respP1.Meta.TotalPages < 2 {
		t.Errorf("expected totalPages >= 2, got %d", respP1.Meta.TotalPages)
	}

	// 2. Test Page 2 with limit 10
	reqP2, _ := http.NewRequest(http.MethodGet, "/api/v1/clients?page=2&limit=10", nil)
	reqP2.Header.Set("Authorization", "Bearer "+token)
	wP2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wP2, reqP2)

	if wP2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for page 2, got %d. Body: %s", wP2.Code, wP2.Body.String())
	}

	var respP2 struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
		Meta struct {
			Page       int   `json:"page"`
			Limit      int   `json:"limit"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"totalPages"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(wP2.Body.Bytes(), &respP2); err != nil {
		t.Fatalf("failed to decode page 2 response: %v", err)
	}

	if respP2.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", respP2.Meta.Page)
	}
	if len(respP2.Data) < 2 {
		t.Errorf("expected at least 2 items on page 2, got %d", len(respP2.Data))
	}
}

func TestApp_ListClientsLiveDB_Search(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 0. Login to get token
	loginBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogin, reqLogin)

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/clients?search=SNOHUND", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 client for search SNOHUND, got %d", len(resp.Data))
	}
	if resp.Data[0].Code != "SNOHUND" {
		t.Errorf("expected code SNOHUND, got %s", resp.Data[0].Code)
	}
}

func TestApp_ListClientsUnauthorizedLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	w := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestApp_ListClientsForbiddenClientRoleLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login with client portal account (client@lukla.local)
	loginBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("client login failed: %s", wLogin.Body.String())
	}

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 2. Client role attempts to access GET /api/v1/clients -> MUST be 403 Forbidden!
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client role on GET /api/v1/clients, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Success {
		t.Errorf("expected success false")
	}
	if resp.Error.Code != "FORBIDDEN" {
		t.Errorf("expected error code FORBIDDEN, got %s", resp.Error.Code)
	}
}

func TestApp_GetClientDetail_LiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login Executive (Internal)
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Login Client 1 (Snøhund)
	loginSnohundBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqSnohundLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginSnohundBody))
	reqSnohundLogin.Header.Set("Content-Type", "application/json")
	wSnohundLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundLogin, reqSnohundLogin)

	var snohundLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSnohundLogin.Body.Bytes(), &snohundLoginResp)
	snohundToken := snohundLoginResp.Data.Token

	// Get Snøhund client ID via auth/me
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+snohundToken)
	wMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMe, reqMe)

	var meResp struct {
		Data struct {
			ClientID *string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wMe.Body.Bytes(), &meResp)
	if meResp.Data.ClientID == nil {
		t.Fatalf("expected non-nil ClientID for Snohund client user")
	}
	snohundClientID := *meResp.Data.ClientID

	// 3. Login Client 2 (Aventura)
	loginAventuraBody, _ := json.Marshal(map[string]string{
		"username": "client.aventura@lukla.local",
		"password": "password",
	})
	reqAventuraLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginAventuraBody))
	reqAventuraLogin.Header.Set("Content-Type", "application/json")
	wAventuraLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAventuraLogin, reqAventuraLogin)

	var aventuraLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAventuraLogin.Body.Bytes(), &aventuraLoginResp)
	aventuraToken := aventuraLoginResp.Data.Token

	reqAventuraMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqAventuraMe.Header.Set("Authorization", "Bearer "+aventuraToken)
	wAventuraMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAventuraMe, reqAventuraMe)

	var aventuraMeResp struct {
		Data struct {
			ClientID *string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAventuraMe.Body.Bytes(), &aventuraMeResp)
	if aventuraMeResp.Data.ClientID == nil {
		t.Fatalf("expected non-nil ClientID for Aventura client user")
	}
	aventuraClientID := *aventuraMeResp.Data.ClientID

	// Test A: Internal Executive can view Snøhund
	reqGetSnohundByExec, _ := http.NewRequest(http.MethodGet, "/api/v1/clients/"+snohundClientID, nil)
	reqGetSnohundByExec.Header.Set("Authorization", "Bearer "+execToken)
	wGetSnohundByExec := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wGetSnohundByExec, reqGetSnohundByExec)

	if wGetSnohundByExec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for exec viewing snohund, got %d. Body: %s", wGetSnohundByExec.Code, wGetSnohundByExec.Body.String())
	}

	// Test B: Snøhund can view own data
	reqGetSnohundBySnohund, _ := http.NewRequest(http.MethodGet, "/api/v1/clients/"+snohundClientID, nil)
	reqGetSnohundBySnohund.Header.Set("Authorization", "Bearer "+snohundToken)
	wGetSnohundBySnohund := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wGetSnohundBySnohund, reqGetSnohundBySnohund)

	if wGetSnohundBySnohund.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for snohund viewing own data, got %d. Body: %s", wGetSnohundBySnohund.Code, wGetSnohundBySnohund.Body.String())
	}

	// Test C: Snøhund tries to view Aventura data -> 403 Forbidden!
	reqGetAventuraBySnohund, _ := http.NewRequest(http.MethodGet, "/api/v1/clients/"+aventuraClientID, nil)
	reqGetAventuraBySnohund.Header.Set("Authorization", "Bearer "+snohundToken)
	wGetAventuraBySnohund := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wGetAventuraBySnohund, reqGetAventuraBySnohund)

	if wGetAventuraBySnohund.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client viewing other client data, got %d. Body: %s", wGetAventuraBySnohund.Code, wGetAventuraBySnohund.Body.String())
	}

	// Test D: Non-existent UUID -> 404 Not Found
	reqNotFound, _ := http.NewRequest(http.MethodGet, "/api/v1/clients/00000000-0000-0000-0000-000000000000", nil)
	reqNotFound.Header.Set("Authorization", "Bearer "+execToken)
	wNotFound := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent client, got %d. Body: %s", wNotFound.Code, wNotFound.Body.String())
	}
}

func TestApp_ArticlesLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login Executive
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Login Client Snøhund
	loginSnohundBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqSnohundLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginSnohundBody))
	reqSnohundLogin.Header.Set("Content-Type", "application/json")
	wSnohundLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundLogin, reqSnohundLogin)

	var snohundLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSnohundLogin.Body.Bytes(), &snohundLoginResp)
	snohundToken := snohundLoginResp.Data.Token

	// Get Snøhund client ID
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+snohundToken)
	wMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMe, reqMe)
	var meResp struct {
		Data struct {
			ClientID *string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wMe.Body.Bytes(), &meResp)
	snohundClientID := *meResp.Data.ClientID

	// 3. Login Client Aventura
	loginAventuraBody, _ := json.Marshal(map[string]string{
		"username": "client.aventura@lukla.local",
		"password": "password",
	})
	reqAventuraLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginAventuraBody))
	reqAventuraLogin.Header.Set("Content-Type", "application/json")
	wAventuraLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAventuraLogin, reqAventuraLogin)

	var aventuraLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAventuraLogin.Body.Bytes(), &aventuraLoginResp)
	aventuraToken := aventuraLoginResp.Data.Token

	reqAveMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqAveMe.Header.Set("Authorization", "Bearer "+aventuraToken)
	wAveMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAveMe, reqAveMe)
	var aveMeResp struct {
		Data struct {
			ClientID *string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAveMe.Body.Bytes(), &aveMeResp)
	aventuraClientID := *aveMeResp.Data.ClientID

	// Test 1: Executive lists all articles
	reqAllArticles, _ := http.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	reqAllArticles.Header.Set("Authorization", "Bearer "+execToken)
	wAllArticles := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAllArticles, reqAllArticles)

	if wAllArticles.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for exec listing articles, got %d. Body: %s", wAllArticles.Code, wAllArticles.Body.String())
	}

	var allArticlesResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID         string `json:"id"`
			ClientID   string `json:"clientId"`
			ClientName string `json:"clientName"`
			Title      string `json:"title"`
			Code       string `json:"code"`
			Category   string `json:"category"`
		} `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(wAllArticles.Body.Bytes(), &allArticlesResp); err != nil {
		t.Fatalf("failed to decode all articles response: %v", err)
	}

	if allArticlesResp.Meta.Total < 5 {
		t.Errorf("expected at least 5 seeded articles, got %d", allArticlesResp.Meta.Total)
	}

	// Test 2: Filter by category GARMENT
	reqGarment, _ := http.NewRequest(http.MethodGet, "/api/v1/articles?category=GARMENT", nil)
	reqGarment.Header.Set("Authorization", "Bearer "+execToken)
	wGarment := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wGarment, reqGarment)

	if wGarment.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for category GARMENT, got %d", wGarment.Code)
	}
	var garmentResp struct {
		Data []struct {
			Category string `json:"category"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wGarment.Body.Bytes(), &garmentResp)
	for _, art := range garmentResp.Data {
		if art.Category != "GARMENT" {
			t.Errorf("expected all articles to be GARMENT, got %s", art.Category)
		}
	}

	// Test 3: Filter by category BAG
	reqBag, _ := http.NewRequest(http.MethodGet, "/api/v1/articles?category=BAG", nil)
	reqBag.Header.Set("Authorization", "Bearer "+execToken)
	wBag := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wBag, reqBag)

	if wBag.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for category BAG, got %d", wBag.Code)
	}
	var bagResp struct {
		Data []struct {
			Category string `json:"category"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wBag.Body.Bytes(), &bagResp)
	for _, art := range bagResp.Data {
		if art.Category != "BAG" {
			t.Errorf("expected all articles to be BAG, got %s", art.Category)
		}
	}

	// Test 4a: Client Snøhund attempts to query general articles without clientId -> 403 Forbidden!
	reqSnohundGeneral, _ := http.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	reqSnohundGeneral.Header.Set("Authorization", "Bearer "+snohundToken)
	wSnohundGeneral := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundGeneral, reqSnohundGeneral)

	if wSnohundGeneral.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for snohund querying general articles without clientId, got %d. Body: %s", wSnohundGeneral.Code, wSnohundGeneral.Body.String())
	}

	// Test 4b: Client Snøhund lists its own articles by supplying clientId -> 200 OK!
	reqSnohundArticles, _ := http.NewRequest(http.MethodGet, "/api/v1/articles?clientId="+snohundClientID, nil)
	reqSnohundArticles.Header.Set("Authorization", "Bearer "+snohundToken)
	wSnohundArticles := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundArticles, reqSnohundArticles)

	if wSnohundArticles.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for snohund listing own articles, got %d. Body: %s", wSnohundArticles.Code, wSnohundArticles.Body.String())
	}

	var snohundArticlesResp struct {
		Data []struct {
			ID       string `json:"id"`
			ClientID string `json:"clientId"`
			Code     string `json:"code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSnohundArticles.Body.Bytes(), &snohundArticlesResp)
	if len(snohundArticlesResp.Data) < 2 {
		t.Fatalf("expected at least 2 articles for Snøhund, got %d", len(snohundArticlesResp.Data))
	}
	var snohundArticleID string
	for _, art := range snohundArticlesResp.Data {
		if art.ClientID != snohundClientID {
			t.Errorf("security breach: client received article for client %s, expected %s", art.ClientID, snohundClientID)
		}
		if art.Code == "ART-SNO-PK01" {
			snohundArticleID = art.ID
		}
	}

	// Test 5: Client Snøhund attempts to query Aventura articles via clientId param -> 403 Forbidden!
	reqForbiddenClientID, _ := http.NewRequest(http.MethodGet, "/api/v1/articles?clientId="+aventuraClientID, nil)
	reqForbiddenClientID.Header.Set("Authorization", "Bearer "+snohundToken)
	wForbiddenClientID := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wForbiddenClientID, reqForbiddenClientID)

	if wForbiddenClientID.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client queries another client's articles, got %d", wForbiddenClientID.Code)
	}

	// Test 6: Snøhund views detail of its own article -> 200 OK
	if snohundArticleID != "" {
		reqArticleDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+snohundArticleID, nil)
		reqArticleDetail.Header.Set("Authorization", "Bearer "+snohundToken)
		wArticleDetail := httptest.NewRecorder()
		application.server.Handler.ServeHTTP(wArticleDetail, reqArticleDetail)

		if wArticleDetail.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for article detail, got %d. Body: %s", wArticleDetail.Code, wArticleDetail.Body.String())
		}
	}

	// Test 7: Snøhund views detail of Aventura's article -> 403 Forbidden!
	reqAventuraArticles, _ := http.NewRequest(http.MethodGet, "/api/v1/articles?clientId="+aventuraClientID, nil)
	reqAventuraArticles.Header.Set("Authorization", "Bearer "+aventuraToken)
	wAventuraArticles := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAventuraArticles, reqAventuraArticles)
	var aveArticlesResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAventuraArticles.Body.Bytes(), &aveArticlesResp)
	if len(aveArticlesResp.Data) > 0 {
		aveArticleID := aveArticlesResp.Data[0].ID
		reqSnohundSneak, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+aveArticleID, nil)
		reqSnohundSneak.Header.Set("Authorization", "Bearer "+snohundToken)
		wSnohundSneak := httptest.NewRecorder()
		application.server.Handler.ServeHTTP(wSnohundSneak, reqSnohundSneak)

		if wSnohundSneak.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden when client accesses other client's article detail, got %d", wSnohundSneak.Code)
		}
	}

	// Test 8: Verify Client detail reflects calculated articlesCount
	reqClientDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/clients/"+snohundClientID, nil)
	reqClientDetail.Header.Set("Authorization", "Bearer "+snohundToken)
	wClientDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientDetail, reqClientDetail)

	if wClientDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for client detail, got %d", wClientDetail.Code)
	}
	var clientDetailResp struct {
		Data struct {
			ArticlesCount int `json:"articlesCount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientDetail.Body.Bytes(), &clientDetailResp)
	if clientDetailResp.Data.ArticlesCount < 2 {
		t.Errorf("expected articlesCount >= 2 for Snøhund, got %d", clientDetailResp.Data.ArticlesCount)
	}

	// Test 9: Unauthenticated request to /api/v1/articles -> 401 Unauthorized!
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	wNoAuth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNoAuth, reqNoAuth)

	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for request without token, got %d", wNoAuth.Code)
	}
}

func TestApp_ComponentsLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login Executive
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Login Client Snøhund
	loginSnohundBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqSnohundLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginSnohundBody))
	reqSnohundLogin.Header.Set("Content-Type", "application/json")
	wSnohundLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundLogin, reqSnohundLogin)

	var snohundLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSnohundLogin.Body.Bytes(), &snohundLoginResp)
	snohundToken := snohundLoginResp.Data.Token

	// 3. Find Snøhund Parka Article ID and Aventura Article ID
	reqArticles, _ := http.NewRequest(http.MethodGet, "/api/v1/articles", nil)
	reqArticles.Header.Set("Authorization", "Bearer "+execToken)
	wArticles := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wArticles, reqArticles)

	var articlesResp struct {
		Data []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wArticles.Body.Bytes(), &articlesResp)

	var snohundParkaID, aventuraPantsID string
	for _, a := range articlesResp.Data {
		if a.Code == "ART-SNO-PK01" {
			snohundParkaID = a.ID
		} else if a.Code == "ART-AVE-TR01" {
			aventuraPantsID = a.ID
		}
	}

	if snohundParkaID == "" || aventuraPantsID == "" {
		t.Fatalf("expected seeded articles ART-SNO-PK01 and ART-AVE-TR01 to exist")
	}

	// 4. Executive lists components of Snøhund Parka
	reqComps, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+snohundParkaID+"/components", nil)
	reqComps.Header.Set("Authorization", "Bearer "+execToken)
	wComps := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wComps, reqComps)

	if wComps.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list components, got %d. Body: %s", wComps.Code, wComps.Body.String())
	}

	var compsResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID             string   `json:"id"`
			ArticleID      string   `json:"articleId"`
			Name           string   `json:"name"`
			Code           *string  `json:"code"`
			SequenceNumber int      `json:"sequenceNumber"`
			StandardSAM    *float64 `json:"standardSAM"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wComps.Body.Bytes(), &compsResp); err != nil {
		t.Fatalf("failed to decode components response: %v", err)
	}

	if len(compsResp.Data) < 5 {
		t.Errorf("expected at least 5 seeded components for Snøhund Parka, got %d", len(compsResp.Data))
	}
	// Verify that StandardSAM is present (Time Study measure)
	hasSAM := false
	for _, c := range compsResp.Data {
		if c.StandardSAM != nil && *c.StandardSAM > 0 {
			hasSAM = true
			break
		}
	}
	if !hasSAM {
		t.Errorf("expected standardSAM values to be present on seeded components")
	}

	// 5. Snøhund Client lists components of its own article -> 200 OK
	reqSnohundComps, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+snohundParkaID+"/components", nil)
	reqSnohundComps.Header.Set("Authorization", "Bearer "+snohundToken)
	wSnohundComps := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundComps, reqSnohundComps)

	if wSnohundComps.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for Snøhund accessing own components, got %d", wSnohundComps.Code)
	}

	// 6. Snøhund Client attempts to list components of Aventura article -> 403 Forbidden!
	reqForbiddenComps, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+aventuraPantsID+"/components", nil)
	reqForbiddenComps.Header.Set("Authorization", "Bearer "+snohundToken)
	wForbiddenComps := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wForbiddenComps, reqForbiddenComps)

	if wForbiddenComps.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Snøhund accessing Aventura components, got %d", wForbiddenComps.Code)
	}

	// 7. Executive creates a new component on Snøhund Parka
	newCompBody := `{"name": "QC Akhir & Labeling", "code": "CMP-SNO-PK-06", "sequenceNumber": 6, "standardSAM": 1.75}`
	reqCreateComp, _ := http.NewRequest(http.MethodPost, "/api/v1/articles/"+snohundParkaID+"/components", bytes.NewBufferString(newCompBody))
	reqCreateComp.Header.Set("Authorization", "Bearer "+execToken)
	reqCreateComp.Header.Set("Content-Type", "application/json")
	wCreateComp := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCreateComp, reqCreateComp)

	if wCreateComp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for creating component, got %d. Body: %s", wCreateComp.Code, wCreateComp.Body.String())
	}

	// 8. Snøhund Client attempts to add component to Aventura article -> 403 Forbidden!
	reqSneakComp, _ := http.NewRequest(http.MethodPost, "/api/v1/articles/"+aventuraPantsID+"/components", bytes.NewBufferString(newCompBody))
	reqSneakComp.Header.Set("Authorization", "Bearer "+snohundToken)
	reqSneakComp.Header.Set("Content-Type", "application/json")
	wSneakComp := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSneakComp, reqSneakComp)

	if wSneakComp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client adds component to other client's article, got %d", wSneakComp.Code)
	}

	// 9. Verify GET /api/v1/articles/:id returns components inside ArticleDetail
	reqDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/articles/"+snohundParkaID, nil)
	reqDetail.Header.Set("Authorization", "Bearer "+execToken)
	wDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wDetail, reqDetail)

	if wDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for article detail, got %d", wDetail.Code)
	}

	var detailResp struct {
		Data struct {
			ID         string `json:"id"`
			Components []struct {
				Name string `json:"name"`
			} `json:"components"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wDetail.Body.Bytes(), &detailResp)
	if len(detailResp.Data.Components) < 5 {
		t.Errorf("expected ArticleDetail to contain at least 5 components, got %d", len(detailResp.Data.Components))
	}
}

func TestApp_OperatorsLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login Executive (Internal Staff)
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Login Client Snøhund (Client Portal)
	loginSnohundBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqSnohundLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginSnohundBody))
	reqSnohundLogin.Header.Set("Content-Type", "application/json")
	wSnohundLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSnohundLogin, reqSnohundLogin)

	var snohundLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSnohundLogin.Body.Bytes(), &snohundLoginResp)
	snohundToken := snohundLoginResp.Data.Token

	// Test 1: Executive lists all operators -> 200 OK
	reqAllOps, _ := http.NewRequest(http.MethodGet, "/api/v1/operators", nil)
	reqAllOps.Header.Set("Authorization", "Bearer "+execToken)
	wAllOps := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAllOps, reqAllOps)

	if wAllOps.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for exec listing operators, got %d. Body: %s", wAllOps.Code, wAllOps.Body.String())
	}

	var allOpsResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID           string  `json:"id"`
			OperatorCode string  `json:"operatorCode"`
			Name         string  `json:"name"`
			Status       string  `json:"status"`
			DefaultLine  *string `json:"defaultLine"`
			SkillTier    *string `json:"skillTier"`
			JoinDate     string  `json:"joinDate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wAllOps.Body.Bytes(), &allOpsResp); err != nil {
		t.Fatalf("failed to decode all operators: %v", err)
	}

	if len(allOpsResp.Data) < 9 {
		t.Fatalf("expected at least 9 seeded operators, got %d", len(allOpsResp.Data))
	}

	// Verify presence of mid-week joiner JH-008 (Skenario Uji SRS 2)
	foundMidWeekJoiner := false
	for _, op := range allOpsResp.Data {
		if op.OperatorCode == "JH-008" && op.JoinDate == "2026-10-14" {
			foundMidWeekJoiner = true
			break
		}
	}
	if !foundMidWeekJoiner {
		t.Errorf("expected mid-week joiner operator JH-008 with joinDate 2026-10-14 to be seeded")
	}

	// Test 2: Filter by status ACTIVE
	reqActive, _ := http.NewRequest(http.MethodGet, "/api/v1/operators?status=ACTIVE", nil)
	reqActive.Header.Set("Authorization", "Bearer "+execToken)
	wActive := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wActive, reqActive)

	if wActive.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for status=ACTIVE, got %d", wActive.Code)
	}
	var activeResp struct {
		Data []struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wActive.Body.Bytes(), &activeResp)
	for _, op := range activeResp.Data {
		if op.Status != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %s", op.Status)
		}
	}

	// Test 3: Filter by status ON_LEAVE
	reqOnLeave, _ := http.NewRequest(http.MethodGet, "/api/v1/operators?status=ON_LEAVE", nil)
	reqOnLeave.Header.Set("Authorization", "Bearer "+execToken)
	wOnLeave := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOnLeave, reqOnLeave)

	if wOnLeave.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for status=ON_LEAVE, got %d", wOnLeave.Code)
	}
	var onLeaveResp struct {
		Data []struct {
			OperatorCode string `json:"operatorCode"`
			Status       string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wOnLeave.Body.Bytes(), &onLeaveResp)
	if len(onLeaveResp.Data) == 0 {
		t.Errorf("expected at least 1 ON_LEAVE operator (JH-004)")
	}

	// Test 4: Filter by Line LINE-1
	reqLine1, _ := http.NewRequest(http.MethodGet, "/api/v1/operators?line=LINE-1", nil)
	reqLine1.Header.Set("Authorization", "Bearer "+execToken)
	wLine1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLine1, reqLine1)

	if wLine1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for line=LINE-1, got %d", wLine1.Code)
	}
	var line1Resp struct {
		Data []struct {
			DefaultLine *string `json:"defaultLine"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLine1.Body.Bytes(), &line1Resp)
	for _, op := range line1Resp.Data {
		if op.DefaultLine == nil || *op.DefaultLine != "LINE-1" {
			t.Errorf("expected LINE-1, got %v", op.DefaultLine)
		}
	}

	// Test 5: Search operator by name
	reqSearch, _ := http.NewRequest(http.MethodGet, "/api/v1/operators?search=Asep", nil)
	reqSearch.Header.Set("Authorization", "Bearer "+execToken)
	wSearch := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSearch, reqSearch)

	if wSearch.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for search=Asep, got %d", wSearch.Code)
	}
	var searchResp struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSearch.Body.Bytes(), &searchResp)
	if len(searchResp.Data) == 0 || !strings.Contains(searchResp.Data[0].Name, "Asep") {
		t.Errorf("expected operator search result containing 'Asep'")
	}

	// Test 6: Client Snøhund attempts to access /api/v1/operators -> 403 Forbidden!
	reqClientAccess, _ := http.NewRequest(http.MethodGet, "/api/v1/operators", nil)
	reqClientAccess.Header.Set("Authorization", "Bearer "+snohundToken)
	wClientAccess := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientAccess, reqClientAccess)

	if wClientAccess.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client accesses /operators, got %d. Body: %s", wClientAccess.Code, wClientAccess.Body.String())
	}

	// Test 7: Request without token -> 401 Unauthorized!
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/operators", nil)
	wNoAuth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNoAuth, reqNoAuth)

	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request to /operators, got %d", wNoAuth.Code)
	}
}

func TestApp_GetOperatorDetail_LiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Executive Login
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	if wExecLogin.Code != http.StatusOK {
		t.Fatalf("failed to login executive: %d", wExecLogin.Code)
	}
	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Client Login
	loginClientBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqClientLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginClientBody))
	reqClientLogin.Header.Set("Content-Type", "application/json")
	wClientLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLogin, reqClientLogin)

	if wClientLogin.Code != http.StatusOK {
		t.Fatalf("failed to login client: %d", wClientLogin.Code)
	}
	var clientLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientLogin.Body.Bytes(), &clientLoginResp)
	clientToken := clientLoginResp.Data.Token

	// Test 1: Executive retrieves operator detail by code (JH-001)
	reqByCode, _ := http.NewRequest(http.MethodGet, "/api/v1/operators/JH-001", nil)
	reqByCode.Header.Set("Authorization", "Bearer "+execToken)
	wByCode := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wByCode, reqByCode)

	if wByCode.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /operators/JH-001, got %d. Body: %s", wByCode.Code, wByCode.Body.String())
	}

	var opResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID           string  `json:"id"`
			OperatorCode string  `json:"operatorCode"`
			Name         string  `json:"name"`
			Status       string  `json:"status"`
			DefaultLine  *string `json:"defaultLine"`
			SkillTier    *string `json:"skillTier"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wByCode.Body.Bytes(), &opResp); err != nil {
		t.Fatalf("failed to unmarshal operator response: %v", err)
	}

	if !opResp.Success {
		t.Fatal("expected success true")
	}
	if opResp.Data.OperatorCode != "JH-001" {
		t.Errorf("expected operatorCode JH-001, got %s", opResp.Data.OperatorCode)
	}
	if opResp.Data.Name != "Ahmad Supriatna" {
		t.Errorf("expected name Ahmad Supriatna, got %s", opResp.Data.Name)
	}
	if opResp.Data.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %s", opResp.Data.Status)
	}
	if opResp.Data.DefaultLine == nil || *opResp.Data.DefaultLine != "LINE-1" {
		t.Errorf("expected defaultLine LINE-1, got %v", opResp.Data.DefaultLine)
	}

	operatorUUID := opResp.Data.ID

	// Test 2: Executive retrieves operator detail by UUID
	reqByUUID, _ := http.NewRequest(http.MethodGet, "/api/v1/operators/"+operatorUUID, nil)
	reqByUUID.Header.Set("Authorization", "Bearer "+execToken)
	wByUUID := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wByUUID, reqByUUID)

	if wByUUID.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /operators by UUID, got %d. Body: %s", wByUUID.Code, wByUUID.Body.String())
	}

	// Test 3: Operator not found
	reqNotFound, _ := http.NewRequest(http.MethodGet, "/api/v1/operators/non-existent-operator-id", nil)
	reqNotFound.Header.Set("Authorization", "Bearer "+execToken)
	wNotFound := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d. Body: %s", wNotFound.Code, wNotFound.Body.String())
	}

	// Test 4: Client role attempts access -> 403 Forbidden
	reqClientAccess, _ := http.NewRequest(http.MethodGet, "/api/v1/operators/JH-001", nil)
	reqClientAccess.Header.Set("Authorization", "Bearer "+clientToken)
	wClientAccess := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientAccess, reqClientAccess)

	if wClientAccess.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client role accessing operator detail, got %d. Body: %s", wClientAccess.Code, wClientAccess.Body.String())
	}

	// Test 5: Unauthenticated request -> 401 Unauthorized
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/operators/JH-001", nil)
	wNoAuth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNoAuth, reqNoAuth)

	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", wNoAuth.Code)
	}
}

func TestApp_RateCardsLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Executive Login
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)

	if wExecLogin.Code != http.StatusOK {
		t.Fatalf("failed to login executive: %d", wExecLogin.Code)
	}
	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Production Head Login
	loginProdBody, _ := json.Marshal(map[string]string{
		"username": "production@lukla.local",
		"password": "password",
	})
	reqProdLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginProdBody))
	reqProdLogin.Header.Set("Content-Type", "application/json")
	wProdLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wProdLogin, reqProdLogin)

	if wProdLogin.Code != http.StatusOK {
		t.Fatalf("failed to login production head: %d", wProdLogin.Code)
	}
	var prodLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wProdLogin.Body.Bytes(), &prodLoginResp)
	prodToken := prodLoginResp.Data.Token

	// 3. Client Login (Snøhund)
	loginClientBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqClientLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginClientBody))
	reqClientLogin.Header.Set("Content-Type", "application/json")
	wClientLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLogin, reqClientLogin)

	if wClientLogin.Code != http.StatusOK {
		t.Fatalf("failed to login client: %d", wClientLogin.Code)
	}
	var clientLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientLogin.Body.Bytes(), &clientLoginResp)
	clientToken := clientLoginResp.Data.Token

	// Fetch Client ID for Snøhund
	reqMe, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+clientToken)
	wMe := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMe, reqMe)
	var meResp struct {
		Data struct {
			ClientID *string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wMe.Body.Bytes(), &meResp)
	if meResp.Data.ClientID == nil {
		t.Fatal("expected clientId for client user in /auth/me")
	}
	snohundClientID := *meResp.Data.ClientID

	// Test 1: Executive lists all rate cards -> 200 OK (at least 8 seeded)
	reqAllRates, _ := http.NewRequest(http.MethodGet, "/api/v1/rate-cards", nil)
	reqAllRates.Header.Set("Authorization", "Bearer "+execToken)
	wAllRates := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAllRates, reqAllRates)

	if wAllRates.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /rate-cards, got %d. Body: %s", wAllRates.Code, wAllRates.Body.String())
	}
	var allRatesResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID            string  `json:"id"`
			ClientID      string  `json:"clientId"`
			ArticleID     string  `json:"articleId"`
			ComponentID   string  `json:"componentId"`
			ComponentName string  `json:"componentName"`
			SewingRate    float64 `json:"sewingRate"`
			CuttingRate   float64 `json:"cuttingRate"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAllRates.Body.Bytes(), &allRatesResp)
	if len(allRatesResp.Data) < 8 {
		t.Fatalf("expected at least 8 rate cards, got %d", len(allRatesResp.Data))
	}

	firstRateCardID := allRatesResp.Data[0].ID

	// Test 2: Client lists rate cards -> only sees its own rate cards
	reqClientRates, _ := http.NewRequest(http.MethodGet, "/api/v1/rate-cards", nil)
	reqClientRates.Header.Set("Authorization", "Bearer "+clientToken)
	wClientRates := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientRates, reqClientRates)

	if wClientRates.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for client GET /rate-cards, got %d", wClientRates.Code)
	}
	var clientRatesResp struct {
		Data []struct {
			ClientID string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientRates.Body.Bytes(), &clientRatesResp)
	for _, rc := range clientRatesResp.Data {
		if rc.ClientID != snohundClientID {
			t.Errorf("client received rate card belonging to other client: %s", rc.ClientID)
		}
	}

	// Test 3: Client tries to query other client's rate cards -> 403 Forbidden
	reqForbiddenClient, _ := http.NewRequest(http.MethodGet, "/api/v1/rate-cards?clientId=00000000-0000-0000-0000-000000000000", nil)
	reqForbiddenClient.Header.Set("Authorization", "Bearer "+clientToken)
	wForbiddenClient := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wForbiddenClient, reqForbiddenClient)

	if wForbiddenClient.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client queries other clientId, got %d", wForbiddenClient.Code)
	}

	// Test 4: Client tries to POST /rate-cards -> 403 Forbidden
	dummyRateCardBody, _ := json.Marshal(map[string]any{
		"clientId":    snohundClientID,
		"articleId":   allRatesResp.Data[0].ArticleID,
		"componentId": allRatesResp.Data[0].ComponentID,
		"sewingRate":  5000,
	})
	reqClientPost, _ := http.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewBuffer(dummyRateCardBody))
	reqClientPost.Header.Set("Authorization", "Bearer "+clientToken)
	reqClientPost.Header.Set("Content-Type", "application/json")
	wClientPost := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientPost, reqClientPost)

	if wClientPost.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client attempts to create rate card, got %d", wClientPost.Code)
	}

	// Test 5: Executive updates rate card with changeReason -> 200 OK
	updateRateBody, _ := json.Marshal(map[string]any{
		"sewingRate":   7200.0,
		"cuttingRate":  850.0,
		"changeReason": "Kenaikan biaya benang tahan abrasi dan penyesuaian UMR",
	})
	reqUpdate, _ := http.NewRequest(http.MethodPut, "/api/v1/rate-cards/"+firstRateCardID, bytes.NewBuffer(updateRateBody))
	reqUpdate.Header.Set("Authorization", "Bearer "+execToken)
	reqUpdate.Header.Set("Content-Type", "application/json")
	wUpdate := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wUpdate, reqUpdate)

	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for update rate card, got %d. Body: %s", wUpdate.Code, wUpdate.Body.String())
	}
	var updateResp struct {
		Success bool `json:"success"`
		Data    struct {
			SewingRate  float64 `json:"sewingRate"`
			CuttingRate float64 `json:"cuttingRate"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUpdate.Body.Bytes(), &updateResp)
	if updateResp.Data.SewingRate != 7200.0 {
		t.Errorf("expected sewingRate 7200, got %f", updateResp.Data.SewingRate)
	}

	// Test 6: Update rate card without changeReason -> 400 Bad Request
	noReasonBody, _ := json.Marshal(map[string]any{
		"sewingRate": 7500.0,
	})
	reqNoReason, _ := http.NewRequest(http.MethodPut, "/api/v1/rate-cards/"+firstRateCardID, bytes.NewBuffer(noReasonBody))
	reqNoReason.Header.Set("Authorization", "Bearer "+execToken)
	reqNoReason.Header.Set("Content-Type", "application/json")
	wNoReason := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNoReason, reqNoReason)

	if wNoReason.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request without changeReason, got %d", wNoReason.Code)
	}

	// Test 7: Production Head views audit history -> 200 OK
	reqHist, _ := http.NewRequest(http.MethodGet, "/api/v1/rate-cards/"+firstRateCardID+"/history", nil)
	reqHist.Header.Set("Authorization", "Bearer "+prodToken)
	wHist := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wHist, reqHist)

	if wHist.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for rate card history, got %d. Body: %s", wHist.Code, wHist.Body.String())
	}
	var histResp struct {
		Data []struct {
			NewSewingRate float64 `json:"newSewingRate"`
			Reason        string  `json:"reason"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wHist.Body.Bytes(), &histResp)
	if len(histResp.Data) == 0 {
		t.Fatalf("expected at least 1 history record after update")
	}

	// Test 8: Unauthenticated access -> 401 Unauthorized
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/rate-cards", nil)
	wNoAuth := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wNoAuth, reqNoAuth)

	if wNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated GET /rate-cards, got %d", wNoAuth.Code)
	}
}

func TestApp_ProductionLinesAndAllocationsLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// Clean up previous test run allocations for idempotent test execution
	_, _ = application.db.Exec("DELETE FROM line_transfers WHERE operator_id = (SELECT id FROM operators WHERE operator_code = 'JH-001')")
	_, _ = application.db.Exec("DELETE FROM line_allocations WHERE operator_id = (SELECT id FROM operators WHERE operator_code = 'JH-001')")

	// 1. Executive Login
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)
	if wExecLogin.Code != http.StatusOK {
		t.Fatalf("failed to login executive: %d", wExecLogin.Code)
	}
	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Client Login
	loginClientBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqClientLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginClientBody))
	reqClientLogin.Header.Set("Content-Type", "application/json")
	wClientLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLogin, reqClientLogin)
	if wClientLogin.Code != http.StatusOK {
		t.Fatalf("failed to login client: %d", wClientLogin.Code)
	}
	var clientLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientLogin.Body.Bytes(), &clientLoginResp)
	clientToken := clientLoginResp.Data.Token

	// Step 1: GET /api/v1/lines as Executive -> 200 OK
	reqLines, _ := http.NewRequest(http.MethodGet, "/api/v1/lines", nil)
	reqLines.Header.Set("Authorization", "Bearer "+execToken)
	wLines := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLines, reqLines)

	if wLines.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /lines, got %d: %s", wLines.Code, wLines.Body.String())
	}

	var linesResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID          string `json:"id"`
			Code        string `json:"code"`
			Name        string `json:"name"`
			MaxCapacity int    `json:"maxCapacity"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLines.Body.Bytes(), &linesResp)
	if len(linesResp.Data) < 2 {
		t.Fatalf("expected at least 2 seeded production lines, got %d", len(linesResp.Data))
	}
	line1ID := linesResp.Data[0].ID
	line2ID := linesResp.Data[1].ID

	// Step 2: GET /api/v1/lines as Client -> 403 Forbidden
	reqClientLines, _ := http.NewRequest(http.MethodGet, "/api/v1/lines", nil)
	reqClientLines.Header.Set("Authorization", "Bearer "+clientToken)
	wClientLines := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLines, reqClientLines)

	if wClientLines.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client accessing /lines, got %d", wClientLines.Code)
	}

	// Step 3: GET /api/v1/lines/:id -> 200 OK
	reqLineDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/lines/"+line1ID, nil)
	reqLineDetail.Header.Set("Authorization", "Bearer "+execToken)
	wLineDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLineDetail, reqLineDetail)

	if wLineDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /lines/:id, got %d: %s", wLineDetail.Code, wLineDetail.Body.String())
	}

	// Step 4: PUT /api/v1/lines/:id -> 200 OK (update name)
	putLineBody, _ := json.Marshal(map[string]interface{}{
		"name": "Line Jahit Parka Utama (Updated)",
	})
	reqPutLine, _ := http.NewRequest(http.MethodPut, "/api/v1/lines/"+line1ID, bytes.NewBuffer(putLineBody))
	reqPutLine.Header.Set("Authorization", "Bearer "+execToken)
	reqPutLine.Header.Set("Content-Type", "application/json")
	wPutLine := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wPutLine, reqPutLine)

	if wPutLine.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on PUT /lines/:id, got %d: %s", wPutLine.Code, wPutLine.Body.String())
	}

	// Fetch Order ID from DB
	var orderID string
	err = application.db.QueryRow("SELECT id FROM orders WHERE order_number = 'PO-2026-10-001'").Scan(&orderID)
	if err != nil {
		t.Fatalf("failed to query seeded order PO-2026-10-001: %v", err)
	}

	// Fetch Operator Siti Rahmawati ID
	var operatorID string
	err = application.db.QueryRow("SELECT id FROM operators WHERE operator_code = 'JH-001'").Scan(&operatorID)
	if err != nil {
		t.Fatalf("failed to query seeded operator JH-001: %v", err)
	}

	// Step 5: POST /api/v1/line-allocations -> 201 Created (Allocate JH-001 to Line 1)
	allocBody, _ := json.Marshal(map[string]interface{}{
		"lineId":     line1ID,
		"orderId":    orderID,
		"operatorId": operatorID,
	})
	reqAlloc, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations", bytes.NewBuffer(allocBody))
	reqAlloc.Header.Set("Authorization", "Bearer "+execToken)
	reqAlloc.Header.Set("Content-Type", "application/json")
	wAlloc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAlloc, reqAlloc)

	if wAlloc.Code != http.StatusCreated && wAlloc.Code != http.StatusConflict {
		t.Fatalf("expected 201 Created or 409 on POST /line-allocations, got %d: %s", wAlloc.Code, wAlloc.Body.String())
	}

	var allocID string
	if wAlloc.Code == http.StatusCreated {
		var allocResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wAlloc.Body.Bytes(), &allocResp)
		allocID = allocResp.Data.ID
	} else {
		// already allocated from previous run, get active allocation id
		_ = application.db.QueryRow("SELECT id FROM line_allocations WHERE operator_id = $1 AND status = 'ACTIVE'", operatorID).Scan(&allocID)
	}

	// Step 6: POST /api/v1/line-allocations/:id/transfer -> 200 OK (BPMN 04 Transfer to Line 2)
	transferBody, _ := json.Marshal(map[string]interface{}{
		"destinationLineId": line2ID,
		"transferReason":    "Mesin jahit di Line 1 mendadak macet / balancing line 2",
	})
	reqTransfer, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations/"+allocID+"/transfer", bytes.NewBuffer(transferBody))
	reqTransfer.Header.Set("Authorization", "Bearer "+execToken)
	reqTransfer.Header.Set("Content-Type", "application/json")
	wTransfer := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wTransfer, reqTransfer)

	if wTransfer.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on POST /line-allocations/:id/transfer, got %d: %s", wTransfer.Code, wTransfer.Body.String())
	}

	var transferResp struct {
		Success bool `json:"success"`
		Data    struct {
			OldAllocationID     string `json:"oldAllocationId"`
			NewAllocationID     string `json:"newAllocationId"`
			DestinationLineCode string `json:"destinationLineCode"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wTransfer.Body.Bytes(), &transferResp)
	if !transferResp.Success || transferResp.Data.NewAllocationID == "" {
		t.Fatalf("transfer response invalid: %+v", transferResp)
	}

	// Step 7: GET /api/v1/line-allocations/:id/history -> 200 OK
	reqHist, _ := http.NewRequest(http.MethodGet, "/api/v1/line-allocations/"+transferResp.Data.NewAllocationID+"/history", nil)
	reqHist.Header.Set("Authorization", "Bearer "+execToken)
	wHist := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wHist, reqHist)

	if wHist.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /line-allocations/:id/history, got %d: %s", wHist.Code, wHist.Body.String())
	}
	var histResp struct {
		Success bool `json:"success"`
		Data    []struct {
			Reason string `json:"reason"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wHist.Body.Bytes(), &histResp)
	if len(histResp.Data) == 0 {
		t.Fatalf("expected at least 1 transfer history record after transfer")
	}
}

func TestApp_ProductionOutputsLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Executive Login
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)
	if wExecLogin.Code != http.StatusOK {
		t.Fatalf("failed to login executive: %d", wExecLogin.Code)
	}
	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Client Login (to test 403 Forbidden)
	loginClientBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqClientLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginClientBody))
	reqClientLogin.Header.Set("Content-Type", "application/json")
	wClientLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLogin, reqClientLogin)
	if wClientLogin.Code != http.StatusOK {
		t.Fatalf("failed to login client: %d", wClientLogin.Code)
	}
	var clientLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientLogin.Body.Bytes(), &clientLoginResp)
	clientToken := clientLoginResp.Data.Token

	// 3. Query required entity IDs from DB
	var lineID, orderID, operator1ID, operator2ID, componentID string
	err = application.db.QueryRow("SELECT id FROM lines WHERE code = 'LINE-01'").Scan(&lineID)
	if err != nil {
		t.Fatalf("failed to find LINE-01: %v", err)
	}
	err = application.db.QueryRow("SELECT id FROM orders WHERE order_number = 'PO-2026-10-001'").Scan(&orderID)
	if err != nil {
		t.Fatalf("failed to find PO-2026-10-001: %v", err)
	}
	err = application.db.QueryRow("SELECT id FROM operators WHERE operator_code = 'JH-001'").Scan(&operator1ID)
	if err != nil {
		t.Fatalf("failed to find JH-001: %v", err)
	}
	err = application.db.QueryRow("SELECT id FROM operators WHERE operator_code = 'JH-002'").Scan(&operator2ID)
	if err != nil {
		t.Fatalf("failed to find JH-002: %v", err)
	}
	err = application.db.QueryRow("SELECT id FROM components WHERE code = 'CMP-SNO-PK-02'").Scan(&componentID)
	if err != nil {
		t.Fatalf("failed to find CMP-SNO-PK-02: %v", err)
	}

	// 4. Test Step 1: POST /api/v1/production-outputs (Single Output & Rate Snapshot Locking)
	singleBody, _ := json.Marshal(map[string]interface{}{
		"lineId":      lineID,
		"orderId":     orderID,
		"operatorId":  operator1ID,
		"componentId": componentID,
		"quantity":    25,
		"timeSlot":    "08:00-08:30",
		"workDate":    "2026-10-14",
		"notes":       "Slot pertama lancar, mesin stabil",
	})
	reqSingle, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs", bytes.NewBuffer(singleBody))
	reqSingle.Header.Set("Authorization", "Bearer "+execToken)
	reqSingle.Header.Set("Content-Type", "application/json")
	wSingle := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSingle, reqSingle)

	if wSingle.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST /production-outputs, got %d: %s", wSingle.Code, wSingle.Body.String())
	}

	var singleResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                 string  `json:"id"`
			LineCode           string  `json:"lineCode"`
			OperatorName       string  `json:"operatorName"`
			ComponentName      string  `json:"componentName"`
			Quantity           int     `json:"quantity"`
			TimeSlot           string  `json:"timeSlot"`
			WorkDate           string  `json:"workDate"`
			SnapshotSewingRate float64 `json:"snapshotSewingRate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wSingle.Body.Bytes(), &singleResp); err != nil {
		t.Fatalf("failed to parse single output response: %v", err)
	}

	if !singleResp.Success || singleResp.Data.ID == "" {
		t.Fatalf("invalid single output response: %+v", singleResp)
	}
	if singleResp.Data.SnapshotSewingRate != 4500.00 {
		t.Fatalf("expected snapshot sewing rate 4500.00 locked from rate card, got %f", singleResp.Data.SnapshotSewingRate)
	}
	if singleResp.Data.Quantity != 25 {
		t.Fatalf("expected quantity 25, got %d", singleResp.Data.Quantity)
	}
	createdOutputID := singleResp.Data.ID

	// 5. Test Step 2: GET /api/v1/production-outputs/:id (Detail output)
	reqDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/production-outputs/"+createdOutputID, nil)
	reqDetail.Header.Set("Authorization", "Bearer "+execToken)
	wDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wDetail, reqDetail)

	if wDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /production-outputs/:id, got %d: %s", wDetail.Code, wDetail.Body.String())
	}

	// 6. Test Step 3: POST /api/v1/production-outputs/batch (Batch Input across operators)
	batchBody, _ := json.Marshal(map[string]interface{}{
		"lineId":   lineID,
		"orderId":  orderID,
		"timeSlot": "08:30-09:00",
		"workDate": "2026-10-14",
		"items": []map[string]interface{}{
			{
				"operatorId":  operator1ID,
				"componentId": componentID,
				"quantity":    28,
				"notes":       "Target tercapai",
			},
			{
				"operatorId":  operator2ID,
				"componentId": componentID,
				"quantity":    26,
				"notes":       "Target tercapai",
			},
		},
	})
	reqBatch, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs/batch", bytes.NewBuffer(batchBody))
	reqBatch.Header.Set("Authorization", "Bearer "+execToken)
	reqBatch.Header.Set("Content-Type", "application/json")
	wBatch := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wBatch, reqBatch)

	if wBatch.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST /production-outputs/batch, got %d: %s", wBatch.Code, wBatch.Body.String())
	}

	var batchResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID                 string  `json:"id"`
			OperatorName       string  `json:"operatorName"`
			Quantity           int     `json:"quantity"`
			SnapshotSewingRate float64 `json:"snapshotSewingRate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wBatch.Body.Bytes(), &batchResp); err != nil {
		t.Fatalf("failed to parse batch output response: %v", err)
	}
	if len(batchResp.Data) != 2 {
		t.Fatalf("expected 2 batch output items, got %d", len(batchResp.Data))
	}
	for _, item := range batchResp.Data {
		if item.SnapshotSewingRate != 4500.00 {
			t.Fatalf("expected batch item snapshotSewingRate 4500.00, got %f", item.SnapshotSewingRate)
		}
	}

	// 7. Test Step 4: GET /api/v1/production-outputs?line_id=...&date=2026-10-14 (Filtered List)
	reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/production-outputs?line_id="+lineID+"&date=2026-10-14", nil)
	reqList.Header.Set("Authorization", "Bearer "+execToken)
	wList := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /production-outputs, got %d: %s", wList.Code, wList.Body.String())
	}
	var listResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID       string `json:"id"`
			LineCode string `json:"lineCode"`
			Quantity int    `json:"quantity"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list output response: %v", err)
	}
	if len(listResp.Data) < 3 {
		t.Fatalf("expected at least 3 output records, got %d", len(listResp.Data))
	}

	// 8. Test Step 5: 403 Forbidden for client role
	reqForbidden, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs", bytes.NewBuffer(singleBody))
	reqForbidden.Header.Set("Authorization", "Bearer "+clientToken)
	reqForbidden.Header.Set("Content-Type", "application/json")
	wForbidden := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wForbidden, reqForbidden)

	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client accessing production-outputs, got %d", wForbidden.Code)
	}
}

func TestApp_AssemblyAndProductionDashboardLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Executive Login
	loginExecBody, _ := json.Marshal(map[string]string{
		"username": "executive@lukla.local",
		"password": "password",
	})
	reqExecLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginExecBody))
	reqExecLogin.Header.Set("Content-Type", "application/json")
	wExecLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wExecLogin, reqExecLogin)
	if wExecLogin.Code != http.StatusOK {
		t.Fatalf("failed to login executive: %d", wExecLogin.Code)
	}
	var execLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wExecLogin.Body.Bytes(), &execLoginResp)
	execToken := execLoginResp.Data.Token

	// 2. Client Login
	loginClientBody, _ := json.Marshal(map[string]string{
		"username": "client@lukla.local",
		"password": "password",
	})
	reqClientLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginClientBody))
	reqClientLogin.Header.Set("Content-Type", "application/json")
	wClientLogin := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientLogin, reqClientLogin)
	if wClientLogin.Code != http.StatusOK {
		t.Fatalf("failed to login client: %d", wClientLogin.Code)
	}
	var clientLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientLogin.Body.Bytes(), &clientLoginResp)
	clientToken := clientLoginResp.Data.Token

	// 3. Query required entity IDs
	var orderID, articleID, lineID string
	err = application.db.QueryRow("SELECT id, article_id FROM orders WHERE order_number = 'PO-2026-10-001'").Scan(&orderID, &articleID)
	if err != nil {
		t.Fatalf("failed to query PO-2026-10-001: %v", err)
	}
	err = application.db.QueryRow("SELECT id FROM lines WHERE code = 'LINE-01'").Scan(&lineID)
	if err != nil {
		t.Fatalf("failed to query LINE-01: %v", err)
	}

	// Ensure there are enough component outputs recorded for orderID
	var totalCompOutputs int
	_ = application.db.QueryRow("SELECT COALESCE(SUM(quantity), 0) FROM production_outputs WHERE order_id = $1", orderID).Scan(&totalCompOutputs)
	if totalCompOutputs < 20 {
		var opID, compID string
		_ = application.db.QueryRow("SELECT id FROM operators WHERE operator_code = 'JH-001'").Scan(&opID)
		_ = application.db.QueryRow("SELECT id FROM components WHERE article_id = $1 LIMIT 1", articleID).Scan(&compID)
		_, _ = application.db.Exec(`
			INSERT INTO production_outputs (line_id, order_id, operator_id, component_id, quantity, time_slot, work_date, snapshot_sewing_rate)
			VALUES ($1, $2, $3, $4, 50, '08:00-08:30', CURRENT_DATE, 4500.00)
		`, lineID, orderID, opID, compID)
	}

	// 4. Test Step 1: POST /api/v1/assemblies -> 201 Created (Status MENUNGGU_QC)
	assemblyBody, _ := json.Marshal(map[string]interface{}{
		"orderId":           orderID,
		"articleId":         articleID,
		"quantityAssembled": 15,
		"assembledDate":     "2026-10-14",
		"notes":             "Batch 1 assembly selesai jahit siap QC",
	})
	reqAssembly, _ := http.NewRequest(http.MethodPost, "/api/v1/assemblies", bytes.NewBuffer(assemblyBody))
	reqAssembly.Header.Set("Authorization", "Bearer "+execToken)
	reqAssembly.Header.Set("Content-Type", "application/json")
	wAssembly := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAssembly, reqAssembly)

	if wAssembly.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on POST /assemblies, got %d: %s", wAssembly.Code, wAssembly.Body.String())
	}

	var assemblyResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                string `json:"id"`
			OrderCode         string `json:"orderCode"`
			ArticleTitle      string `json:"articleTitle"`
			QuantityAssembled int    `json:"quantityAssembled"`
			Status            string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wAssembly.Body.Bytes(), &assemblyResp); err != nil {
		t.Fatalf("failed to parse assembly response: %v", err)
	}
	if !assemblyResp.Success || assemblyResp.Data.Status != "MENUNGGU_QC" {
		t.Fatalf("expected status MENUNGGU_QC, got %+v", assemblyResp)
	}

	// 5. Test Step 2: GET /api/v1/assemblies -> 200 OK
	reqListAsm, _ := http.NewRequest(http.MethodGet, "/api/v1/assemblies?order_id="+orderID, nil)
	reqListAsm.Header.Set("Authorization", "Bearer "+execToken)
	wListAsm := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wListAsm, reqListAsm)

	if wListAsm.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /assemblies, got %d: %s", wListAsm.Code, wListAsm.Body.String())
	}

	// 6. Test Step 3: GET /api/v1/assemblies/queue-qc -> 200 OK
	reqQueue, _ := http.NewRequest(http.MethodGet, "/api/v1/assemblies/queue-qc", nil)
	reqQueue.Header.Set("Authorization", "Bearer "+execToken)
	wQueue := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wQueue, reqQueue)

	if wQueue.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /assemblies/queue-qc, got %d: %s", wQueue.Code, wQueue.Body.String())
	}
	var queueResp struct {
		Success bool `json:"success"`
		Data    []struct {
			AssemblyID      string `json:"assemblyId"`
			OrderCode       string `json:"orderCode"`
			PendingQuantity int    `json:"pendingQuantity"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wQueue.Body.Bytes(), &queueResp); err != nil {
		t.Fatalf("failed to parse queue response: %v", err)
	}
	if len(queueResp.Data) == 0 {
		t.Fatalf("expected at least 1 item in QC queue")
	}

	// 7. Test Step 4: GET /api/v1/production-dashboard/realtime -> 200 OK
	reqRealtime, _ := http.NewRequest(http.MethodGet, "/api/v1/production-dashboard/realtime", nil)
	reqRealtime.Header.Set("Authorization", "Bearer "+execToken)
	wRealtime := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRealtime, reqRealtime)

	if wRealtime.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /production-dashboard/realtime, got %d: %s", wRealtime.Code, wRealtime.Body.String())
	}
	var rtResp struct {
		Success bool `json:"success"`
		Data    struct {
			FactoryMachineCapacity int `json:"factoryMachineCapacity"`
			TotalTodayOutputPcs    int `json:"totalTodayOutputPcs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wRealtime.Body.Bytes(), &rtResp); err != nil {
		t.Fatalf("failed to parse realtime dashboard response: %v", err)
	}
	if rtResp.Data.FactoryMachineCapacity != 70 {
		t.Fatalf("expected factoryMachineCapacity 70, got %d", rtResp.Data.FactoryMachineCapacity)
	}

	// 8. Test Step 5: GET /api/v1/production-dashboard/lines/:line_id -> 200 OK
	reqLineProg, _ := http.NewRequest(http.MethodGet, "/api/v1/production-dashboard/lines/"+lineID, nil)
	reqLineProg.Header.Set("Authorization", "Bearer "+execToken)
	wLineProg := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLineProg, reqLineProg)

	if wLineProg.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /production-dashboard/lines/:line_id, got %d: %s", wLineProg.Code, wLineProg.Body.String())
	}

	// 9. Test Step 6: GET /api/v1/production-dashboard/orders/:order_id -> 200 OK
	reqOrderProg, _ := http.NewRequest(http.MethodGet, "/api/v1/production-dashboard/orders/"+orderID, nil)
	reqOrderProg.Header.Set("Authorization", "Bearer "+execToken)
	wOrderProg := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wOrderProg, reqOrderProg)

	if wOrderProg.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on GET /production-dashboard/orders/:order_id, got %d: %s", wOrderProg.Code, wOrderProg.Body.String())
	}
	var ordProgResp struct {
		Success bool `json:"success"`
		Data    struct {
			OrderCode          string `json:"orderCode"`
			TotalTargetPcs     int    `json:"totalTargetPcs"`
			TotalAssembledPcs  int    `json:"totalAssembledPcs"`
			ComponentsProgress []struct {
				ComponentName string `json:"componentName"`
				CompletedPcs  int    `json:"completedPcs"`
			} `json:"componentsProgress"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wOrderProg.Body.Bytes(), &ordProgResp); err != nil {
		t.Fatalf("failed to parse order progress response: %v", err)
	}
	if ordProgResp.Data.TotalTargetPcs != 1000 || len(ordProgResp.Data.ComponentsProgress) == 0 {
		t.Fatalf("unexpected order progress data: %+v", ordProgResp)
	}

	// 10. Test Step 7: 403 Forbidden for client role
	reqClientAsm, _ := http.NewRequest(http.MethodPost, "/api/v1/assemblies", bytes.NewBuffer(assemblyBody))
	reqClientAsm.Header.Set("Authorization", "Bearer "+clientToken)
	reqClientAsm.Header.Set("Content-Type", "application/json")
	wClientAsm := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientAsm, reqClientAsm)

	if wClientAsm.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client creating assembly, got %d", wClientAsm.Code)
	}
}

func getAuthToken(t *testing.T, app *App, username, password string) string {
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("failed to login %s: %d, body: %s", username, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.Token
}

func TestApp_PayrollSkenario2LiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	// 1. Login as executive, production, and finance
	execToken := getAuthToken(t, application, "executive@lukla.local", "password")
	finToken := getAuthToken(t, application, "finance@lukla.local", "password")
	prodToken := getAuthToken(t, application, "production@lukla.local", "password")

	// 2. Fetch active line, order, and component
	var lineID, orderID, componentID string
	err = application.db.QueryRow(`SELECT id FROM lines WHERE status = 'ACTIVE' LIMIT 1`).Scan(&lineID)
	if err != nil {
		t.Fatalf("no active line found: %v", err)
	}
	err = application.db.QueryRow(`SELECT id FROM orders WHERE status = 'IN_PRODUCTION' LIMIT 1`).Scan(&orderID)
	if err != nil {
		t.Fatalf("no active order found: %v", err)
	}
	err = application.db.QueryRow(`SELECT id FROM components LIMIT 1`).Scan(&componentID)
	if err != nil {
		t.Fatalf("no component found: %v", err)
	}

	// 3. Skenario Uji 2: Create a new operator joining mid-week (Wednesday 2026-10-14)
	midweekCode := fmt.Sprintf("OP-MID-%d", time.Now().UnixNano())
	var newOperatorID string
	err = application.db.QueryRow(`
		INSERT INTO operators (operator_code, name, join_date, status, default_line)
		VALUES ($1, 'Dewi Penjahit Baru Midweek', '2026-10-14', 'ACTIVE', 'LINE-01')
		RETURNING id
	`, midweekCode).Scan(&newOperatorID)
	if err != nil {
		t.Fatalf("failed to insert midweek operator: %v", err)
	}

	periodCode := fmt.Sprintf("PAY-TEST-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = application.db.Exec(`DELETE FROM payroll_periods WHERE period_code = $1`, periodCode)
		_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id = $1`, newOperatorID)
		_, _ = application.db.Exec(`DELETE FROM operators WHERE id = $1`, newOperatorID)
	})

	// 4. Insert production outputs for this operator:
	// Output Wednesday (2026-10-14): 80 pcs @ 3000 = 240,000
	// Output Thursday  (2026-10-15): 120 pcs @ 3000 = 360,000
	// Output Monday prior to join (2026-10-12): 50 pcs @ 3000 = 150,000 (Should be EXCLUDED by business rule!)
	_, err = application.db.Exec(`
		INSERT INTO production_outputs (line_id, order_id, operator_id, component_id, quantity, time_slot, work_date, snapshot_sewing_rate)
		VALUES 
			($1, $2, $3, $4, 80, '08:00-08:30', '2026-10-14', 3000.00),
			($1, $2, $3, $4, 120, '08:30-09:00', '2026-10-15', 3000.00),
			($1, $2, $3, $4, 50, '08:00-08:30', '2026-10-12', 3000.00)
	`, lineID, orderID, newOperatorID, componentID)
	if err != nil {
		t.Fatalf("failed to insert production outputs: %v", err)
	}

	// 5. Create Payroll Period via POST /api/v1/payroll/periods by Finance
	createReqBody := map[string]interface{}{
		"periodCode":              periodCode,
		"startDate":               "2026-10-12",
		"endDate":                 "2026-10-18",
		"mealAllowancePercentage": 12.50,
		"notes":                   "Uji Skenario 2 Penjahit Baru di Tengah Minggu",
	}
	createJSON, _ := json.Marshal(createReqBody)

	reqCreate, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBuffer(createJSON))
	reqCreate.Header.Set("Authorization", "Bearer "+finToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for payroll period, got %d. Body: %s", wCreate.Code, wCreate.Body.String())
	}

	var createdPeriodResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                      string  `json:"id"`
			PeriodCode              string  `json:"periodCode"`
			Status                  string  `json:"status"`
			MealAllowancePercentage float64 `json:"mealAllowancePercentage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wCreate.Body.Bytes(), &createdPeriodResp); err != nil {
		t.Fatalf("failed to parse create period response: %v", err)
	}
	periodID := createdPeriodResp.Data.ID
	if periodID == "" || createdPeriodResp.Data.Status != "DRAFT" {
		t.Fatalf("unexpected period data: %+v", createdPeriodResp)
	}

	// 6. Test RBAC: Production Head cannot calculate payroll (403 Forbidden)
	reqProdCalc, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/calculate", bytes.NewBufferString(`{}`))
	reqProdCalc.Header.Set("Authorization", "Bearer "+prodToken)
	reqProdCalc.Header.Set("Content-Type", "application/json")
	wProdCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wProdCalc, reqProdCalc)

	if wProdCalc.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for production head calculating payroll, got %d", wProdCalc.Code)
	}

	// 7. Calculate Payroll via POST /api/v1/payroll/periods/{id}/calculate by Executive
	reqCalc, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/calculate", bytes.NewBufferString(`{}`))
	reqCalc.Header.Set("Authorization", "Bearer "+execToken)
	reqCalc.Header.Set("Content-Type", "application/json")
	wCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalc, reqCalc)

	if wCalc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for payroll calculation, got %d. Body: %s", wCalc.Code, wCalc.Body.String())
	}

	// 8. List slips via GET /api/v1/payroll/periods/{id}/slips
	reqSlips, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/"+periodID+"/slips?operatorId="+newOperatorID, nil)
	reqSlips.Header.Set("Authorization", "Bearer "+finToken)
	wSlips := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSlips, reqSlips)

	if wSlips.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list slips, got %d. Body: %s", wSlips.Code, wSlips.Body.String())
	}

	var slipsResp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID                  string  `json:"id"`
			OperatorID          string  `json:"operatorId"`
			OperatorCode        string  `json:"operatorCode"`
			TotalPieces         int     `json:"totalPieces"`
			GrossPieceRateTotal float64 `json:"grossPieceRateTotal"`
			MealAllowanceAmount float64 `json:"mealAllowanceAmount"`
			NetPayableTotal     float64 `json:"netPayableTotal"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wSlips.Body.Bytes(), &slipsResp); err != nil {
		t.Fatalf("failed to parse slips response: %v", err)
	}

	if len(slipsResp.Data) == 0 {
		t.Fatalf("expected at least 1 slip for midweek operator, got 0")
	}

	slip := slipsResp.Data[0]
	// Skenario 2 Verification:
	// Quantity: Wednesday (80) + Thursday (120) = 200 pcs (Monday's 50 pcs must be EXCLUDED because joinDate was Wednesday!)
	if slip.TotalPieces != 200 {
		t.Fatalf("Skenario 2 FAILED: expected TotalPieces = 200 (excluding Monday output prior to join date), got %d", slip.TotalPieces)
	}
	// Gross Piece Rate: 200 pcs * 3000 = 600,000.00
	if slip.GrossPieceRateTotal != 600000.00 {
		t.Fatalf("Skenario 2 FAILED: expected GrossPieceRateTotal = 600000.00, got %f", slip.GrossPieceRateTotal)
	}
	// Meal Allowance: 12.5% of 600,000 = 75,000.00
	if slip.MealAllowanceAmount != 75000.00 {
		t.Fatalf("Skenario 2 FAILED: expected MealAllowanceAmount = 75000.00, got %f", slip.MealAllowanceAmount)
	}
	// Total Net Payable: 600,000 + 75,000 = 675,000.00
	if slip.NetPayableTotal != 675000.00 {
		t.Fatalf("Skenario 2 FAILED: expected NetPayableTotal = 675000.00, got %f", slip.NetPayableTotal)
	}

	// 9. Get detailed slip via GET /api/v1/payroll/slips/{id}
	reqDetail, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/slips/"+slip.ID, nil)
	reqDetail.Header.Set("Authorization", "Bearer "+finToken)
	wDetail := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wDetail, reqDetail)

	if wDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for slip detail, got %d. Body: %s", wDetail.Code, wDetail.Body.String())
	}

	var detailResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                      string  `json:"id"`
			MealAllowancePercentage float64 `json:"mealAllowancePercentage"`
			Items                   []struct {
				WorkDate           string  `json:"workDate"`
				Quantity           int     `json:"quantity"`
				SnapshotSewingRate float64 `json:"snapshotSewingRate"`
				Subtotal           float64 `json:"subtotal"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wDetail.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("failed to parse slip detail: %v", err)
	}
	if len(detailResp.Data.Items) != 2 {
		t.Fatalf("expected 2 slip items (Wednesday & Thursday), got %d", len(detailResp.Data.Items))
	}
	if detailResp.Data.MealAllowancePercentage != 12.50 {
		t.Fatalf("expected MealAllowancePercentage 12.50, got %f", detailResp.Data.MealAllowancePercentage)
	}
}

func TestApp_PayrollLockingAndAdjustmentLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	execToken := getAuthToken(t, application, "executive@lukla.local", "password")
	finToken := getAuthToken(t, application, "finance@lukla.local", "password")

	var lineID, orderID, componentID string
	err = application.db.QueryRow(`SELECT id FROM lines WHERE status = 'ACTIVE' LIMIT 1`).Scan(&lineID)
	if err != nil {
		t.Fatalf("no active line found: %v", err)
	}
	err = application.db.QueryRow(`SELECT id FROM orders WHERE status = 'IN_PRODUCTION' LIMIT 1`).Scan(&orderID)
	if err != nil {
		t.Fatalf("no active order found: %v", err)
	}
	err = application.db.QueryRow(`SELECT id FROM components LIMIT 1`).Scan(&componentID)
	if err != nil {
		t.Fatalf("no component found: %v", err)
	}

	opCode := fmt.Sprintf("OP-LOCK-%d", time.Now().UnixNano())
	var operatorID string
	err = application.db.QueryRow(`
		INSERT INTO operators (operator_code, name, join_date, status, default_line)
		VALUES ($1, 'Rudi Penjahit Uji Locking', '2026-10-01', 'ACTIVE', 'LINE-01')
		RETURNING id
	`, opCode).Scan(&operatorID)
	if err != nil {
		t.Fatalf("failed to insert operator: %v", err)
	}

	periodCode := fmt.Sprintf("PAY-LOCK-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = application.db.Exec(`DELETE FROM payroll_periods WHERE period_code = $1`, periodCode)
		_, _ = application.db.Exec(`DELETE FROM production_outputs WHERE operator_id = $1`, operatorID)
		_, _ = application.db.Exec(`DELETE FROM operators WHERE id = $1`, operatorID)
	})

	// Output: 100 pcs @ 4000 = 400,000
	_, err = application.db.Exec(`
		INSERT INTO production_outputs (line_id, order_id, operator_id, component_id, quantity, time_slot, work_date, snapshot_sewing_rate)
		VALUES ($1, $2, $3, $4, 100, '09:00-09:30', '2026-10-20', 4000.00)
	`, lineID, orderID, operatorID, componentID)
	if err != nil {
		t.Fatalf("failed to insert production outputs: %v", err)
	}

	// 1. Create period: meal allowance 10.0%
	createReqBody := map[string]interface{}{
		"periodCode":              periodCode,
		"startDate":               "2026-10-19",
		"endDate":                 "2026-10-25",
		"mealAllowancePercentage": 10.00,
		"notes":                   "Uji Locking & Adjustment Tahap 2.2",
	}
	createJSON, _ := json.Marshal(createReqBody)

	reqCreate, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBuffer(createJSON))
	reqCreate.Header.Set("Authorization", "Bearer "+finToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for period, got %d. Body: %s", wCreate.Code, wCreate.Body.String())
	}

	var createdPeriodResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createdPeriodResp)
	periodID := createdPeriodResp.Data.ID

	// 2. Cannot lock DRAFT period before calculation
	reqLockDraft, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/lock", nil)
	reqLockDraft.Header.Set("Authorization", "Bearer "+execToken)
	wLockDraft := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLockDraft, reqLockDraft)

	if wLockDraft.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when locking DRAFT period, got %d. Body: %s", wLockDraft.Code, wLockDraft.Body.String())
	}

	// 3. Calculate payroll
	reqCalc, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/calculate", bytes.NewBufferString(`{}`))
	reqCalc.Header.Set("Authorization", "Bearer "+finToken)
	reqCalc.Header.Set("Content-Type", "application/json")
	wCalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wCalc, reqCalc)

	if wCalc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for calculate, got %d. Body: %s", wCalc.Code, wCalc.Body.String())
	}

	// 4. Create Adjustment 1: BONUS Rp 60.000
	adj1Body := map[string]interface{}{
		"operatorId":     operatorID,
		"adjustmentType": "BONUS",
		"amount":         60000.0,
		"reason":         "Bonus pencapaian target mingguan",
	}
	adj1JSON, _ := json.Marshal(adj1Body)
	reqAdj1, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/adjustments", bytes.NewBuffer(adj1JSON))
	reqAdj1.Header.Set("Authorization", "Bearer "+finToken)
	reqAdj1.Header.Set("Content-Type", "application/json")
	wAdj1 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAdj1, reqAdj1)

	if wAdj1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for adjustment 1, got %d. Body: %s", wAdj1.Code, wAdj1.Body.String())
	}

	// 5. Create Adjustment 2: DEDUCTION Rp 15.000
	adj2Body := map[string]interface{}{
		"operatorId":     operatorID,
		"adjustmentType": "DEDUCTION",
		"amount":         15000.0,
		"reason":         "Potongan kasbon harian",
	}
	adj2JSON, _ := json.Marshal(adj2Body)
	reqAdj2, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/adjustments", bytes.NewBuffer(adj2JSON))
	reqAdj2.Header.Set("Authorization", "Bearer "+finToken)
	reqAdj2.Header.Set("Content-Type", "application/json")
	wAdj2 := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wAdj2, reqAdj2)

	if wAdj2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for adjustment 2, got %d. Body: %s", wAdj2.Code, wAdj2.Body.String())
	}

	// 6. List Adjustments
	reqListAdj, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/"+periodID+"/adjustments", nil)
	reqListAdj.Header.Set("Authorization", "Bearer "+finToken)
	wListAdj := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wListAdj, reqListAdj)

	if wListAdj.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list adjustments, got %d", wListAdj.Code)
	}
	var listAdjResp struct {
		Data []struct {
			AdjustmentType string  `json:"adjustmentType"`
			Amount         float64 `json:"amount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wListAdj.Body.Bytes(), &listAdjResp)
	if len(listAdjResp.Data) != 2 {
		t.Fatalf("expected 2 adjustments, got %d", len(listAdjResp.Data))
	}

	// 7. Check Slip totals:
	// Gross: 400,000, Meal: 40,000, Adjustments: +60,000 -15,000 = 45,000
	// Net Payable: 400,000 + 40,000 + 45,000 = 485,000
	reqSlips, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/"+periodID+"/slips?operatorId="+operatorID, nil)
	reqSlips.Header.Set("Authorization", "Bearer "+finToken)
	wSlips := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSlips, reqSlips)

	if wSlips.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for slips, got %d", wSlips.Code)
	}
	var slipsResp struct {
		Data []struct {
			GrossPieceRateTotal float64 `json:"grossPieceRateTotal"`
			MealAllowanceAmount float64 `json:"mealAllowanceAmount"`
			AdjustmentsTotal    float64 `json:"adjustmentsTotal"`
			NetPayableTotal     float64 `json:"netPayableTotal"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSlips.Body.Bytes(), &slipsResp)
	if len(slipsResp.Data) == 0 {
		t.Fatalf("expected at least 1 slip")
	}
	slip := slipsResp.Data[0]
	if slip.GrossPieceRateTotal != 400000.00 {
		t.Errorf("expected gross 400000.00, got %f", slip.GrossPieceRateTotal)
	}
	if slip.MealAllowanceAmount != 40000.00 {
		t.Errorf("expected meal 40000.00, got %f", slip.MealAllowanceAmount)
	}
	if slip.AdjustmentsTotal != 45000.00 {
		t.Errorf("expected adjustments 45000.00, got %f", slip.AdjustmentsTotal)
	}
	if slip.NetPayableTotal != 485000.00 {
		t.Errorf("expected net payable 485000.00, got %f", slip.NetPayableTotal)
	}

	// 8. Get Summary Report
	reqSummary, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/"+periodID+"/summary", nil)
	reqSummary.Header.Set("Authorization", "Bearer "+execToken)
	wSummary := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wSummary, reqSummary)

	if wSummary.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for summary, got %d. Body: %s", wSummary.Code, wSummary.Body.String())
	}
	var summaryResp struct {
		Data struct {
			TotalGrossPieceRate float64 `json:"totalGrossPieceRate"`
			TotalMealAllowance  float64 `json:"totalMealAllowance"`
			TotalAdjustments    float64 `json:"totalAdjustments"`
			GrandTotalPayable   float64 `json:"grandTotalPayable"`
			TotalOperators      int     `json:"totalOperators"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSummary.Body.Bytes(), &summaryResp)
	if summaryResp.Data.GrandTotalPayable != 485000.00 {
		t.Errorf("expected grandTotalPayable 485000.00, got %f", summaryResp.Data.GrandTotalPayable)
	}

	// 9. Lock the Period via POST /api/v1/payroll/periods/{id}/lock by Executive
	lockBody := map[string]interface{}{
		"notes": "Periode diverifikasi dan disetujui untuk transfer perbankan",
	}
	lockJSON, _ := json.Marshal(lockBody)
	reqLock, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/lock", bytes.NewBuffer(lockJSON))
	reqLock.Header.Set("Authorization", "Bearer "+execToken)
	reqLock.Header.Set("Content-Type", "application/json")
	wLock := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLock, reqLock)

	if wLock.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for lock, got %d. Body: %s", wLock.Code, wLock.Body.String())
	}

	// Verify status in DB is LOCKED
	var dbStatus string
	err = application.db.QueryRow(`SELECT status FROM payroll_periods WHERE id = $1`, periodID).Scan(&dbStatus)
	if err != nil || dbStatus != "LOCKED" {
		t.Fatalf("expected period status LOCKED in DB, got %s, err: %v", dbStatus, err)
	}

	// Verify slip status in DB is LOCKED
	var slipStatus string
	err = application.db.QueryRow(`SELECT status FROM payroll_slips WHERE payroll_period_id = $1 AND operator_id = $2`, periodID, operatorID).Scan(&slipStatus)
	if err != nil || slipStatus != "LOCKED" {
		t.Fatalf("expected slip status LOCKED in DB, got %s, err: %v", slipStatus, err)
	}

	// 10. CRITICAL GOVERNANCE RULE: Recalculate on LOCKED period MUST RETURN 409 CONFLICT!
	reqRecalc, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/calculate", bytes.NewBufferString(`{}`))
	reqRecalc.Header.Set("Authorization", "Bearer "+finToken)
	reqRecalc.Header.Set("Content-Type", "application/json")
	wRecalc := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRecalc, reqRecalc)

	if wRecalc.Code != http.StatusConflict {
		t.Fatalf("GOVERNANCE BREACH: expected 409 Conflict when recalculating LOCKED period, got %d. Body: %s", wRecalc.Code, wRecalc.Body.String())
	}

	// 11. Lock again MUST RETURN 409 CONFLICT!
	reqRelock, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/"+periodID+"/lock", bytes.NewBuffer(lockJSON))
	reqRelock.Header.Set("Authorization", "Bearer "+execToken)
	reqRelock.Header.Set("Content-Type", "application/json")
	wRelock := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wRelock, reqRelock)

	if wRelock.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when locking already LOCKED period, got %d", wRelock.Code)
	}
}

func TestApp_InventoryInboundLiveDB(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set, skipping live DB test")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to initialize app: %v", err)
	}

	wrhToken := getAuthToken(t, application, "warehouse@lukla.local", "password")
	execToken := getAuthToken(t, application, "executive@lukla.local", "password")
	snohundToken := getAuthToken(t, application, "client@lukla.local", "password")

	// 1. Fetch first order and its client
	var snoClientID, snoOrderID string
	err = application.db.QueryRow(`SELECT id, client_id FROM orders LIMIT 1`).Scan(&snoOrderID, &snoClientID)
	if err != nil {
		t.Fatalf("failed to find seeded order: %v", err)
	}

	// 2. Fetch another client and its order (or create a temporary test client & order if only 1 client exists)
	var otherClientID, otherOrderID string
	err = application.db.QueryRow(`SELECT id, client_id FROM orders WHERE client_id != $1 LIMIT 1`, snoClientID).Scan(&otherOrderID, &otherClientID)
	if err != nil {
		// Insert temporary other client and order for testing mismatch rejection
		_ = application.db.QueryRow(`INSERT INTO clients (code, name, client_type) VALUES ('CLIENT-OTHER-TEST', 'Urban Culture Test', 'EXTERNAL') ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name RETURNING id`).Scan(&otherClientID)
		_ = application.db.QueryRow(`INSERT INTO orders (client_id, order_number, total_quantity) VALUES ($1, 'PO-TEST-OTHER-001', 50) ON CONFLICT (order_number) DO UPDATE SET total_quantity = EXCLUDED.total_quantity RETURNING id`, otherClientID).Scan(&otherOrderID)
	}

	// 3. Create test material via POST /api/v1/inventory/materials by Warehouse
	matCode := fmt.Sprintf("MAT-TEST-%d", time.Now().UnixNano())
	matBody := map[string]interface{}{
		"code":        matCode,
		"name":        "Kain Katun Bambu Premium 30s",
		"category":    "FABRIC",
		"defaultUnit": "KG",
		"description": "Material uji integrasi live DB inbound",
	}
	matJSON, _ := json.Marshal(matBody)

	reqMat, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/materials", bytes.NewBuffer(matJSON))
	reqMat.Header.Set("Authorization", "Bearer "+wrhToken)
	reqMat.Header.Set("Content-Type", "application/json")
	wMat := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMat, reqMat)

	if wMat.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for material, got %d. Body: %s", wMat.Code, wMat.Body.String())
	}

	var matResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wMat.Body.Bytes(), &matResp)
	matID := matResp.Data.ID

	rcvNum := fmt.Sprintf("RCV-TEST-%d", time.Now().UnixNano())
	lot1Num := fmt.Sprintf("LOT-A-%d", time.Now().UnixNano())
	lot2Num := fmt.Sprintf("LOT-B-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = application.db.Exec(`DELETE FROM inventory_receipts WHERE receipt_number = $1`, rcvNum)
		_, _ = application.db.Exec(`DELETE FROM materials WHERE id = $1`, matID)
	})

	// 4. Test FR-6.1 Mandatory Tagging: Attempt receipt with Mismatched Client and Order
	mismatchBody := map[string]interface{}{
		"receiptNumber": rcvNum + "-ERR",
		"clientId":      snoClientID,  // Snøhund
		"orderId":       otherOrderID, // Order of Urban Culture!
		"receivedDate":  "2026-10-15",
		"items": []map[string]interface{}{
			{
				"materialId":        matID,
				"lotNumber":         lot1Num + "-ERR",
				"quantity":          100.0,
				"unit":              "KG",
				"warehouseLocation": "RAK-TEST-1",
			},
		},
	}
	mismatchJSON, _ := json.Marshal(mismatchBody)
	reqMismatch, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBuffer(mismatchJSON))
	reqMismatch.Header.Set("Authorization", "Bearer "+wrhToken)
	reqMismatch.Header.Set("Content-Type", "application/json")
	wMismatch := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMismatch, reqMismatch)

	if wMismatch.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for mismatched client/order tagging, got %d. Body: %s", wMismatch.Code, wMismatch.Body.String())
	}

	// 5. Create Valid Inbound Receipt with 2 Lots
	validReceiptBody := map[string]interface{}{
		"receiptNumber":      rcvNum,
		"clientId":           snoClientID,
		"orderId":            snoOrderID,
		"receivedDate":       "2026-10-15",
		"supplierName":       "PT Grand Textile Bandung",
		"deliveryNoteNumber": "SJ-TEST-999",
		"notes":              "Bahan titipan uji FR-6.1",
		"items": []map[string]interface{}{
			{
				"materialId":        matID,
				"lotNumber":         lot1Num,
				"quantity":          120.50,
				"unit":              "KG",
				"rollCount":         4,
				"warehouseLocation": "RAK-A1-01",
			},
			{
				"materialId":        matID,
				"lotNumber":         lot2Num,
				"quantity":          79.50,
				"unit":              "KG",
				"rollCount":         2,
				"warehouseLocation": "RAK-A1-02",
			},
		},
	}
	validJSON, _ := json.Marshal(validReceiptBody)
	reqValid, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBuffer(validJSON))
	reqValid.Header.Set("Authorization", "Bearer "+wrhToken)
	reqValid.Header.Set("Content-Type", "application/json")
	wValid := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wValid, reqValid)

	if wValid.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for receipt, got %d. Body: %s", wValid.Code, wValid.Body.String())
	}

	var rcvResp struct {
		Data struct {
			ID    string `json:"id"`
			Items []struct {
				LotID string `json:"lotId"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wValid.Body.Bytes(), &rcvResp)
	receiptID := rcvResp.Data.ID
	if receiptID == "" || len(rcvResp.Data.Items) != 2 {
		t.Fatalf("unexpected receipt response structure: %+v", rcvResp)
	}
	lot1ID := rcvResp.Data.Items[0].LotID

	// 6. Verify Receipt Detail via GET /api/v1/inventory/receipts/{id}
	reqGetRcv, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/receipts/"+receiptID, nil)
	reqGetRcv.Header.Set("Authorization", "Bearer "+execToken)
	wGetRcv := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wGetRcv, reqGetRcv)

	if wGetRcv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get receipt, got %d", wGetRcv.Code)
	}

	// 7. Verify Lot Detail via GET /api/v1/inventory/lots/{id}
	reqLot, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/lots/"+lot1ID, nil)
	reqLot.Header.Set("Authorization", "Bearer "+wrhToken)
	wLot := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wLot, reqLot)

	if wLot.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get lot, got %d", wLot.Code)
	}
	var lotResp struct {
		Data struct {
			LotNumber       string  `json:"lotNumber"`
			InitialQuantity float64 `json:"initialQuantity"`
			CurrentQuantity float64 `json:"currentQuantity"`
			Status          string  `json:"status"`
			ClientID        string  `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wLot.Body.Bytes(), &lotResp)
	if lotResp.Data.InitialQuantity != 120.50 || lotResp.Data.CurrentQuantity != 120.50 {
		t.Errorf("expected 120.50 quantity, got initial=%f, current=%f", lotResp.Data.InitialQuantity, lotResp.Data.CurrentQuantity)
	}
	if lotResp.Data.ClientID != snoClientID {
		t.Errorf("expected lot to be tagged with Snøhund client ID %s, got %s", snoClientID, lotResp.Data.ClientID)
	}

	// 8. Verify Stock Balances via GET /api/v1/inventory/stocks
	reqStocks, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/stocks?clientId="+snoClientID+"&materialId="+matID, nil)
	reqStocks.Header.Set("Authorization", "Bearer "+wrhToken)
	wStocks := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wStocks, reqStocks)

	if wStocks.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for stocks, got %d", wStocks.Code)
	}
	var stocksResp struct {
		Data []struct {
			TotalQuantity      float64 `json:"totalQuantity"`
			AvailableLotsCount int     `json:"availableLotsCount"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wStocks.Body.Bytes(), &stocksResp)
	if len(stocksResp.Data) == 0 {
		t.Fatalf("expected stock balance for created material")
	}
	// 120.50 + 79.50 = 200.00
	if stocksResp.Data[0].TotalQuantity != 200.00 {
		t.Errorf("expected total quantity 200.00, got %f", stocksResp.Data[0].TotalQuantity)
	}
	if stocksResp.Data[0].AvailableLotsCount != 2 {
		t.Errorf("expected 2 available lots, got %d", stocksResp.Data[0].AvailableLotsCount)
	}

	// 9. Test Multi-Tenant Isolation: Snøhund client views /stocks
	reqClientStocks, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/stocks", nil)
	reqClientStocks.Header.Set("Authorization", "Bearer "+snohundToken)
	wClientStocks := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wClientStocks, reqClientStocks)

	if wClientStocks.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for client stocks, got %d", wClientStocks.Code)
	}
	var clientStocksResp struct {
		Data []struct {
			ClientID string `json:"clientId"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wClientStocks.Body.Bytes(), &clientStocksResp)
	for _, b := range clientStocksResp.Data {
		if b.ClientID != snoClientID {
			t.Fatalf("SECURITY VIOLATION: Client Snøhund can see materials of client %s", b.ClientID)
		}
	}
}

func TestApp_PrintInventoryActualResponses(t *testing.T) {
	_ = godotenv.Load("../../.env")
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	application, err := New(ctx)
	if err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	wrhToken := getAuthToken(t, application, "warehouse@lukla.local", "password")

	// 1. GET /api/v1/inventory/stocks
	reqStocks, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/stocks", nil)
	reqStocks.Header.Set("Authorization", "Bearer "+wrhToken)
	wStocks := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wStocks, reqStocks)
	var prettyStocks bytes.Buffer
	_ = json.Indent(&prettyStocks, wStocks.Body.Bytes(), "", "    ")
	t.Logf("\n=== ACTUAL GET /api/v1/inventory/stocks ===\n%s\n", prettyStocks.String())

	// 2. GET /api/v1/inventory/receipts
	reqReceipts, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/receipts", nil)
	reqReceipts.Header.Set("Authorization", "Bearer "+wrhToken)
	wReceipts := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wReceipts, reqReceipts)
	var prettyReceipts bytes.Buffer
	_ = json.Indent(&prettyReceipts, wReceipts.Body.Bytes(), "", "    ")
	t.Logf("\n=== ACTUAL GET /api/v1/inventory/receipts ===\n%s\n", prettyReceipts.String())

	// 3. GET /api/v1/inventory/materials
	reqMaterials, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/materials", nil)
	reqMaterials.Header.Set("Authorization", "Bearer "+wrhToken)
	wMaterials := httptest.NewRecorder()
	application.server.Handler.ServeHTTP(wMaterials, reqMaterials)
	var prettyMaterials bytes.Buffer
	_ = json.Indent(&prettyMaterials, wMaterials.Body.Bytes(), "", "    ")
	t.Logf("\n=== ACTUAL GET /api/v1/inventory/materials ===\n%s\n", prettyMaterials.String())
}
