package quality

import (
	"context"
	"strings"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Service provides quality control business logic and validation.
type Service struct {
	repo Repository
}

// NewService creates a new quality Service instance.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListDefectCategories retrieves defect categories.
func (s *Service) ListDefectCategories(ctx context.Context, actor *authn.UserIdentity, isActive *bool) ([]DefectCategory, error) {
	if actor == nil {
		return nil, ErrForbiddenQuality
	}
	return s.repo.ListDefectCategories(ctx, isActive)
}

// CreateDefectCategory creates a new defect category in master data.
func (s *Service) CreateDefectCategory(ctx context.Context, actor *authn.UserIdentity, req CreateDefectCategoryRequest) (*DefectCategory, error) {
	if actor == nil || !isQCAdminOrInspector(actor.RoleCode) {
		return nil, ErrForbiddenQuality
	}

	code := strings.TrimSpace(strings.ToUpper(req.Code))
	name := strings.TrimSpace(req.Name)
	severity := strings.TrimSpace(strings.ToUpper(req.Severity))

	cat := &DefectCategory{
		Code:        code,
		Name:        name,
		Severity:    severity,
		Description: req.Description,
		IsActive:    true,
	}

	if err := s.repo.CreateDefectCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

// ListQCQueue retrieves items ready for QC inspection.
func (s *Service) ListQCQueue(ctx context.Context, actor *authn.UserIdentity, filter ListQCQueueFilter) ([]QCQueueItem, int, error) {
	if actor == nil {
		return nil, 0, ErrForbiddenQuality
	}
	return s.repo.ListQCQueue(ctx, filter)
}

// CreateInspection submits inspection results and defect logs.
func (s *Service) CreateInspection(ctx context.Context, actor *authn.UserIdentity, req CreateQCInspectionRequest) (*QCInspectionDetail, error) {
	if actor == nil || !isQCAdminOrInspector(actor.RoleCode) {
		return nil, ErrForbiddenQuality
	}

	totalChecked := req.PassedQuantity + req.RepairQuantity + req.RejectQuantity
	if totalChecked <= 0 {
		return nil, ErrInvalidQuantity
	}

	// Validate defect breakdown if there are repairs or rejects
	totalDefectNeeded := req.RepairQuantity + req.RejectQuantity
	if totalDefectNeeded > 0 && len(req.Defects) == 0 {
		return nil, ErrDefectDetailsRequired
	}

	repairCount := 0
	rejectCount := 0
	for _, d := range req.Defects {
		switch strings.ToUpper(d.Disposition) {
		case "REPAIR":
			repairCount += d.Quantity
		case "REJECT":
			rejectCount += d.Quantity
		default:
			return nil, ErrInvalidQuantity
		}
	}

	if repairCount != req.RepairQuantity || rejectCount != req.RejectQuantity {
		return nil, ErrDefectQuantityMismatch
	}

	return s.repo.CreateInspection(ctx, &req, actor.ID)
}

// ListInspections retrieves inspection history.
func (s *Service) ListInspections(ctx context.Context, actor *authn.UserIdentity, filter ListQCInspectionsFilter) ([]QCInspectionSummary, int, error) {
	if actor == nil {
		return nil, 0, ErrForbiddenQuality
	}
	return s.repo.ListInspections(ctx, filter)
}

// GetInspectionDetail retrieves a single inspection by ID.
func (s *Service) GetInspectionDetail(ctx context.Context, actor *authn.UserIdentity, id string) (*QCInspectionDetail, error) {
	if actor == nil {
		return nil, ErrForbiddenQuality
	}
	return s.repo.FindInspectionByID(ctx, id)
}

// GetOrderQualitySummary retrieves quality metrics and defect ranking for an order.
func (s *Service) GetOrderQualitySummary(ctx context.Context, actor *authn.UserIdentity, orderID string) (*QCOrderSummary, error) {
	if actor == nil {
		return nil, ErrForbiddenQuality
	}
	return s.repo.GetOrderQualitySummary(ctx, orderID)
}

func isQCAdminOrInspector(role string) bool {
	switch strings.ToLower(role) {
	case "qc_inspector", "production_head", "owner_executive":
		return true
	default:
		return false
	}
}

