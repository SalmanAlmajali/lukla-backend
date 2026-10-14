package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
	"golang.org/x/crypto/bcrypt"
)

type mockRepository struct {
	users map[string]*User
}

func (m *mockRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockRepository) FindByID(ctx context.Context, id string) (*User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

const testJWTSecret = "test-secret-key-1234567890123456"

func setupTestRouter() (*gin.Engine, *mockRepository, *Service) {
	gin.SetMode(gin.TestMode)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	hashStr := string(hash)
	emailActive := "production@lukla.local"
	emailTailor := "tailor@lukla.local"
	emailClient := "client@lukla.local"
	clientIDVal := "c001-mock-client-id"

	repo := &mockRepository{
		users: map[string]*User{
			emailActive: {
				ID:           "test-uuid-prod",
				FullName:     "Bambang Supriyadi",
				Email:        &emailActive,
				PasswordHash: &hashStr,
				RoleCode:     "production_head",
				RoleName:     "Kepala Produksi / Mandor",
				CanAccessApp: true,
				IsActive:     true,
			},
			emailTailor: {
				ID:           "test-uuid-tailor",
				FullName:     "Ahmad Supriatna",
				Email:        &emailTailor,
				PasswordHash: nil,
				RoleCode:     "tailor",
				RoleName:     "Penjahit / Tailor",
				CanAccessApp: false,
				IsActive:     true,
			},
			emailClient: {
				ID:           "test-uuid-client",
				FullName:     "Klien Snøhund",
				Email:        &emailClient,
				PasswordHash: &hashStr,
				RoleCode:     "client",
				RoleName:     "Klien Maklon / External Client",
				ClientID:     &clientIDVal,
				CanAccessApp: true,
				IsActive:     true,
			},
		},
	}

	svc := NewService(repo, testJWTSecret, 1*time.Hour)
	handler := NewHandler(svc)
	authnMiddleware := authn.Middleware(testJWTSecret)

	r := gin.New()
	RegisterRoutes(&r.RouterGroup, handler, authnMiddleware)

	return r, repo, svc
}

func TestLogin_Success(t *testing.T) {
	router, _, _ := setupTestRouter()

	body := map[string]string{
		"username": "production@lukla.local",
		"password": "password",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Data.Token == "" {
		t.Errorf("expected non-empty token")
	}
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	router, _, _ := setupTestRouter()

	body := map[string]string{
		"username": "not-an-email",
		"password": "password",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid email, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	router, _, _ := setupTestRouter()

	body := map[string]string{
		"username": "production@lukla.local",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_TailorNoAppAccess(t *testing.T) {
	router, _, _ := setupTestRouter()

	body := map[string]string{
		"username": "tailor@lukla.local",
		"password": "password",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for tailor without app access, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestMe_Success(t *testing.T) {
	router, _, svc := setupTestRouter()

	// 1. Login first to get token
	loginData, err := svc.Login(context.Background(), LoginRequest{
		Username: "production@lukla.local",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// 2. Call GET /auth/me with Bearer token
	req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginData.Token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    UserResponse `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Data.ID != "test-uuid-prod" {
		t.Errorf("expected id test-uuid-prod, got %s", resp.Data.ID)
	}
	if resp.Data.Name != "Bambang Supriyadi" {
		t.Errorf("expected name Bambang Supriyadi, got %s", resp.Data.Name)
	}
	if resp.Data.Role != "KEPALA_PRODUKSI" {
		t.Errorf("expected role KEPALA_PRODUKSI, got %s", resp.Data.Role)
	}
}

func TestMe_Unauthorized_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for missing token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestMe_Unauthorized_InvalidToken(t *testing.T) {
	router, _, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.fake.token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for invalid token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestMe_WithClientID(t *testing.T) {
	router, _, svc := setupTestRouter()

	loginData, err := svc.Login(context.Background(), LoginRequest{
		Username: "client@lukla.local",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginData.Token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    UserResponse `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Data.Role != "CLIENT_PORTAL" {
		t.Errorf("expected role CLIENT_PORTAL, got %s", resp.Data.Role)
	}
	if resp.Data.ClientID == nil || *resp.Data.ClientID != "c001-mock-client-id" {
		t.Errorf("expected ClientID c001-mock-client-id, got %v", resp.Data.ClientID)
	}
}

func TestLogout_Success(t *testing.T) {
	router, _, svc := setupTestRouter()

	loginData, err := svc.Login(context.Background(), LoginRequest{
		Username: "production@lukla.local",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+loginData.Token)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}
	if resp.Message != "Berhasil logout." {
		t.Errorf("expected message 'Berhasil logout.', got %s", resp.Message)
	}
}

func TestLogout_Unauthorized_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for missing token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestLogout_Unauthorized_InvalidToken(t *testing.T) {
	router, _, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for invalid token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

