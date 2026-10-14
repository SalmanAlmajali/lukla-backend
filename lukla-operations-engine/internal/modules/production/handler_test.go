package production

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockRepository struct {
	lines             []ProductionLine
	lineDetails       map[string]*ProductionLineDetail
	allocations       []LineAllocation
	transferLogs      map[string][]AllocationTransferLog
	outputs           []ProductionOutput
	factoryCapacity   int
	operatorErr       error
	orderErr          error
	componentErr      error
	transferErr       error
	outputErr         error
	sewingRate        float64
	activeAllocResult *LineAllocation
}

func (m *mockRepository) ListLines(ctx context.Context) ([]ProductionLine, error) {
	return m.lines, nil
}

func (m *mockRepository) FindLineByID(ctx context.Context, id string) (*ProductionLineDetail, error) {
	if d, ok := m.lineDetails[id]; ok {
		return d, nil
	}
	return nil, ErrLineNotFound
}

func (m *mockRepository) FindLineByCode(ctx context.Context, code string) (*ProductionLine, error) {
	for _, l := range m.lines {
		if l.Code == code {
			return &l, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) CreateLine(ctx context.Context, line *ProductionLine) error {
	line.ID = "new-line-uuid-123"
	line.CreatedAt = time.Now()
	line.UpdatedAt = time.Now()
	m.lines = append(m.lines, *line)
	return nil
}

func (m *mockRepository) UpdateLine(ctx context.Context, line *ProductionLine) error {
	for i, l := range m.lines {
		if l.ID == line.ID {
			m.lines[i] = *line
			return nil
		}
	}
	return ErrLineNotFound
}

func (m *mockRepository) GetTotalFactoryCapacity(ctx context.Context, excludeLineID string) (int, error) {
	return m.factoryCapacity, nil
}

func (m *mockRepository) ListAllocations(ctx context.Context, filter ListAllocationsFilter) ([]LineAllocation, error) {
	return m.allocations, nil
}

func (m *mockRepository) FindAllocationByID(ctx context.Context, id string) (*LineAllocation, error) {
	for _, a := range m.allocations {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, ErrAllocationNotFound
}

func (m *mockRepository) FindActiveAllocationByOperator(ctx context.Context, operatorID string) (*LineAllocation, error) {
	return m.activeAllocResult, nil
}

func (m *mockRepository) CreateAllocation(ctx context.Context, alloc *LineAllocation) error {
	alloc.ID = "new-alloc-uuid-456"
	alloc.AssignedAt = time.Now()
	m.allocations = append(m.allocations, *alloc)
	return nil
}

func (m *mockRepository) TransferOperator(ctx context.Context, oldAllocID string, newAlloc *LineAllocation, transferLog *AllocationTransferLog) error {
	if m.transferErr != nil {
		return m.transferErr
	}
	newAlloc.ID = "new-transferred-alloc-uuid-789"
	newAlloc.AssignedAt = time.Now()
	transferLog.ID = "transfer-log-uuid-001"
	transferLog.TransferredAt = time.Now()
	return nil
}

func (m *mockRepository) ListTransferHistoriesByAllocation(ctx context.Context, allocationID string) ([]AllocationTransferLog, error) {
	if logs, ok := m.transferLogs[allocationID]; ok {
		return logs, nil
	}
	return []AllocationTransferLog{}, nil
}

func (m *mockRepository) ValidateOperator(ctx context.Context, operatorID string) (string, string, string, error) {
	if m.operatorErr != nil {
		return "", "", "", m.operatorErr
	}
	return "Siti Rahmawati", "JH-001", "SENIOR", nil
}

func (m *mockRepository) ValidateOrder(ctx context.Context, orderID string) (string, string, string, error) {
	if m.orderErr != nil {
		return "", "", "", m.orderErr
	}
	return "PO-2026-10-001", "client-uuid", "article-uuid", nil
}

func (m *mockRepository) ValidateComponent(ctx context.Context, componentID string) (string, error) {
	if m.componentErr != nil {
		return "", m.componentErr
	}
	return "Saku Depan", nil
}

func (m *mockRepository) ListOutputs(ctx context.Context, filter ListOutputsFilter) ([]ProductionOutput, error) {
	if m.outputErr != nil {
		return nil, m.outputErr
	}
	return m.outputs, nil
}

func (m *mockRepository) FindOutputByID(ctx context.Context, id string) (*ProductionOutput, error) {
	if m.outputErr != nil {
		return nil, m.outputErr
	}
	for _, o := range m.outputs {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, ErrOutputNotFound
}

func (m *mockRepository) CreateOutput(ctx context.Context, output *ProductionOutput) error {
	if m.outputErr != nil {
		return m.outputErr
	}
	output.ID = "new-output-uuid-111"
	output.CreatedAt = time.Now()
	output.UpdatedAt = time.Now()
	m.outputs = append(m.outputs, *output)
	return nil
}

func (m *mockRepository) CreateOutputsBatch(ctx context.Context, outputs []*ProductionOutput) error {
	if m.outputErr != nil {
		return m.outputErr
	}
	for i, o := range outputs {
		o.ID = fmt.Sprintf("new-batch-output-%d", i+1)
		o.CreatedAt = time.Now()
		o.UpdatedAt = time.Now()
		m.outputs = append(m.outputs, *o)
	}
	return nil
}

func (m *mockRepository) GetActiveSewingRate(ctx context.Context, articleID string, componentID string) (float64, error) {
	if m.sewingRate > 0 {
		return m.sewingRate, nil
	}
	return 2500.0, nil
}

func setupTestRouter(repo Repository, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	authnFake := func(c *gin.Context) {
		c.Set("role", role)
		c.Set("user_id", "user-uuid-123")
		if role == "client" {
			c.Set("client_id", "client-uuid-snohund")
		}
		c.Next()
	}

	svc := NewService(repo)
	h := NewHandler(svc)
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1, authnFake)

	return r
}

func TestHandler_ListLines(t *testing.T) {
	repo := &mockRepository{
		lines: []ProductionLine{
			{ID: "line-1", Code: "LINE-01", Name: "Line 1", MaxCapacity: 24, Status: "ACTIVE"},
		},
	}

	t.Run("200 OK for executive", func(t *testing.T) {
		r := setupTestRouter(repo, "owner_executive")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lines", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("403 Forbidden for client", func(t *testing.T) {
		r := setupTestRouter(repo, "client")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/lines", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}

func TestHandler_CreateLine(t *testing.T) {
	repo := &mockRepository{
		factoryCapacity: 46,
		lines:           []ProductionLine{{Code: "LINE-01"}},
	}

	t.Run("201 Created for production head", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"code":        "LINE-02",
			"name":        "Line Celana",
			"maxCapacity": 20,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lines", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", w.Code)
		}
	})

	t.Run("409 Conflict on duplicate code", func(t *testing.T) {
		r := setupTestRouter(repo, "owner_executive")
		body, _ := json.Marshal(map[string]interface{}{
			"code":        "LINE-01",
			"name":        "Duplicate Line",
			"maxCapacity": 10,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lines", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("409 Conflict on factory capacity exceeded > 70", func(t *testing.T) {
		repoExceed := &mockRepository{
			factoryCapacity: 60,
		}
		r := setupTestRouter(repoExceed, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"code":        "LINE-BIG",
			"name":        "Too Big Line",
			"maxCapacity": 15, // 60 + 15 = 75 > 70
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/lines", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})
}

func TestHandler_LineAllocation_And_Transfer(t *testing.T) {
	repo := &mockRepository{
		lineDetails: map[string]*ProductionLineDetail{
			"line-1": {
				ProductionLine: ProductionLine{ID: "line-1", Code: "LINE-01", MaxCapacity: 20, ActiveOperatorsCount: 5, Status: "ACTIVE"},
			},
			"line-2": {
				ProductionLine: ProductionLine{ID: "line-2", Code: "LINE-02", MaxCapacity: 20, ActiveOperatorsCount: 10, Status: "ACTIVE"},
			},
		},
		allocations: []LineAllocation{
			{
				ID:         "alloc-1",
				LineID:     "line-1",
				LineCode:   "LINE-01",
				OperatorID: "op-1",
				Status:     "ACTIVE",
			},
		},
	}

	t.Run("201 Created for allocating operator", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":     "line-1",
			"orderId":    "order-uuid-1",
			"operatorId": "op-new",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("409 Conflict if operator already active", func(t *testing.T) {
		repoBusy := &mockRepository{
			activeAllocResult: &LineAllocation{ID: "busy-alloc", Status: "ACTIVE"},
			lineDetails:       repo.lineDetails,
		}
		r := setupTestRouter(repoBusy, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":     "line-1",
			"orderId":    "order-uuid-1",
			"operatorId": "op-busy",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("200 OK for operator line transfer (BPMN 04)", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"destinationLineId": "line-2",
			"transferReason":    "Mesin jahit di Line 1 rusak",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations/alloc-1/transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp struct {
			Success bool           `json:"success"`
			Data    TransferResult `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if !resp.Success || resp.Data.DestinationLineCode != "LINE-02" {
			t.Fatalf("unexpected transfer response: %+v", resp)
		}
	})

	t.Run("400 Bad Request if transfer reason is empty", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"destinationLineId": "line-2",
			"transferReason":    "",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations/alloc-1/transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("400 Bad Request on same line transfer", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"destinationLineId": "line-1", // same line as alloc-1
			"transferReason":    "Alasan",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/line-allocations/alloc-1/transfer", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("200 OK list transfer history", func(t *testing.T) {
		repoWithHistory := &mockRepository{
			transferLogs: map[string][]AllocationTransferLog{
				"alloc-1": {
					{
						ID:           "log-1",
						AllocationID: "alloc-1",
						FromLineCode: "LINE-01",
						ToLineCode:   "LINE-02",
						Reason:       "Mesin rusak",
					},
				},
			},
		}
		r := setupTestRouter(repoWithHistory, "production_head")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/line-allocations/alloc-1/history", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestProductionOutputHandlers(t *testing.T) {
	sampleOutputs := []ProductionOutput{
		{
			ID:                 "output-1",
			LineID:             "line-1",
			LineCode:           "LINE-01",
			OrderID:            "order-1",
			OperatorID:         "op-1",
			OperatorName:       "Siti Rahmawati",
			ComponentID:        "comp-1",
			ComponentName:      "Saku Depan",
			Quantity:           25,
			TimeSlot:           "08:00-08:30",
			WorkDate:           "2026-10-14",
			SnapshotSewingRate: 2500.0,
			RecordedByUserName: "user-uuid-123",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}

	sampleLines := map[string]*ProductionLineDetail{
		"line-1": {
			ProductionLine: ProductionLine{
				ID:          "line-1",
				Code:        "LINE-01",
				MaxCapacity: 20,
				Status:      "ACTIVE",
			},
		},
	}

	repo := &mockRepository{
		lineDetails: sampleLines,
		outputs:     sampleOutputs,
	}

	t.Run("200 OK list production outputs", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/production-outputs?date=2026-10-14", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("200 OK get production output detail", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/production-outputs/output-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("404 Not Found get production output detail non-existent", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/production-outputs/non-existent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("201 Created single production output with snapshot rate", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":      "line-1",
			"orderId":     "order-1",
			"operatorId":  "op-1",
			"componentId": "comp-1",
			"quantity":    30,
			"timeSlot":    "08:30-09:00",
			"workDate":    "2026-10-14",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		if data["snapshotSewingRate"] != float64(2500) {
			t.Fatalf("expected snapshotSewingRate 2500, got %v", data["snapshotSewingRate"])
		}
	})

	t.Run("400 Bad Request on invalid quantity <= 0", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":      "line-1",
			"orderId":     "order-1",
			"operatorId":  "op-1",
			"componentId": "comp-1",
			"quantity":    0,
			"timeSlot":    "08:30-09:00",
			"workDate":    "2026-10-14",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("201 Created batch production outputs", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":   "line-1",
			"orderId":  "order-1",
			"timeSlot": "09:00-09:30",
			"workDate": "2026-10-14",
			"items": []map[string]interface{}{
				{
					"operatorId":  "op-1",
					"componentId": "comp-1",
					"quantity":    15,
				},
				{
					"operatorId":  "op-2",
					"componentId": "comp-2",
					"quantity":    18,
				},
			},
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("400 Bad Request batch with empty items", func(t *testing.T) {
		r := setupTestRouter(repo, "production_head")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":   "line-1",
			"orderId":  "order-1",
			"timeSlot": "09:00-09:30",
			"workDate": "2026-10-14",
			"items":    []map[string]interface{}{},
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("403 Forbidden for client role accessing mandor terminal", func(t *testing.T) {
		r := setupTestRouter(repo, "client")
		body, _ := json.Marshal(map[string]interface{}{
			"lineId":      "line-1",
			"orderId":     "order-1",
			"operatorId":  "op-1",
			"componentId": "comp-1",
			"quantity":    20,
			"timeSlot":    "09:30-10:00",
			"workDate":    "2026-10-14",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/production-outputs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for client role, got %d", w.Code)
		}
	})
}

