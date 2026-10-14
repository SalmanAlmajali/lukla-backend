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





