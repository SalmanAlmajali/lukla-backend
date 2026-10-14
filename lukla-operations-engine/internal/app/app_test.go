package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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










