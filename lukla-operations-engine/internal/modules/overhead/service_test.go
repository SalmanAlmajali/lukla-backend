package overhead

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRepo implementasi mock in-memory dari Repository untuk unit testing service.
type mockRepo struct {
	categories  map[string]OverheadCategory
	periods     map[string]OverheadPeriodDetail
	expenses    map[string][]OverheadExpenseItem
	totalUnits  int
	garmentUnit int
	bagUnit     int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		categories:  make(map[string]OverheadCategory),
		periods:     make(map[string]OverheadPeriodDetail),
		expenses:    make(map[string][]OverheadExpenseItem),
		totalUnits:  5000,
		garmentUnit: 4000,
		bagUnit:     1000,
	}
}

func (m *mockRepo) ListCategories(ctx context.Context) ([]OverheadCategory, error) {
	list := make([]OverheadCategory, 0)
	for _, c := range m.categories {
		list = append(list, c)
	}
	return list, nil
}

func (m *mockRepo) CreateCategory(ctx context.Context, cat *OverheadCategory) (*OverheadCategory, error) {
	for _, existing := range m.categories {
		if existing.Code == cat.Code {
			return nil, ErrCategoryCodeExists
		}
	}
	cat.ID = "cat-" + cat.Code
	cat.IsActive = true
	cat.CreatedAt = time.Now()
	cat.UpdatedAt = time.Now()
	m.categories[cat.ID] = *cat
	return cat, nil
}

func (m *mockRepo) GetCategoryByID(ctx context.Context, id string) (*OverheadCategory, error) {
	cat, ok := m.categories[id]
	if !ok {
		return nil, ErrCategoryNotFound
	}
	return &cat, nil
}

func (m *mockRepo) ListPeriods(ctx context.Context, year *int, status *string) ([]OverheadPeriodSummary, error) {
	list := make([]OverheadPeriodSummary, 0)
	for _, p := range m.periods {
		list = append(list, OverheadPeriodSummary{
			ID:                 p.ID,
			PeriodCode:         p.PeriodCode,
			PeriodMonth:        p.PeriodMonth,
			PeriodYear:         p.PeriodYear,
			Status:             p.Status,
			TotalExpenseAmount: p.TotalExpenseAmount,
		})
	}
	return list, nil
}

func (m *mockRepo) CreatePeriod(ctx context.Context, period *OverheadPeriodDetail) (*OverheadPeriodDetail, error) {
	for _, existing := range m.periods {
		if existing.PeriodYear == period.PeriodYear && existing.PeriodMonth == period.PeriodMonth {
			return nil, ErrPeriodAlreadyExists
		}
	}
	period.ID = "period-uuid-1"
	period.PeriodCode = "OH-202610"
	period.Status = StatusOpen
	period.StartDate = "2026-10-01"
	period.EndDate = "2026-10-31"
	period.Currency = "IDR"
	m.periods[period.ID] = *period
	return period, nil
}

func (m *mockRepo) GetPeriodByID(ctx context.Context, id string) (*OverheadPeriodDetail, error) {
	p, ok := m.periods[id]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	p.Expenses = m.expenses[id]
	return &p, nil
}

func (m *mockRepo) RecordExpense(ctx context.Context, expense *OverheadExpenseItem) (*OverheadExpenseItem, error) {
	p, ok := m.periods[expense.OverheadPeriodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if p.Status == StatusLocked {
		return nil, ErrPeriodAlreadyLocked
	}
	expense.ID = "exp-1"
	expense.CreatedAt = time.Now()
	expense.UpdatedAt = time.Now()
	m.expenses[expense.OverheadPeriodID] = append(m.expenses[expense.OverheadPeriodID], *expense)
	p.TotalExpenseAmount += expense.Amount
	m.periods[expense.OverheadPeriodID] = p
	return expense, nil
}

func (m *mockRepo) DeleteExpense(ctx context.Context, periodID string, expenseID string) error {
	p, ok := m.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	if p.Status == StatusLocked {
		return ErrPeriodAlreadyLocked
	}
	items := m.expenses[periodID]
	newItems := make([]OverheadExpenseItem, 0)
	var found bool
	for _, it := range items {
		if it.ID == expenseID {
			found = true
			p.TotalExpenseAmount -= it.Amount
		} else {
			newItems = append(newItems, it)
		}
	}
	if !found {
		return ErrExpenseNotFound
	}
	m.expenses[periodID] = newItems
	m.periods[periodID] = p
	return nil
}

func (m *mockRepo) GetProductionVolumeByDateRange(ctx context.Context, startDate, endDate string) (int, int, int, error) {
	return m.totalUnits, m.garmentUnit, m.bagUnit, nil
}

func (m *mockRepo) UpdateCalculation(ctx context.Context, periodID string, res *OverheadCalculationResult, weightGarment, weightBag float64, notes *string) error {
	p, ok := m.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	if p.Status == StatusLocked {
		return ErrPeriodAlreadyLocked
	}
	p.TotalProducedUnits = res.TotalProducedUnits
	p.GarmentUnits = res.GarmentUnits
	p.BagUnits = res.BagUnits
	p.WeightGarment = weightGarment
	p.WeightBag = weightBag
	p.FlatRatePerUnit = res.FlatRatePerUnit
	p.GarmentRatePerUnit = res.GarmentRatePerUnit
	p.BagRatePerUnit = res.BagRatePerUnit
	p.Status = StatusCalculated
	m.periods[periodID] = p
	return nil
}

func (m *mockRepo) LockPeriod(ctx context.Context, periodID string, lockedBy string) (*OverheadPeriodDetail, error) {
	p, ok := m.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if p.Status == StatusLocked {
		return nil, ErrPeriodAlreadyLocked
	}
	p.Status = StatusLocked
	now := time.Now()
	p.LockedAt = &now
	p.LockedBy = &lockedBy
	m.periods[periodID] = p
	return &p, nil
}

func (m *mockRepo) GetPeriodSummaryReport(ctx context.Context, periodID string) (*OverheadPeriodSummaryReport, error) {
	p, err := m.GetPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	return &OverheadPeriodSummaryReport{
		PeriodCode:         p.PeriodCode,
		PeriodMonth:        p.PeriodMonth,
		PeriodYear:         p.PeriodYear,
		Status:             p.Status,
		TotalExpenseAmount: p.TotalExpenseAmount,
		ProductionBreakdown: ProductionBreakdown{
			TotalUnits:   p.TotalProducedUnits,
			GarmentUnits: p.GarmentUnits,
			BagUnits:     p.BagUnits,
		},
		Rates: RatesBreakdown{
			FlatRatePerUnit:    p.FlatRatePerUnit,
			GarmentRatePerUnit: p.GarmentRatePerUnit,
			BagRatePerUnit:     p.BagRatePerUnit,
		},
	}, nil
}

func (m *mockRepo) GetOrderCOGSReport(ctx context.Context, orderID string) (*OrderCOGSReport, error) {
	if orderID != "order-123" {
		return nil, ErrOrderNotFound
	}
	return &OrderCOGSReport{
		OrderID:              orderID,
		OrderNumber:          "PO-SNO-2026-M9370",
		ProducedQuantity:     2000,
		ContractPricePerUnit: 45000,
		TotalRevenue:         90000000,
		CostComponents: COGSComponents{
			DirectLaborPerUnit: 22500,
			TotalDirectLabor:   45000000,
			QCBonusPerUnit:     500,
			TotalQCBonus:       1000000,
			OverheadPerUnit:    4761.90,
			TotalOverhead:      9523800,
		},
		TotalCostPerUnit:   27761.90,
		TotalCOGS:          55523800,
		GrossMarginPerUnit: 17238.10,
		TotalGrossMargin:   34476200,
		GrossMarginPct:     38.31,
	}, nil
}

func TestService_OverheadLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	svc := NewService(repo)

	// 1. Create Category
	cat, err := svc.CreateCategory(ctx, CreateOverheadCategoryRequest{
		Code: "ELECTRICITY",
		Name: "Listrik PLN Pabrik",
	})
	if err != nil {
		t.Fatalf("unexpected error creating category: %v", err)
	}
	if cat.Code != "ELECTRICITY" {
		t.Fatalf("expected code ELECTRICITY, got %s", cat.Code)
	}

	// 2. Reject duplicate category code
	_, err = svc.CreateCategory(ctx, CreateOverheadCategoryRequest{
		Code: "ELECTRICITY",
		Name: "Listrik Duplikat",
	})
	if !errors.Is(err, ErrCategoryCodeExists) {
		t.Fatalf("expected ErrCategoryCodeExists, got %v", err)
	}

	// 3. Create Period
	period, err := svc.CreatePeriod(ctx, CreateOverheadPeriodRequest{
		PeriodMonth: 10,
		PeriodYear:  2026,
	})
	if err != nil {
		t.Fatalf("unexpected error creating period: %v", err)
	}
	if period.PeriodCode != "OH-202610" || period.Status != StatusOpen {
		t.Fatalf("unexpected period: %+v", period)
	}

	// 4. Record Expense
	exp1, err := svc.RecordExpense(ctx, period.ID, RecordOverheadExpenseRequest{
		CategoryID:  cat.ID,
		ExpenseDate: "2026-10-15",
		Amount:      15000000, // Rp 15.000.000
	})
	if err != nil {
		t.Fatalf("unexpected error recording expense: %v", err)
	}
	if exp1.Amount != 15000000 {
		t.Fatalf("expected 15000000, got %.2f", exp1.Amount)
	}

	exp2, err := svc.RecordExpense(ctx, period.ID, RecordOverheadExpenseRequest{
		CategoryID:  cat.ID,
		ExpenseDate: "2026-10-20",
		Amount:      10000000, // Rp 10.000.000 (total Rp 25.000.000)
	})
	if err != nil {
		t.Fatalf("unexpected error recording expense 2: %v", err)
	}
	if exp2.Amount != 10000000 {
		t.Fatalf("expected 10000000, got %.2f", exp2.Amount)
	}

	// 5. Calculate Overhead
	// Total expense: Rp 25.000.000
	// Total volume: 5.000 pcs (4.000 Garment, 1.000 Bag)
	// Bobot: Garment 1.0, Bag 1.25
	wGarment := 1.00
	wBag := 1.25
	calc, err := svc.CalculateOverhead(ctx, period.ID, CalculateOverheadRequest{
		WeightGarment: &wGarment,
		WeightBag:     &wBag,
	})
	if err != nil {
		t.Fatalf("unexpected error calculating overhead: %v", err)
	}

	// Flat Rate: 25.000.000 / 5.000 = Rp 5.000
	if calc.FlatRatePerUnit != 5000.00 {
		t.Fatalf("expected flat rate 5000.00, got %.2f", calc.FlatRatePerUnit)
	}

	// Weighted units: (4000 * 1.0) + (1000 * 1.25) = 5250
	// Garment rate: (25.000.000 / 5250) * 1.0 = ~4761.90
	if calc.GarmentRatePerUnit < 4761.89 || calc.GarmentRatePerUnit > 4761.91 {
		t.Fatalf("expected garment rate ~4761.90, got %.2f", calc.GarmentRatePerUnit)
	}

	// Bag rate: (25.000.000 / 5250) * 1.25 = ~5952.38
	if calc.BagRatePerUnit < 5952.37 || calc.BagRatePerUnit > 5952.39 {
		t.Fatalf("expected bag rate ~5952.38, got %.2f", calc.BagRatePerUnit)
	}

	// 6. Lock Period
	locked, err := svc.LockPeriod(ctx, period.ID, "user-finance-1")
	if err != nil {
		t.Fatalf("unexpected error locking period: %v", err)
	}
	if locked.Status != StatusLocked {
		t.Fatalf("expected status LOCKED, got %s", locked.Status)
	}

	// 7. Prevent modification when locked
	_, err = svc.RecordExpense(ctx, period.ID, RecordOverheadExpenseRequest{
		CategoryID:  cat.ID,
		ExpenseDate: "2026-10-30",
		Amount:      500000,
	})
	if !errors.Is(err, ErrPeriodAlreadyLocked) {
		t.Fatalf("expected ErrPeriodAlreadyLocked when recording expense on locked period, got %v", err)
	}

	_, err = svc.CalculateOverhead(ctx, period.ID, CalculateOverheadRequest{})
	if !errors.Is(err, ErrPeriodAlreadyLocked) {
		t.Fatalf("expected ErrPeriodAlreadyLocked when calculating on locked period, got %v", err)
	}

	// 8. Order COGS Report (BPMN 01)
	cogs, err := svc.GetOrderCOGSReport(ctx, "order-123")
	if err != nil {
		t.Fatalf("unexpected error getting COGS report: %v", err)
	}
	if cogs.TotalRevenue != 90000000 || cogs.TotalCOGS != 55523800 {
		t.Fatalf("unexpected COGS values: %+v", cogs)
	}
	if cogs.GrossMarginPct < 38.0 || cogs.GrossMarginPct > 39.0 {
		t.Fatalf("expected gross margin pct ~38.31, got %.2f", cogs.GrossMarginPct)
	}
}

