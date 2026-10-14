package quality

import (
	"context"
	"testing"
	"time"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

type mockQualityRepo struct {
	categories  []DefectCategory
	queue       []QCQueueItem
	inspections map[string]*QCInspectionDetail
	summaries   []QCInspectionSummary
	orderSum    map[string]*QCOrderSummary
}

func newMockQualityRepo() *mockQualityRepo {
	return &mockQualityRepo{
		categories:  []DefectCategory{},
		queue:       []QCQueueItem{},
		inspections: make(map[string]*QCInspectionDetail),
		summaries:   []QCInspectionSummary{},
		orderSum:    make(map[string]*QCOrderSummary),
	}
}

func (m *mockQualityRepo) ListDefectCategories(ctx context.Context, isActive *bool) ([]DefectCategory, error) {
	if isActive == nil {
		return m.categories, nil
	}
	var res []DefectCategory
	for _, c := range m.categories {
		if c.IsActive == *isActive {
			res = append(res, c)
		}
	}
	return res, nil
}

func (m *mockQualityRepo) CreateDefectCategory(ctx context.Context, cat *DefectCategory) error {
	for _, c := range m.categories {
		if c.Code == cat.Code {
			return ErrDefectCategoryCodeExists
		}
	}
	cat.ID = "cat-mock-id"
	cat.CreatedAt = time.Now()
	cat.UpdatedAt = time.Now()
	m.categories = append(m.categories, *cat)
	return nil
}

func (m *mockQualityRepo) FindDefectCategoryByID(ctx context.Context, id string) (*DefectCategory, error) {
	for _, c := range m.categories {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, ErrDefectCategoryNotFound
}

func (m *mockQualityRepo) ListQCQueue(ctx context.Context, filter ListQCQueueFilter) ([]QCQueueItem, int, error) {
	return m.queue, len(m.queue), nil
}

func (m *mockQualityRepo) CreateInspection(ctx context.Context, req *CreateQCInspectionRequest, inspectorID string) (*QCInspectionDetail, error) {
	totalChecked := req.PassedQuantity + req.RepairQuantity + req.RejectQuantity
	detail := &QCInspectionDetail{
		ID:               "mock-insp-id",
		InspectionNumber: "QC-20261015-0001",
		AssemblyID:       req.AssemblyID,
		OrderID:          req.OrderID,
		OrderCode:        "PO-2026-10-001",
		ClientName:       "Snøhund Brand",
		ArticleName:      "Arctic Parka V1",
		InspectorID:      inspectorID,
		InspectorName:    "QC Inspector Test",
		InspectionDate:   req.InspectionDate,
		PassedQuantity:   req.PassedQuantity,
		RepairQuantity:   req.RepairQuantity,
		RejectQuantity:   req.RejectQuantity,
		TotalChecked:     totalChecked,
		Notes:            req.Notes,
		CreatedAt:        time.Now(),
		Defects:          []QCInspectionDefect{},
	}
	for _, d := range req.Defects {
		detail.Defects = append(detail.Defects, QCInspectionDefect{
			ID:                 "mock-defect-id",
			DefectCategoryCode: "DEF-SEW-01",
			DefectCategoryName: "Jahitan Lepas",
			Severity:           "MAJOR",
			ComponentID:        d.ComponentID,
			Quantity:           d.Quantity,
			Disposition:        d.Disposition,
			Notes:              d.Notes,
		})
	}
	m.inspections[detail.ID] = detail
	return detail, nil
}

func (m *mockQualityRepo) ListInspections(ctx context.Context, filter ListQCInspectionsFilter) ([]QCInspectionSummary, int, error) {
	return m.summaries, len(m.summaries), nil
}

func (m *mockQualityRepo) FindInspectionByID(ctx context.Context, id string) (*QCInspectionDetail, error) {
	detail, ok := m.inspections[id]
	if !ok {
		return nil, ErrInspectionNotFound
	}
	return detail, nil
}

func (m *mockQualityRepo) GetOrderQualitySummary(ctx context.Context, orderID string) (*QCOrderSummary, error) {
	s, ok := m.orderSum[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	return s, nil
}

func TestQualityService_CreateDefectCategory(t *testing.T) {
	repo := newMockQualityRepo()
	svc := NewService(repo)
	ctx := context.Background()

	qcActor := &authn.UserIdentity{
		ID:       "user-qc-1",
		RoleCode: "qc_inspector",
		IsActive: true,
	}
	tailorActor := &authn.UserIdentity{
		ID:       "user-tailor-1",
		RoleCode: "tailor",
		IsActive: true,
	}

	// 1. Forbidden for tailor
	_, err := svc.CreateDefectCategory(ctx, tailorActor, CreateDefectCategoryRequest{
		Code:     "DEF-TEST-01",
		Name:     "Test Defect",
		Severity: "MINOR",
	})
	if err != ErrForbiddenQuality {
		t.Fatalf("expected ErrForbiddenQuality for tailor, got %v", err)
	}

	// 2. Success for QC Inspector
	cat, err := svc.CreateDefectCategory(ctx, qcActor, CreateDefectCategoryRequest{
		Code:     "DEF-TEST-01",
		Name:     "Test Defect",
		Severity: "MINOR",
	})
	if err != nil {
		t.Fatalf("unexpected error creating defect category: %v", err)
	}
	if cat.Code != "DEF-TEST-01" {
		t.Fatalf("expected code DEF-TEST-01, got %s", cat.Code)
	}

	// 3. Duplicate code returns error
	_, err = svc.CreateDefectCategory(ctx, qcActor, CreateDefectCategoryRequest{
		Code:     "DEF-TEST-01",
		Name:     "Duplicate",
		Severity: "MAJOR",
	})
	if err != ErrDefectCategoryCodeExists {
		t.Fatalf("expected ErrDefectCategoryCodeExists, got %v", err)
	}
}

func TestQualityService_CreateInspection_Validation(t *testing.T) {
	repo := newMockQualityRepo()
	svc := NewService(repo)
	ctx := context.Background()

	qcActor := &authn.UserIdentity{
		ID:       "user-qc-1",
		RoleCode: "qc_inspector",
		IsActive: true,
	}

	// 1. Zero quantity checked
	_, err := svc.CreateInspection(ctx, qcActor, CreateQCInspectionRequest{
		AssemblyID:     "asm-1",
		OrderID:        "order-1",
		PassedQuantity: 0,
		RepairQuantity: 0,
		RejectQuantity: 0,
	})
	if err != ErrInvalidQuantity {
		t.Fatalf("expected ErrInvalidQuantity for 0 checked, got %v", err)
	}

	// 2. Repair > 0 but defects empty
	_, err = svc.CreateInspection(ctx, qcActor, CreateQCInspectionRequest{
		AssemblyID:     "asm-1",
		OrderID:        "order-1",
		PassedQuantity: 10,
		RepairQuantity: 2,
		RejectQuantity: 0,
		Defects:        []CreateQCInspectionDefect{},
	})
	if err != ErrDefectDetailsRequired {
		t.Fatalf("expected ErrDefectDetailsRequired, got %v", err)
	}

	// 3. Defect quantity mismatch (repair 2, but defect line has quantity 1)
	_, err = svc.CreateInspection(ctx, qcActor, CreateQCInspectionRequest{
		AssemblyID:     "asm-1",
		OrderID:        "order-1",
		PassedQuantity: 10,
		RepairQuantity: 2,
		RejectQuantity: 0,
		Defects: []CreateQCInspectionDefect{
			{
				DefectCategoryID: "cat-1",
				Quantity:         1,
				Disposition:      "REPAIR",
			},
		},
	})
	if err != ErrDefectQuantityMismatch {
		t.Fatalf("expected ErrDefectQuantityMismatch, got %v", err)
	}

	// 4. Valid inspection submission
	detail, err := svc.CreateInspection(ctx, qcActor, CreateQCInspectionRequest{
		AssemblyID:     "asm-1",
		OrderID:        "order-1",
		InspectionDate: "2026-10-15",
		PassedQuantity: 48,
		RepairQuantity: 2,
		RejectQuantity: 0,
		Defects: []CreateQCInspectionDefect{
			{
				DefectCategoryID: "cat-1",
				Quantity:         2,
				Disposition:      "REPAIR",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error creating valid inspection: %v", err)
	}
	if detail.TotalChecked != 50 {
		t.Fatalf("expected TotalChecked 50, got %d", detail.TotalChecked)
	}
	if detail.PassedQuantity != 48 || detail.RepairQuantity != 2 {
		t.Fatalf("unexpected counts: passed=%d, repair=%d", detail.PassedQuantity, detail.RepairQuantity)
	}
}

