package ratecard

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	listRateCardsFn               func(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error)
	findRateCardByIDFn            func(ctx context.Context, id string) (*RateCard, error)
	findRateCardByComponentFn     func(ctx context.Context, articleID, componentID string) (*RateCard, error)
	createRateCardFn              func(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error
	updateRateCardFn              func(ctx context.Context, rc *RateCard, history *RateCardHistory) error
	listRateCardHistoriesFn       func(ctx context.Context, rateCardID string) ([]RateCardHistory, error)
	validateComponentHierarchyFn func(ctx context.Context, clientID, articleID, componentID string) (*ComponentValidationResult, error)
}

func (m *mockRepository) ListRateCards(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error) {
	if m.listRateCardsFn != nil {
		return m.listRateCardsFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockRepository) FindRateCardByID(ctx context.Context, id string) (*RateCard, error) {
	if m.findRateCardByIDFn != nil {
		return m.findRateCardByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) FindRateCardByComponent(ctx context.Context, articleID, componentID string) (*RateCard, error) {
	if m.findRateCardByComponentFn != nil {
		return m.findRateCardByComponentFn(ctx, articleID, componentID)
	}
	return nil, nil
}

func (m *mockRepository) CreateRateCard(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error {
	if m.createRateCardFn != nil {
		return m.createRateCardFn(ctx, rc, initialHistory)
	}
	return nil
}

func (m *mockRepository) UpdateRateCard(ctx context.Context, rc *RateCard, history *RateCardHistory) error {
	if m.updateRateCardFn != nil {
		return m.updateRateCardFn(ctx, rc, history)
	}
	return nil
}

func (m *mockRepository) ListRateCardHistories(ctx context.Context, rateCardID string) ([]RateCardHistory, error) {
	if m.listRateCardHistoriesFn != nil {
		return m.listRateCardHistoriesFn(ctx, rateCardID)
	}
	return nil, nil
}

func (m *mockRepository) ValidateComponentHierarchy(ctx context.Context, clientID, articleID, componentID string) (*ComponentValidationResult, error) {
	if m.validateComponentHierarchyFn != nil {
		return m.validateComponentHierarchyFn(ctx, clientID, articleID, componentID)
	}
	return &ComponentValidationResult{
		ClientName:    "Test Client",
		ArticleTitle:  "Test Article",
		ComponentName: "Test Component",
	}, nil
}

func setupTestRateCardRouter(repo Repository, role, clientID, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authMiddleware := func(c *gin.Context) {
		if role != "" {
			c.Set("role", role)
		}
		if clientID != "" {
			c.Set("client_id", clientID)
		}
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	}

	svc := NewService(repo)
	handler := NewHandler(svc)

	v1 := r.Group("/api/v1")
	handler.RegisterRoutes(v1, authMiddleware)

	return r
}

func TestListRateCards_InternalStaff_Success(t *testing.T) {
	repo := &mockRepository{
		listRateCardsFn: func(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error) {
			return []RateCard{
				{
					ID:            "rc-1",
					ClientID:      "c-1",
					ClientName:    "Snohund",
					ArticleID:     "art-1",
					ArticleTitle:  "Parka",
					ComponentID:   "cmp-1",
					ComponentName: "Hood",
					SewingRate:    4500,
					CuttingRate:   500,
				},
			}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool       `json:"success"`
		Data    []RateCard `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "rc-1", resp.Data[0].ID)
}

func TestListRateCards_ClientSelfOnly_Success(t *testing.T) {
	repo := &mockRepository{
		listRateCardsFn: func(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error) {
			assert.Equal(t, "client-own-id", filter.ClientID)
			return []RateCard{
				{
					ID:         "rc-client",
					ClientID:   "client-own-id",
					SewingRate: 5000,
				},
			}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "client", "client-own-id", "u-client")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListRateCards_ClientOtherClient_Forbidden(t *testing.T) {
	repo := &mockRepository{}
	router := setupTestRateCardRouter(repo, "client", "client-own-id", "u-client")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards?client_id=other-client-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetRateCardDetail_Success(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			if id == "rc-10" {
				return &RateCard{
					ID:         "rc-10",
					ClientID:   "c-1",
					SewingRate: 7500,
				}, nil
			}
			return nil, nil
		},
	}

	router := setupTestRateCardRouter(repo, "production_head", "", "u-prod")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards/rc-10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRateCardDetail_NotFound(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			return nil, nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards/rc-nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetRateCardDetail_ClientOtherClient_Forbidden(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			return &RateCard{
				ID:       "rc-foreign",
				ClientID: "other-client-id",
			}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "client", "my-client-id", "u-client")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards/rc-foreign", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateRateCard_Executive_Success(t *testing.T) {
	repo := &mockRepository{
		findRateCardByComponentFn: func(ctx context.Context, articleID, componentID string) (*RateCard, error) {
			return nil, nil
		},
		createRateCardFn: func(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error {
			rc.ID = "rc-new-uuid"
			rc.CreatedAt = time.Now()
			rc.UpdatedAt = time.Now()
			return nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	body := CreateRateCardRequest{
		ClientID:    "c-1",
		ArticleID:   "art-1",
		ComponentID: "cmp-1",
		SewingRate:  6000,
		CuttingRate: 1000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateRateCard_ProductionHead_Success(t *testing.T) {
	repo := &mockRepository{
		createRateCardFn: func(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error {
			rc.ID = "rc-prod-uuid"
			return nil
		},
	}

	router := setupTestRateCardRouter(repo, "production_head", "", "u-prod")

	body := CreateRateCardRequest{
		ClientID:    "c-1",
		ArticleID:   "art-1",
		ComponentID: "cmp-1",
		SewingRate:  6500,
		CuttingRate: 1000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateRateCard_Client_Forbidden(t *testing.T) {
	repo := &mockRepository{}
	router := setupTestRateCardRouter(repo, "client", "c-1", "u-client")

	body := CreateRateCardRequest{
		ClientID:    "c-1",
		ArticleID:   "art-1",
		ComponentID: "cmp-1",
		SewingRate:  6500,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateRateCard_Duplicate_Conflict(t *testing.T) {
	repo := &mockRepository{
		findRateCardByComponentFn: func(ctx context.Context, articleID, componentID string) (*RateCard, error) {
			return &RateCard{ID: "rc-exists"}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	body := CreateRateCardRequest{
		ClientID:    "c-1",
		ArticleID:   "art-1",
		ComponentID: "cmp-1",
		SewingRate:  6500,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateRateCard_ComponentMismatch_BadRequest(t *testing.T) {
	repo := &mockRepository{
		validateComponentHierarchyFn: func(ctx context.Context, clientID, articleID, componentID string) (*ComponentValidationResult, error) {
			return nil, ErrComponentMismatch
		},
	}

	router := setupTestRateCardRouter(repo, "production_head", "", "u-prod")

	body := CreateRateCardRequest{
		ClientID:    "c-1",
		ArticleID:   "art-1",
		ComponentID: "cmp-other",
		SewingRate:  6500,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/rate-cards", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateRateCard_Executive_Success(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			return &RateCard{
				ID:          "rc-1",
				SewingRate:  5000,
				CuttingRate: 800,
			}, nil
		},
		updateRateCardFn: func(ctx context.Context, rc *RateCard, history *RateCardHistory) error {
			assert.Equal(t, float64(5500), rc.SewingRate)
			assert.Equal(t, "Penyesuaian benang", history.Reason)
			return nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	body := UpdateRateCardRequest{
		SewingRate:   5500,
		ChangeReason: "Penyesuaian benang",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/rate-cards/rc-1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateRateCard_MissingReason_BadRequest(t *testing.T) {
	repo := &mockRepository{}
	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	body := map[string]interface{}{
		"sewingRate": 6000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/rate-cards/rc-1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateRateCard_Client_Forbidden(t *testing.T) {
	repo := &mockRepository{}
	router := setupTestRateCardRouter(repo, "client", "c-1", "u-client")

	body := UpdateRateCardRequest{
		SewingRate:   6000,
		ChangeReason: "Client update",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/rate-cards/rc-1", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListRateCardHistories_Success(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			return &RateCard{ID: "rc-1", ClientID: "c-1"}, nil
		},
		listRateCardHistoriesFn: func(ctx context.Context, rateCardID string) ([]RateCardHistory, error) {
			return []RateCardHistory{
				{
					ID:            "hist-1",
					RateCardID:    "rc-1",
					OldSewingRate: 5000,
					NewSewingRate: 5500,
					Reason:        "Kenaikan tarif",
					ChangedByName: "Budi Santoso",
				},
			}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "owner_executive", "", "u-exec")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards/rc-1/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool              `json:"success"`
		Data    []RateCardHistory `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "Budi Santoso", resp.Data[0].ChangedByName)
}

func TestListRateCardHistories_ClientOther_Forbidden(t *testing.T) {
	repo := &mockRepository{
		findRateCardByIDFn: func(ctx context.Context, id string) (*RateCard, error) {
			return &RateCard{ID: "rc-foreign", ClientID: "c-other"}, nil
		},
	}

	router := setupTestRateCardRouter(repo, "client", "my-client", "u-client")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-cards/rc-foreign/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

