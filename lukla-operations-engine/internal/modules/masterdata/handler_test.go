package masterdata

import (
	"bytes"
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
	clients            []Client
	articles           []Article
	components         []Component
	operators          []Operator
}

func (m *mockRepository) ListOperators(ctx context.Context, filter ListOperatorsFilter) ([]Operator, error) {
	var matched []Operator
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, op := range m.operators {
		if filter.Status != "" && !strings.EqualFold(string(op.Status), filter.Status) {
			continue
		}
		if filter.Line != "" {
			if op.DefaultLine == nil || !strings.EqualFold(*op.DefaultLine, filter.Line) {
				continue
			}
		}
		if search != "" {
			matchName := strings.Contains(strings.ToLower(op.Name), search)
			matchCode := strings.Contains(strings.ToLower(op.OperatorCode), search)
			if !matchName && !matchCode {
				continue
			}
		}
		matched = append(matched, op)
	}
	return matched, nil
}

func (m *mockRepository) FindOperatorByID(ctx context.Context, id string) (*Operator, error) {
	for _, op := range m.operators {
		if op.ID == id || strings.EqualFold(op.OperatorCode, id) {
			return &op, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) ListComponentsByArticleID(ctx context.Context, articleID string) ([]Component, error) {
	var comps []Component
	for _, c := range m.components {
		if c.ArticleID == articleID {
			comps = append(comps, c)
		}
	}
	return comps, nil
}

func (m *mockRepository) CreateComponent(ctx context.Context, comp *Component) error {
	comp.ID = fmt.Sprintf("mock-comp-%d", len(m.components)+1)
	comp.CreatedAt = time.Now()
	comp.UpdatedAt = time.Now()
	m.components = append(m.components, *comp)
	return nil
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

func (m *mockRepository) ListArticles(ctx context.Context, filter ListArticlesFilter) ([]Article, int64, error) {
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

	var matched []Article
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, a := range m.articles {
		if filter.ClientID != "" && a.ClientID != filter.ClientID {
			continue
		}
		if filter.Category != "" && a.Category != filter.Category {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(a.Title), search) && !strings.Contains(strings.ToLower(a.Code), search) {
			continue
		}
		matched = append(matched, a)
	}

	total := int64(len(matched))
	offset := (page - 1) * limit
	if offset >= len(matched) {
		return []Article{}, total, nil
	}

	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}

	return matched[offset:end], total, nil
}

func (m *mockRepository) FindArticleByID(ctx context.Context, id string) (*Article, error) {
	for _, a := range m.articles {
		if a.ID == id || strings.EqualFold(a.Code, id) {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) CreateArticle(ctx context.Context, article *Article) error {
	article.ID = fmt.Sprintf("mock-art-%d", len(m.articles)+1)
	article.CreatedAt = time.Now()
	article.UpdatedAt = time.Now()
	for _, c := range m.clients {
		if c.ID == article.ClientID {
			article.ClientName = c.Name
			break
		}
	}
	m.articles = append(m.articles, *article)
	return nil
}

func (m *mockRepository) ArticleCodeExists(ctx context.Context, code string) (bool, error) {
	for _, a := range m.articles {
		if strings.EqualFold(a.Code, code) {
			return true, nil
		}
	}
	return false, nil
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

	mockArticles := []Article{
		{
			ID:         "art-01",
			ClientID:   "mock-id-01",
			ClientName: "Snøhund Brand",
			Code:       "ART-SNO-PK01",
			Title:      "Snøhund Arctic Parka",
			Category:   "GARMENT",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "art-02",
			ClientID:   "mock-id-01",
			ClientName: "Snøhund Brand",
			Code:       "ART-SNO-BP01",
			Title:      "Snøhund Summit Pack",
			Category:   "BAG",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "art-03",
			ClientID:   "mock-id-02",
			ClientName: "Client Brand 02",
			Code:       "ART-AVE-TR01",
			Title:      "Aventura Cargo Pants",
			Category:   "GARMENT",
			CreatedAt:  time.Now(),
		},
		{
			ID:         "art-04",
			ClientID:   "mock-id-02",
			ClientName: "Client Brand 02",
			Code:       "ART-AVE-DP01",
			Title:      "Aventura Daypack",
			Category:   "BAG",
			CreatedAt:  time.Now(),
		},
	}

	sam35 := 3.50
	sam48 := 4.80
	sam52 := 5.20
	sam50 := 5.00
	code1 := "CMP-SNO-PK-01"
	code2 := "CMP-SNO-PK-02"
	code3 := "CMP-SNO-PK-03"
	code4 := "CMP-AVE-TR-01"

	mockComponents := []Component{
		{
			ID:             "comp-01",
			ArticleID:      "art-01",
			Code:           &code1,
			Name:           "Potong Pola",
			SequenceNumber: 1,
			StandardSAM:    &sam35,
		},
		{
			ID:             "comp-02",
			ArticleID:      "art-01",
			Code:           &code2,
			Name:           "Jahit Saku Kargo",
			SequenceNumber: 2,
			StandardSAM:    &sam48,
		},
		{
			ID:             "comp-03",
			ArticleID:      "art-01",
			Code:           &code3,
			Name:           "Pasang Resleting",
			SequenceNumber: 3,
			StandardSAM:    &sam52,
		},
		{
			ID:             "comp-04",
			ArticleID:      "art-03",
			Code:           &code4,
			Name:           "Jahit Saku Celana",
			SequenceNumber: 1,
			StandardSAM:    &sam50,
		},
	}

	line1 := "LINE-1"
	line2 := "LINE-2"
	tierSenior := "SENIOR"
	tierMed := "MEDIUM"
	tierJun := "JUNIOR"
	phone1 := "081122334411"
	phone2 := "081122334412"
	phone3 := "081122334413"
	phone4 := "081122334414"

	mockOperators := []Operator{
		{
			ID:           "op-01",
			OperatorCode: "JH-001",
			Name:         "Ahmad Supriatna",
			Phone:        &phone1,
			JoinDate:     "2025-01-15",
			Status:       OperatorStatusActive,
			DefaultLine:  &line1,
			SkillTier:    &tierSenior,
		},
		{
			ID:           "op-02",
			OperatorCode: "JH-002",
			Name:         "Dedi Kusnadi",
			Phone:        &phone2,
			JoinDate:     "2025-06-01",
			Status:       OperatorStatusActive,
			DefaultLine:  &line1,
			SkillTier:    &tierMed,
		},
		{
			ID:           "op-03",
			OperatorCode: "JH-003",
			Name:         "Endang Suhendar",
			Phone:        &phone3,
			JoinDate:     "2025-03-10",
			Status:       OperatorStatusOnLeave,
			DefaultLine:  &line2,
			SkillTier:    &tierSenior,
		},
		{
			ID:           "op-04",
			OperatorCode: "JH-004",
			Name:         "Iwan Setiawan",
			Phone:        &phone4,
			JoinDate:     "2025-08-20",
			Status:       OperatorStatusInactive,
			DefaultLine:  &line2,
			SkillTier:    &tierJun,
		},
		{
			ID:           "op-05",
			OperatorCode: "JH-005",
			Name:         "Asep Saepudin",
			JoinDate:     "2026-01-10",
			Status:       OperatorStatusActive,
			DefaultLine:  nil,
			SkillTier:    &tierMed,
		},
	}

	repo := &mockRepository{
		clients:    mockClients,
		articles:   mockArticles,
		components: mockComponents,
		operators:  mockOperators,
	}

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
		} else if userID == "user-prod-head" {
			role = "production_head"
		} else if userID == "user-tailor" {
			role = "tailor"
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

// ---------------- Articles Tests ---------------- //

func TestListArticles_Internal_All(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    []Article            `json:"data"`
		Meta    httpx.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if len(resp.Data) != 4 {
		t.Errorf("expected 4 articles, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 4 {
		t.Errorf("expected total 4, got %d", resp.Meta.Total)
	}
}

func TestListArticles_FilterCategory(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles?category=GARMENT", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool      `json:"success"`
		Data    []Article `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 garment articles, got %d", len(resp.Data))
	}
	for _, art := range resp.Data {
		if art.Category != "GARMENT" {
			t.Errorf("expected category GARMENT, got %s", art.Category)
		}
	}
}

func TestListArticles_FilterClientID_Internal(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles?clientId=mock-id-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool      `json:"success"`
		Data    []Article `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 articles for mock-id-01, got %d", len(resp.Data))
	}
	for _, art := range resp.Data {
		if art.ClientID != "mock-id-01" {
			t.Errorf("expected clientId mock-id-01, got %s", art.ClientID)
		}
	}
}

func TestListArticles_ClientPortal_NoClientID_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01

	// Client requests general articles without specifying clientId -> 403 Forbidden!
	req, _ := http.NewRequest(http.MethodGet, "/articles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client queries general articles without clientId, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListArticles_ClientPortal_OwnClient_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01

	// Client requests articles by explicitly supplying their own clientId -> 200 OK!
	req, _ := http.NewRequest(http.MethodGet, "/articles?clientId=mock-id-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool      `json:"success"`
		Data    []Article `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 articles for mock-id-01, got %d", len(resp.Data))
	}
	for _, art := range resp.Data {
		if art.ClientID != "mock-id-01" {
			t.Errorf("expected clientId mock-id-01, got %s", art.ClientID)
		}
	}
}

func TestListArticles_ClientPortal_OtherClient_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01

	req, _ := http.NewRequest(http.MethodGet, "/articles?clientId=mock-id-02", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client queries another client's articles, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestArticlesAndComponents_Unauthorized(t *testing.T) {
	r := setupTestRouter()

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/articles"},
		{http.MethodPost, "/articles"},
		{http.MethodGet, "/articles/art-01"},
		{http.MethodGet, "/articles/art-01/components"},
		{http.MethodPost, "/articles/art-01/components"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized for unauthenticated request to %s %s, got %d", ep.method, ep.path, w.Code)
			}
		})
	}
}

func TestGetArticleDetail_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    ArticleDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.ID != "art-01" {
		t.Errorf("expected art-01, got %s", resp.Data.ID)
	}
	if resp.Data.Code != "ART-SNO-PK01" {
		t.Errorf("expected ART-SNO-PK01, got %s", resp.Data.Code)
	}
}

func TestGetArticleDetail_NotFound(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles/non-existent-art", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestGetArticleDetail_ClientPortal_OtherClient_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01, trying to access art-03 (mock-id-02)

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-03", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for accessing other client article, got %d", w.Code)
	}
}

func TestCreateArticle_Internal_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	body := `{"clientId": "mock-id-01", "code": "ART-NEW-01", "title": "New Raincoat", "category": "GARMENT", "description": "Waterproof raincoat"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool    `json:"success"`
		Data    Article `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Code != "ART-NEW-01" {
		t.Errorf("expected ART-NEW-01, got %s", resp.Data.Code)
	}
}

func TestCreateArticle_ClientPortal_OwnClient_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01

	body := `{"clientId": "mock-id-01", "code": "ART-CLI-01", "title": "Client Parka", "category": "GARMENT"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateArticle_ClientPortal_OtherClient_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01, attempting to create for mock-id-02

	body := `{"clientId": "mock-id-02", "code": "ART-HACK-01", "title": "Hack Article", "category": "GARMENT"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client creates article for another client, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateArticle_InvalidCategory(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	body := `{"clientId": "mock-id-01", "code": "ART-INV-01", "title": "Shoes", "category": "FOOTWEAR"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid category, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateArticle_DuplicateCode(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	// ART-SNO-PK01 already exists
	body := `{"clientId": "mock-id-01", "code": "ART-SNO-PK01", "title": "Duplicate Parka", "category": "GARMENT"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for duplicate code, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// ---------------- Component Tests ---------------- //

func TestListComponents_Internal_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-01/components", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool        `json:"success"`
		Data    []Component `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 components for art-01, got %d", len(resp.Data))
	}
	if resp.Data[0].Name != "Potong Pola" {
		t.Errorf("expected Potong Pola, got %s", resp.Data[0].Name)
	}
	if resp.Data[0].StandardSAM == nil || *resp.Data[0].StandardSAM != 3.50 {
		t.Errorf("expected standardSAM 3.50, got %v", resp.Data[0].StandardSAM)
	}
}

func TestListComponents_ClientPortal_OwnArticle_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01 (owns art-01)

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-01/components", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for client accessing own article components, got %d", w.Code)
	}
}

func TestListComponents_ClientPortal_OtherClient_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01, art-03 belongs to mock-id-02

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-03/components", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client accessing other client's components, got %d", w.Code)
	}
}

func TestListComponents_ArticleNotFound(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles/non-existent-art/components", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestCreateComponent_Internal_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	body := `{"name": "Pasang Label & Hangtag", "code": "CMP-SNO-PK-04", "sequenceNumber": 4, "standardSAM": 2.25}`
	req, _ := http.NewRequest(http.MethodPost, "/articles/art-01/components", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool      `json:"success"`
		Data    Component `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.Name != "Pasang Label & Hangtag" {
		t.Errorf("expected name Pasang Label & Hangtag, got %s", resp.Data.Name)
	}
	if resp.Data.StandardSAM == nil || *resp.Data.StandardSAM != 2.25 {
		t.Errorf("expected standardSAM 2.25, got %v", resp.Data.StandardSAM)
	}
}

func TestCreateComponent_ClientPortal_OtherClient_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to mock-id-01, art-03 belongs to mock-id-02

	body := `{"name": "Hack Component"}`
	req, _ := http.NewRequest(http.MethodPost, "/articles/art-03/components", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}

func TestGetArticleDetail_IncludesComponents(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/articles/art-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    ArticleDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data.Components) != 3 {
		t.Errorf("expected 3 components in article detail, got %d", len(resp.Data.Components))
	}
}

func TestListOperators_InternalStaff_Success(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool       `json:"success"`
		Data    []Operator `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 5 {
		t.Fatalf("expected 5 operators, got %d", len(resp.Data))
	}
}

func TestListOperators_FilterStatus_Active(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators?status=ACTIVE", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Data []Operator `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 ACTIVE operators, got %d", len(resp.Data))
	}
	for _, op := range resp.Data {
		if op.Status != OperatorStatusActive {
			t.Errorf("expected status ACTIVE, got %s", op.Status)
		}
	}
}

func TestListOperators_FilterStatus_OnLeave(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators?status=ON_LEAVE", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Data []Operator `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 ON_LEAVE operator, got %d", len(resp.Data))
	}
	if resp.Data[0].OperatorCode != "JH-003" {
		t.Errorf("expected operator JH-003, got %s", resp.Data[0].OperatorCode)
	}
}

func TestListOperators_FilterStatus_Invalid(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators?status=UNKNOWN", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid status, got %d", w.Code)
	}
}

func TestListOperators_FilterLine(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators?line=LINE-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Data []Operator `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 operators on LINE-1, got %d", len(resp.Data))
	}
	for _, op := range resp.Data {
		if op.DefaultLine == nil || *op.DefaultLine != "LINE-1" {
			t.Errorf("expected defaultLine LINE-1, got %v", op.DefaultLine)
		}
	}
}

func TestListOperators_Search(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators?search=Asep", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Data []Operator `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 operator matching 'Asep', got %d", len(resp.Data))
	}
	if resp.Data[0].OperatorCode != "JH-005" {
		t.Errorf("expected operator JH-005, got %s", resp.Data[0].OperatorCode)
	}
}

func TestListOperators_ClientRole_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to role "client"

	req, _ := http.NewRequest(http.MethodGet, "/operators", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client role accesses /operators, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListOperators_Unauthorized_NoToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/operators", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request to /operators, got %d", w.Code)
	}
}

func TestListOperators_Unauthorized_InvalidToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/operators", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d", w.Code)
	}
}

func TestGetOperatorDetail_Success_ByID(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators/op-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool     `json:"success"`
		Data    Operator `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Fatal("expected success true")
	}
	if resp.Data.ID != "op-01" {
		t.Errorf("expected op-01, got %s", resp.Data.ID)
	}
	if resp.Data.OperatorCode != "JH-001" {
		t.Errorf("expected JH-001, got %s", resp.Data.OperatorCode)
	}
	if resp.Data.Name != "Ahmad Supriatna" {
		t.Errorf("expected Ahmad Supriatna, got %s", resp.Data.Name)
	}
	if resp.Data.Status != OperatorStatusActive {
		t.Errorf("expected ACTIVE, got %s", resp.Data.Status)
	}
}

func TestGetOperatorDetail_Success_ByCode(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators/JH-002", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool     `json:"success"`
		Data    Operator `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.ID != "op-02" {
		t.Errorf("expected op-02, got %s", resp.Data.ID)
	}
	if resp.Data.OperatorCode != "JH-002" {
		t.Errorf("expected JH-002, got %s", resp.Data.OperatorCode)
	}
	if resp.Data.Name != "Dedi Kusnadi" {
		t.Errorf("expected Dedi Kusnadi, got %s", resp.Data.Name)
	}
}

func TestGetOperatorDetail_NotFound(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-internal-id")

	req, _ := http.NewRequest(http.MethodGet, "/operators/non-existent-operator-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetOperatorDetail_ClientRole_Forbidden(t *testing.T) {
	r := setupTestRouter()
	token := createTestToken("user-client-1") // bound to role "client"

	req, _ := http.NewRequest(http.MethodGet, "/operators/op-01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when client role accesses /operators/:id, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetOperatorDetail_Unauthorized_NoToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/operators/op-01", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request to /operators/:id, got %d", w.Code)
	}
}

func TestGetOperatorDetail_Unauthorized_InvalidToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/operators/op-01", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d", w.Code)
	}
}
