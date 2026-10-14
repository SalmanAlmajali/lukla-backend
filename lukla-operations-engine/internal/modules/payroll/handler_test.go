package payroll

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockPayrollRepository struct {
	periods        map[string]*PayrollPeriod
	slips          map[string]*PayrollSlip
	slipDetails    map[string]*PayrollSlipDetail
	adjustments    map[string][]PayrollAdjustment
	outputs        []OperatorOutputAggregation
	createErr      error
	calculateErr   error
	listSlipsTotal int
}

func newMockRepository() *mockPayrollRepository {
	return &mockPayrollRepository{
		periods:     make(map[string]*PayrollPeriod),
		slips:       make(map[string]*PayrollSlip),
		slipDetails: make(map[string]*PayrollSlipDetail),
		adjustments: make(map[string][]PayrollAdjustment),
	}
}

func (m *mockPayrollRepository) CreatePeriod(ctx context.Context, p *PayrollPeriod) error {
	if m.createErr != nil {
		return m.createErr
	}
	p.ID = "period-001"
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.periods[p.ID] = p
	return nil
}

func (m *mockPayrollRepository) FindPeriodByID(ctx context.Context, id string) (*PayrollPeriod, error) {
	if p, ok := m.periods[id]; ok {
		return p, nil
	}
	return nil, ErrPeriodNotFound
}

func (m *mockPayrollRepository) FindPeriodByCode(ctx context.Context, code string) (*PayrollPeriod, error) {
	for _, p := range m.periods {
		if p.PeriodCode == code {
			return p, nil
		}
	}
	return nil, ErrPeriodNotFound
}

func (m *mockPayrollRepository) ListPeriods(ctx context.Context, filter ListPeriodsFilter) ([]PayrollPeriod, int, error) {
	result := []PayrollPeriod{}
	for _, p := range m.periods {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		result = append(result, *p)
	}
	return result, len(result), nil
}

func (m *mockPayrollRepository) UpdatePeriodTotalsAndStatus(ctx context.Context, periodID string, status string, opsCount int, gross, meal, adj, net float64) error {
	p, ok := m.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	p.Status = status
	p.TotalOperatorsCount = opsCount
	p.TotalGrossPieceRate = gross
	p.TotalMealAllowance = meal
	p.TotalAdjustments = adj
	p.TotalNetPayable = net
	return nil
}

func (m *mockPayrollRepository) GetEligibleOperatorOutputs(ctx context.Context, startDate, endDate string, operatorIDs []string) ([]OperatorOutputAggregation, error) {
	return m.outputs, nil
}

func (m *mockPayrollRepository) SaveCalculationResults(ctx context.Context, periodID string, slips []PayrollSlip, itemsMap map[string][]PayrollSlipItem, totalOps int, totalGross, totalMeal, totalAdj, totalNet float64) error {
	if m.calculateErr != nil {
		return m.calculateErr
	}
	p, ok := m.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	if p.Status == string(PeriodStatusLocked) {
		return ErrPeriodAlreadyLocked
	}
	p.Status = string(PeriodStatusCalculated)
	p.TotalOperatorsCount = totalOps
	p.TotalGrossPieceRate = totalGross
	p.TotalMealAllowance = totalMeal
	p.TotalAdjustments = totalAdj
	p.TotalNetPayable = totalNet

	for _, s := range slips {
		sCopy := s
		sCopy.ID = "slip-" + s.OperatorID
		m.slips[sCopy.ID] = &sCopy
		m.slipDetails[sCopy.ID] = &PayrollSlipDetail{
			PayrollSlip:             sCopy,
			MealAllowancePercentage: p.MealAllowancePercentage,
			Items:                   itemsMap[s.OperatorID],
		}
	}
	return nil
}

func (m *mockPayrollRepository) ListSlipsByPeriod(ctx context.Context, periodID string, filter ListSlipsFilter) ([]PayrollSlip, int, error) {
	result := []PayrollSlip{}
	for _, s := range m.slips {
		if s.PayrollPeriodID == periodID {
			result = append(result, *s)
		}
	}
	return result, len(result), nil
}

func (m *mockPayrollRepository) FindSlipByID(ctx context.Context, slipID string) (*PayrollSlipDetail, error) {
	if s, ok := m.slipDetails[slipID]; ok {
		return s, nil
	}
	return nil, ErrSlipNotFound
}

func (m *mockPayrollRepository) LockPeriod(ctx context.Context, periodID, lockedByUserID string, notes *string) (*PayrollPeriod, error) {
	p, ok := m.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if p.Status == string(PeriodStatusLocked) {
		return nil, ErrPeriodAlreadyLocked
	}
	if p.Status != string(PeriodStatusCalculated) {
		return nil, ErrPeriodMustBeCalculatedToLock
	}
	p.Status = string(PeriodStatusLocked)
	now := time.Now()
	p.LockedAt = &now
	p.LockedByUserID = &lockedByUserID
	if notes != nil {
		p.Notes = notes
	}
	for _, slip := range m.slips {
		if slip.PayrollPeriodID == periodID {
			slip.Status = string(SlipStatusLocked)
		}
	}
	return p, nil
}

func (m *mockPayrollRepository) CreateAdjustment(ctx context.Context, periodID string, adj *PayrollAdjustment) error {
	p, ok := m.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	adj.ID = "adj-001"
	adj.CreatedAt = time.Now()
	m.adjustments[periodID] = append(m.adjustments[periodID], *adj)

	for _, s := range m.slips {
		if s.PayrollPeriodID == periodID && s.OperatorID == adj.OperatorID {
			s.AdjustmentsTotal += adj.Amount
			s.NetPayableTotal += adj.Amount
		}
	}
	p.TotalAdjustments += adj.Amount
	p.TotalNetPayable += adj.Amount
	return nil
}

func (m *mockPayrollRepository) ListAdjustmentsByPeriod(ctx context.Context, periodID string) ([]PayrollAdjustment, error) {
	if _, ok := m.periods[periodID]; !ok {
		return nil, ErrPeriodNotFound
	}
	adjs, ok := m.adjustments[periodID]
	if !ok {
		return []PayrollAdjustment{}, nil
	}
	return adjs, nil
}

func (m *mockPayrollRepository) GetPayrollSummaryReport(ctx context.Context, periodID string) (*PayrollSummaryReport, error) {
	p, ok := m.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	var lineSummaries []LinePayrollSummary
	var slips []PayrollSlip
	for _, s := range m.slips {
		if s.PayrollPeriodID == periodID {
			slips = append(slips, *s)
		}
	}
	return &PayrollSummaryReport{
		PeriodID:                p.ID,
		PeriodCode:              p.PeriodCode,
		StartDate:               p.StartDate,
		EndDate:                 p.EndDate,
		MealAllowancePercentage: p.MealAllowancePercentage,
		Status:                  p.Status,
		TotalOperators:          p.TotalOperatorsCount,
		TotalGrossPieceRate:     p.TotalGrossPieceRate,
		TotalMealAllowance:      p.TotalMealAllowance,
		TotalAdjustments:        p.TotalAdjustments,
		GrandTotalPayable:       p.TotalNetPayable,
		LinesSummary:            lineSummaries,
		OperatorsSummary:        slips,
	}, nil
}

func setupTestRouter(repo Repository, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := NewService(repo)
	handler := NewHandler(svc)

	authnMock := func(c *gin.Context) {
		c.Set("user_id", "user-mock-123")
		c.Set("role", role)
		c.Next()
	}

	api := r.Group("/api/v1")
	RegisterRoutes(api, handler, authnMock)
	return r
}

func TestCreatePeriod_Success_ByOwner(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "owner_executive")

	body := `{"periodCode":"PAY-2026-W42","startDate":"2026-10-12","endDate":"2026-10-18","mealAllowancePercentage":12.5}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePeriod_Success_ByFinance(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "finance")

	body := `{"startDate":"2026-10-12","endDate":"2026-10-18","mealAllowancePercentage":10.0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePeriod_Forbidden_ByProductionHead(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "production_head")

	body := `{"startDate":"2026-10-12","endDate":"2026-10-18","mealAllowancePercentage":12.0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePeriod_MealAllowanceOutOfRange(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "owner_executive")

	// 9.5% is below 10%
	body := `{"startDate":"2026-10-12","endDate":"2026-10-18","mealAllowancePercentage":9.5}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePeriod_InvalidDates(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "owner_executive")

	// Start date after end date
	body := `{"startDate":"2026-10-19","endDate":"2026-10-12","mealAllowancePercentage":12.0}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGetPeriod_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:                      "p-1",
		PeriodCode:              "PAY-2026-W42",
		StartDate:               "2026-10-12",
		EndDate:                 "2026-10-18",
		MealAllowancePercentage: 12.0,
		Status:                  "DRAFT",
	}
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/p-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestGetPeriod_NotFound(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/non-existent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestListPeriods_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1", PeriodCode: "PAY-1", Status: "DRAFT"}
	repo.periods["p-2"] = &PayrollPeriod{ID: "p-2", PeriodCode: "PAY-2", Status: "CALCULATED"}
	r := setupTestRouter(repo, "owner_executive")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestCalculatePayroll_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:                      "p-1",
		PeriodCode:              "PAY-2026-W42",
		StartDate:               "2026-10-12",
		EndDate:                 "2026-10-18",
		MealAllowancePercentage: 10.0,
		Status:                  "DRAFT",
	}

	// 2 outputs for operator 1
	repo.outputs = []OperatorOutputAggregation{
		{
			OperatorID:         "op-1",
			OperatorCode:       "OP-01",
			OperatorName:       "Siti Rahma",
			OperatorJoinDate:   "2026-10-14",
			OrderID:            "order-1",
			OrderCode:          "PO-1",
			ClientName:         "Client A",
			ComponentID:        "comp-1",
			ComponentName:      "Saku",
			WorkDate:           "2026-10-14",
			Quantity:           100,
			SnapshotSewingRate: 2000.0,
			Subtotal:           200000.0,
		},
		{
			OperatorID:         "op-1",
			OperatorCode:       "OP-01",
			OperatorName:       "Siti Rahma",
			OperatorJoinDate:   "2026-10-14",
			OrderID:            "order-1",
			OrderCode:          "PO-1",
			ClientName:         "Client A",
			ComponentID:        "comp-1",
			ComponentName:      "Saku",
			WorkDate:           "2026-10-15",
			Quantity:           50,
			SnapshotSewingRate: 2000.0,
			Subtotal:           100000.0,
		},
	}

	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/calculate", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify calculated results in mock
	p := repo.periods["p-1"]
	if p.Status != string(PeriodStatusCalculated) {
		t.Errorf("expected status CALCULATED, got %s", p.Status)
	}
	if p.TotalGrossPieceRate != 300000.0 {
		t.Errorf("expected TotalGross 300000.0, got %f", p.TotalGrossPieceRate)
	}
	// 10% of 300,000 = 30,000
	if p.TotalMealAllowance != 30000.0 {
		t.Errorf("expected TotalMealAllowance 30000.0, got %f", p.TotalMealAllowance)
	}
	if p.TotalNetPayable != 330000.0 {
		t.Errorf("expected TotalNetPayable 330000.0, got %f", p.TotalNetPayable)
	}
}

func TestCalculatePayroll_Forbidden_Operator(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1", Status: "DRAFT"}
	r := setupTestRouter(repo, "tailor_operator")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/calculate", bytes.NewBufferString(`{}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}

func TestCalculatePayroll_LockedPeriodConflict(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:     "p-1",
		Status: string(PeriodStatusLocked),
	}
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/calculate", bytes.NewBufferString(`{}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListSlips_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1", Status: "CALCULATED"}
	repo.slips["slip-1"] = &PayrollSlip{
		ID:                  "slip-1",
		PayrollPeriodID:     "p-1",
		OperatorID:          "op-1",
		OperatorName:        "Siti",
		GrossPieceRateTotal: 500000.0,
	}
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/p-1/slips", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestGetSlipDetail_Success(t *testing.T) {
	repo := newMockRepository()
	repo.slipDetails["slip-1"] = &PayrollSlipDetail{
		PayrollSlip: PayrollSlip{
			ID:                  "slip-1",
			PayrollPeriodID:     "p-1",
			OperatorID:          "op-1",
			OperatorName:        "Siti",
			GrossPieceRateTotal: 500000.0,
			MealAllowanceAmount: 62500.0,
			NetPayableTotal:     562500.0,
		},
		MealAllowancePercentage: 12.5,
		Items: []PayrollSlipItem{
			{
				ComponentName:      "Lengan",
				Quantity:           100,
				SnapshotSewingRate: 5000.0,
				Subtotal:           500000.0,
			},
		},
	}
	r := setupTestRouter(repo, "owner_executive")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/slips/slip-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    PayrollSlipDetail `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.OperatorName != "Siti" {
		t.Errorf("expected operator Siti, got %s", resp.Data.OperatorName)
	}
	if len(resp.Data.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Data.Items))
	}
}

func TestGetSlipDetail_NotFound(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/slips/non-existent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestLockPeriod_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:         "p-1",
		PeriodCode: "PAY-2026-W42",
		Status:     string(PeriodStatusCalculated),
	}
	repo.slips["slip-1"] = &PayrollSlip{
		ID:              "slip-1",
		PayrollPeriodID: "p-1",
		OperatorID:      "op-1",
		Status:          string(SlipStatusCalculated),
	}
	r := setupTestRouter(repo, "owner_executive")

	body := `{"notes":"Approved and finalized by Owner"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/lock", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool          `json:"success"`
		Message string        `json:"message"`
		Data    PayrollPeriod `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Status != string(PeriodStatusLocked) {
		t.Errorf("expected status LOCKED, got %s", resp.Data.Status)
	}
	if resp.Data.Notes == nil || *resp.Data.Notes != "Approved and finalized by Owner" {
		t.Errorf("expected notes to be saved, got %v", resp.Data.Notes)
	}
	if repo.slips["slip-1"].Status != string(SlipStatusLocked) {
		t.Errorf("expected slip status LOCKED, got %s", repo.slips["slip-1"].Status)
	}
}

func TestLockPeriod_DraftPeriod_Rejected(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:         "p-1",
		PeriodCode: "PAY-2026-W42",
		Status:     string(PeriodStatusDraft),
	}
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/lock", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for locking DRAFT period, got %d", w.Code)
	}
}

func TestLockPeriod_AlreadyLocked_Conflict(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:         "p-1",
		PeriodCode: "PAY-2026-W42",
		Status:     string(PeriodStatusLocked),
	}
	r := setupTestRouter(repo, "owner_executive")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/lock", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for already locked period, got %d", w.Code)
	}
}

func TestLockPeriod_UnauthorizedRole(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:         "p-1",
		PeriodCode: "PAY-2026-W42",
		Status:     string(PeriodStatusCalculated),
	}
	r := setupTestRouter(repo, "tailor_operator")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/lock", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}

func TestCreateAdjustment_Bonus_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:              "p-1",
		PeriodCode:      "PAY-2026-W42",
		Status:          string(PeriodStatusCalculated),
		TotalNetPayable: 500000.0,
	}
	repo.slips["slip-1"] = &PayrollSlip{
		ID:                  "slip-1",
		PayrollPeriodID:     "p-1",
		OperatorID:          "op-1",
		GrossPieceRateTotal: 500000.0,
		NetPayableTotal:     500000.0,
	}
	r := setupTestRouter(repo, "finance")

	body := `{"operatorId":"op-1","adjustmentType":"BONUS","amount":50000,"reason":"Bonus lembur darurat shift malam"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    PayrollAdjustment `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Amount != 50000 {
		t.Errorf("expected amount 50000, got %f", resp.Data.Amount)
	}
	if resp.Data.AdjustmentType != "BONUS" {
		t.Errorf("expected type BONUS, got %s", resp.Data.AdjustmentType)
	}

	// Verify slip and period totals updated
	slip := repo.slips["slip-1"]
	if slip.AdjustmentsTotal != 50000 {
		t.Errorf("expected slip adjustment 50000, got %f", slip.AdjustmentsTotal)
	}
	if slip.NetPayableTotal != 550000 {
		t.Errorf("expected slip net payable 550000, got %f", slip.NetPayableTotal)
	}
	if repo.periods["p-1"].TotalNetPayable != 550000 {
		t.Errorf("expected period net payable 550000, got %f", repo.periods["p-1"].TotalNetPayable)
	}
}

func TestCreateAdjustment_Deduction_NegativeAmount(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:              "p-1",
		PeriodCode:      "PAY-2026-W42",
		Status:          string(PeriodStatusCalculated),
		TotalNetPayable: 500000.0,
	}
	repo.slips["slip-1"] = &PayrollSlip{
		ID:                  "slip-1",
		PayrollPeriodID:     "p-1",
		OperatorID:          "op-1",
		GrossPieceRateTotal: 500000.0,
		NetPayableTotal:     500000.0,
	}
	r := setupTestRouter(repo, "finance")

	body := `{"operatorId":"op-1","adjustmentType":"DEDUCTION","amount":25000,"reason":"Koreksi potongan kasbon"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    PayrollAdjustment `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Amount != -25000 {
		t.Errorf("expected deduction amount -25000, got %f", resp.Data.Amount)
	}
	if repo.slips["slip-1"].NetPayableTotal != 475000 {
		t.Errorf("expected slip net payable 475000, got %f", repo.slips["slip-1"].NetPayableTotal)
	}
}

func TestCreateAdjustment_InvalidType_Rejected(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1"}
	r := setupTestRouter(repo, "owner_executive")

	body := `{"operatorId":"op-1","adjustmentType":"INVALID_TYPE","amount":50000,"reason":"Testing"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestCreateAdjustment_MissingReason_Rejected(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1"}
	r := setupTestRouter(repo, "finance")

	body := `{"operatorId":"op-1","adjustmentType":"BONUS","amount":50000,"reason":"   "}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing reason, got %d", w.Code)
	}
}

func TestCreateAdjustment_ZeroAmount_Rejected(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1"}
	r := setupTestRouter(repo, "finance")

	body := `{"operatorId":"op-1","adjustmentType":"BONUS","amount":0,"reason":"Zero test"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/payroll/periods/p-1/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for zero amount, got %d", w.Code)
	}
}

func TestListAdjustments_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{ID: "p-1"}
	repo.adjustments["p-1"] = []PayrollAdjustment{
		{ID: "adj-1", AdjustmentType: "BONUS", Amount: 50000, Reason: "Bonus"},
	}
	r := setupTestRouter(repo, "finance")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/p-1/adjustments", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool                `json:"success"`
		Data    []PayrollAdjustment `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 adjustment, got %d", len(resp.Data))
	}
}

func TestGetPayrollSummary_Success(t *testing.T) {
	repo := newMockRepository()
	repo.periods["p-1"] = &PayrollPeriod{
		ID:                  "p-1",
		PeriodCode:          "PAY-2026-W42",
		Status:              string(PeriodStatusCalculated),
		TotalGrossPieceRate: 500000,
		TotalMealAllowance:  62500,
		TotalNetPayable:     562500,
	}
	repo.slips["slip-1"] = &PayrollSlip{
		ID:                  "slip-1",
		PayrollPeriodID:     "p-1",
		OperatorName:        "Siti",
		GrossPieceRateTotal: 500000,
		MealAllowanceAmount: 62500,
		NetPayableTotal:     562500,
	}
	r := setupTestRouter(repo, "owner_executive")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/payroll/periods/p-1/summary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    PayrollSummaryReport `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.PeriodCode != "PAY-2026-W42" {
		t.Errorf("expected period code PAY-2026-W42, got %s", resp.Data.PeriodCode)
	}
	if len(resp.Data.OperatorsSummary) != 1 {
		t.Errorf("expected 1 slip in summary, got %d", len(resp.Data.OperatorsSummary))
	}
}


