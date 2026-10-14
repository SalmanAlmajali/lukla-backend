package masterdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lukla/operations-engine/internal/platform/authn"
	"github.com/lukla/operations-engine/internal/platform/authz"
	"github.com/lukla/operations-engine/internal/platform/httpx"
)

type mockRepository struct {
	clients []Client
}

func (m *mockRepository) ListClients(ctx context.Context, filter ListClientsFilter) ([]Client, int64, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	var matched []Client
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, c := range m.clients {
		if search == "" || strings.Contains(strings.ToLower(c.Name), search) || strings.Contains(strings.ToLower(c.Code), search) {
			matched = append(matched, c)
		}
	}

	total := int64(len(matched))
	offset := (page - 1) * limit
	if offset >= len(matched) {
		return []Client{}, total, nil
	}

	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}

	return matched[offset:end], total, nil
}

func (m *mockRepository) FindClientByID(ctx context.Context, id string) (*ClientDetail, error) {
	for _, c := range m.clients {
		if c.ID == id || strings.EqualFold(c.Code, id) {
			return &ClientDetail{
				Client:            c,
				ArticlesCount:     2,
				ActiveOrdersCount: 1,
			}, nil
		}
	}
	return nil, nil
}

const testJWTSecret = "test-secret-key-1234567890123456"

func createTestToken(userID string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tStr, _ := token.SignedString([]byte(testJWTSecret))
	return tStr
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Create 12 mock clients to test pagination
	mockClients := make([]Client, 12)
	for i := 1; i <= 12; i++ {
		code := fmt.Sprintf("CLI-%03d", i)
		name := fmt.Sprintf("Client Brand %02d", i)
		if i == 1 {
			code = "SNOHUND"
			name = "Snøhund Brand"
		}
		mockClients[i-1] = Client{
			ID:         fmt.Sprintf("mock-id-%02d", i),
			Code:       code,
			Name:       name,
			ClientType: "B2B",
			CreatedAt:  time.Now(),
		}
	}

	repo := &mockRepository{clients: mockClients}
	svc := NewService(repo)
	handler := NewHandler(svc)

	userLoader := func(ctx context.Context, userID string) (*authn.UserIdentity, error) {
		role := "owner_executive"
		var clientID *string
		if userID == "user-client-1" || userID == "user-client-id" {
			role = "client"
			cid := "mock-id-01"
			clientID = &cid
		} else if userID == "user-client-2" {
			role = "client"
			cid := "mock-id-02"
			clientID = &cid
		}
		return &authn.UserIdentity{
			ID:       userID,
			RoleCode: role,
			ClientID: clientID,
			IsActive: true,
		}, nil
	}

	authnMiddleware := authn.Middleware(testJWTSecret, userLoader)
	internalAuthz := authz.RequireInternal()

	r := gin.New()
	RegisterRoutes(&r.RouterGroup, handler, authnMiddleware, internalAuthz)
	return r
}

func TestListClients_DefaultPagination(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    []Client             `json:"data"`
		Meta    httpx.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if len(resp.Data) != 10 {
		t.Errorf("expected 10 items on page 1, got %d", len(resp.Data))
	}
	if resp.Meta.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Meta.Page)
	}
	if resp.Meta.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Meta.Limit)
	}
	if resp.Meta.Total != 12 {
		t.Errorf("expected total 12, got %d", resp.Meta.Total)
	}
	if resp.Meta.TotalPages != 2 {
		t.Errorf("expected totalPages 2, got %d", resp.Meta.TotalPages)
	}
}

func TestListClients_Page2(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients?page=2&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    []Client             `json:"data"`
		Meta    httpx.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(resp.Data))
	}
	if resp.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Meta.Page)
	}
	if resp.Meta.TotalPages != 2 {
		t.Errorf("expected totalPages 2, got %d", resp.Meta.TotalPages)
	}
}

func TestListClients_CustomLimit(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients?page=2&limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    []Client             `json:"data"`
		Meta    httpx.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 5 {
		t.Errorf("expected 5 items on page 2 with limit 5, got %d", len(resp.Data))
	}
	if resp.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", resp.Meta.Page)
	}
	if resp.Meta.Limit != 5 {
		t.Errorf("expected limit 5, got %d", resp.Meta.Limit)
	}
	if resp.Meta.TotalPages != 3 {
		t.Errorf("expected totalPages 3, got %d", resp.Meta.TotalPages)
	}
}

func TestListClients_Search(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients?search=Snøhund", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    []Client             `json:"data"`
		Meta    httpx.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 item found for search 'Snøhund', got %d", len(resp.Data))
	}
	if resp.Data[0].Code != "SNOHUND" {
		t.Errorf("expected code SNOHUND, got %s", resp.Data[0].Code)
	}
	if resp.Meta.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Meta.Total)
	}
	if resp.Meta.TotalPages != 1 {
		t.Errorf("expected totalPages 1, got %d", resp.Meta.TotalPages)
	}
}

func TestListClients_Unauthorized_NoToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/clients", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListClients_Unauthorized_InvalidToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/clients", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListClients_Forbidden_ClientRole(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client role, got %d. Body: %s", w.Code, w.Body.String())
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
		t.Errorf("expected code FORBIDDEN, got %s", resp.Error.Code)
	}
}

func TestGetClientDetail_InternalStaff_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients/mock-id-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    ClientDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Data.ID != "mock-id-01" {
		t.Errorf("expected client id mock-id-01, got %s", resp.Data.ID)
	}
	if resp.Data.Code != "SNOHUND" {
		t.Errorf("expected code SNOHUND, got %s", resp.Data.Code)
	}
}

func TestGetClientDetail_ClientOwnData_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1")

	// user-client-1 is linked to mock-id-01
	req, _ := http.NewRequest(http.MethodGet, "/clients/mock-id-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for client accessing own data, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    ClientDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.ID != "mock-id-01" {
		t.Errorf("expected id mock-id-01, got %s", resp.Data.ID)
	}
}

func TestGetClientDetail_ClientOtherData_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1")

	// user-client-1 is linked to mock-id-01, trying to access mock-id-02
	req, _ := http.NewRequest(http.MethodGet, "/clients/mock-id-02", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client accessing another client's data, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "FORBIDDEN" {
		t.Errorf("expected error code FORBIDDEN, got %s", resp.Error.Code)
	}
}

func TestGetClientDetail_NotFound(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/clients/non-existent-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetClientDetail_Unauthorized(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/clients/mock-id-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing token, got %d. Body: %s", w.Code, w.Body.String())
	}
}

