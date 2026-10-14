package production

import (
	"context"
	"strings"
	"time"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Service defines business logic for production line operations.
type Service struct {
	repo Repository
}

// NewService creates a new production domain service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListLines lists all production lines on the factory floor.
func (s *Service) ListLines(ctx context.Context, actor *authn.UserIdentity) ([]ProductionLine, error) {
	if actor == nil || actor.RoleCode == "client" {
		return nil, ErrForbiddenProduction
	}
	return s.repo.ListLines(ctx)
}

// GetLineDetail retrieves detail of a production line along with active allocations.
func (s *Service) GetLineDetail(ctx context.Context, actor *authn.UserIdentity, id string) (*ProductionLineDetail, error) {
	if actor == nil || actor.RoleCode == "client" {
		return nil, ErrForbiddenProduction
	}
	return s.repo.FindLineByID(ctx, id)
}

// CreateLine creates a new production line with machine capacity limit validation (~70 machines factory limit).
func (s *Service) CreateLine(ctx context.Context, actor *authn.UserIdentity, req CreateLineRequest) (*ProductionLine, error) {
	if !isAuthorizedLineManager(actor) {
		return nil, ErrForbiddenProduction
	}
	if req.MaxCapacity <= 0 {
		return nil, ErrLineCapacityExceeded
	}

	code := strings.TrimSpace(strings.ToUpper(req.Code))
	existing, err := s.repo.FindLineByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrLineCodeExists
	}

	// Validate factory capacity (~70 machine limit)
	currentTotal, err := s.repo.GetTotalFactoryCapacity(ctx, "")
	if err != nil {
		return nil, err
	}
	if currentTotal+req.MaxCapacity > 70 {
		return nil, ErrFactoryCapacityExceeded
	}

	line := &ProductionLine{
		Code:         code,
		Name:         strings.TrimSpace(req.Name),
		MaxCapacity:  req.MaxCapacity,
		SupervisorID: req.SupervisorID,
		Status:       "ACTIVE",
	}

	if err := s.repo.CreateLine(ctx, line); err != nil {
		return nil, err
	}
	return line, nil
}

// UpdateLine updates an existing line's details and capacity.
func (s *Service) UpdateLine(ctx context.Context, actor *authn.UserIdentity, id string, req UpdateLineRequest) (*ProductionLineDetail, error) {
	if !isAuthorizedLineManager(actor) {
		return nil, ErrForbiddenProduction
	}

	detail, err := s.repo.FindLineByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		detail.Name = strings.TrimSpace(*req.Name)
	}
	if req.SupervisorID != nil {
		detail.SupervisorID = req.SupervisorID
	}
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		st := strings.ToUpper(strings.TrimSpace(*req.Status))
		if st == "ACTIVE" || st == "INACTIVE" || st == "MAINTENANCE" {
			detail.Status = st
		}
	}
	if req.MaxCapacity != nil {
		if *req.MaxCapacity <= 0 {
			return nil, ErrLineCapacityExceeded
		}
		// Validate factory limit excluding this line
		currentTotal, err := s.repo.GetTotalFactoryCapacity(ctx, id)
		if err != nil {
			return nil, err
		}
		if currentTotal+*req.MaxCapacity > 70 {
			return nil, ErrFactoryCapacityExceeded
		}
		detail.MaxCapacity = *req.MaxCapacity
	}

	lineUpdate := &detail.ProductionLine
	if err := s.repo.UpdateLine(ctx, lineUpdate); err != nil {
		return nil, err
	}

	return s.repo.FindLineByID(ctx, id)
}

// ListAllocations lists operator allocations.
func (s *Service) ListAllocations(ctx context.Context, actor *authn.UserIdentity, filter ListAllocationsFilter) ([]LineAllocation, error) {
	if actor == nil || actor.RoleCode == "client" {
		return nil, ErrForbiddenProduction
	}
	return s.repo.ListAllocations(ctx, filter)
}

// CreateAllocation assigns an operator to a line and order.
func (s *Service) CreateAllocation(ctx context.Context, actor *authn.UserIdentity, req CreateAllocationRequest) (*LineAllocation, error) {
	if !isAuthorizedLineManager(actor) {
		return nil, ErrForbiddenProduction
	}

	// 1. Validate operator
	opName, opCode, skillTier, err := s.repo.ValidateOperator(ctx, req.OperatorID)
	if err != nil {
		return nil, err
	}

	// 2. Check if operator is already active
	activeAlloc, err := s.repo.FindActiveAllocationByOperator(ctx, req.OperatorID)
	if err != nil {
		return nil, err
	}
	if activeAlloc != nil {
		return nil, ErrOperatorAlreadyAllocated
	}

	// 3. Validate line
	lineDetail, err := s.repo.FindLineByID(ctx, req.LineID)
	if err != nil {
		return nil, err
	}
	if lineDetail.Status != "ACTIVE" {
		return nil, ErrLineNotFound
	}
	if lineDetail.ActiveOperatorsCount >= lineDetail.MaxCapacity {
		return nil, ErrLineCapacityExceeded
	}

	// 4. Validate order
	orderCode, _, _, err := s.repo.ValidateOrder(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	// 5. Validate component if specified
	var compName *string
	if req.AssignedComponentID != nil && *req.AssignedComponentID != "" {
		cn, err := s.repo.ValidateComponent(ctx, *req.AssignedComponentID)
		if err != nil {
			return nil, err
		}
		compName = &cn
	}

	assignedAt := time.Now()
	if req.AssignedAt != nil && *req.AssignedAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.AssignedAt); err == nil {
			assignedAt = t
		}
	}

	alloc := &LineAllocation{
		LineID:                req.LineID,
		LineCode:              lineDetail.Code,
		OrderID:               req.OrderID,
		OrderCode:             orderCode,
		OperatorID:            req.OperatorID,
		OperatorCode:          opCode,
		OperatorName:          opName,
		SkillTier:             skillTier,
		AssignedComponentID:   req.AssignedComponentID,
		AssignedComponentName: compName,
		Status:                "ACTIVE",
		AssignedAt:            assignedAt,
	}

	if err := s.repo.CreateAllocation(ctx, alloc); err != nil {
		return nil, err
	}
	return alloc, nil
}

// TransferOperator transfers an operator from one line to another (BPMN 04).
func (s *Service) TransferOperator(ctx context.Context, actor *authn.UserIdentity, allocationID string, req TransferOperatorRequest) (*TransferResult, error) {
	if !isAuthorizedLineManager(actor) {
		return nil, ErrForbiddenProduction
	}
	if strings.TrimSpace(req.TransferReason) == "" {
		return nil, ErrTransferReasonRequired
	}

	// 1. Find existing allocation
	currentAlloc, err := s.repo.FindAllocationByID(ctx, allocationID)
	if err != nil {
		return nil, err
	}
	if currentAlloc.Status != "ACTIVE" {
		return nil, ErrAllocationNotActive
	}

	// 2. Validate destination line
	if currentAlloc.LineID == req.DestinationLineID {
		return nil, ErrSameLineTransfer
	}
	destLine, err := s.repo.FindLineByID(ctx, req.DestinationLineID)
	if err != nil {
		return nil, ErrDestinationLineNotFound
	}
	if destLine.Status != "ACTIVE" {
		return nil, ErrDestinationLineNotFound
	}
	if destLine.ActiveOperatorsCount >= destLine.MaxCapacity {
		return nil, ErrLineCapacityExceeded
	}

	// 3. Optional new component assignment
	assignedCompID := currentAlloc.AssignedComponentID
	if req.AssignedComponentID != nil && *req.AssignedComponentID != "" {
		assignedCompID = req.AssignedComponentID
	}

	transferredAt := time.Now()
	if req.TransferredAt != nil && *req.TransferredAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.TransferredAt); err == nil {
			transferredAt = t
		}
	}

	newAlloc := &LineAllocation{
		LineID:              destLine.ID,
		OrderID:             currentAlloc.OrderID,
		OperatorID:          currentAlloc.OperatorID,
		AssignedComponentID: assignedCompID,
		Status:              "ACTIVE",
		AssignedAt:          transferredAt,
	}

	var actorUserID *string
	if actor != nil && actor.ID != "" {
		actorUserID = &actor.ID
	}

	transferLog := &AllocationTransferLog{
		OperatorID:          currentAlloc.OperatorID,
		FromLineID:          currentAlloc.LineID,
		ToLineID:            destLine.ID,
		Reason:              strings.TrimSpace(req.TransferReason),
		TransferredByUserID: actorUserID,
		TransferredAt:       transferredAt,
	}

	if err := s.repo.TransferOperator(ctx, currentAlloc.ID, newAlloc, transferLog); err != nil {
		return nil, err
	}

	return &TransferResult{
		OldAllocationID:     currentAlloc.ID,
		NewAllocationID:     newAlloc.ID,
		OperatorID:          currentAlloc.OperatorID,
		PreviousLineCode:    currentAlloc.LineCode,
		DestinationLineCode: destLine.Code,
		TransferredAt:       transferredAt,
	}, nil
}

// ListTransferHistories retrieves audit records of line transfers for an allocation.
func (s *Service) ListTransferHistories(ctx context.Context, actor *authn.UserIdentity, allocationID string) ([]AllocationTransferLog, error) {
	if actor == nil || actor.RoleCode == "client" {
		return nil, ErrForbiddenProduction
	}
	return s.repo.ListTransferHistoriesByAllocation(ctx, allocationID)
}

func isAuthorizedLineManager(actor *authn.UserIdentity) bool {
	if actor == nil {
		return false
	}
	return actor.RoleCode == "owner_executive" || actor.RoleCode == "production_head"
}

