package qcbonus

import (
	"context"
	"strings"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Service provides business logic and RBAC guards for QC pool bonus (BPMN 08).
type Service struct {
	repo Repository
}

// NewService creates a new qcbonus Service instance.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetBonusConfig retrieves current bonus rate configuration.
func (s *Service) GetBonusConfig(ctx context.Context, actor *authn.UserIdentity) (*QCBonusConfig, error) {
	if actor == nil {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.GetBonusConfig(ctx)
}

// UpdateBonusRate allows only OWNER to set or modify rate per pcs.
func (s *Service) UpdateBonusRate(ctx context.Context, actor *authn.UserIdentity, req UpdateQCBonusRateRequest) (*QCBonusConfig, error) {
	if actor == nil || strings.ToLower(actor.RoleCode) != "owner_executive" {
		return nil, ErrForbiddenOwnerOnly
	}
	if req.RatePerPcs < 0 {
		return nil, ErrInvalidBonusRate
	}
	return s.repo.UpdateBonusRate(ctx, req.RatePerPcs, actor.ID, req.Notes)
}

// ListRecipients retrieves the active list of bonus pool recipients.
func (s *Service) ListRecipients(ctx context.Context, actor *authn.UserIdentity) ([]QCBonusRecipient, error) {
	if actor == nil {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.ListRecipients(ctx)
}

// UpdateRecipients allows only OWNER to assign bonus recipients.
func (s *Service) UpdateRecipients(ctx context.Context, actor *authn.UserIdentity, req UpdateQCBonusRecipientsRequest) ([]QCBonusRecipient, error) {
	if actor == nil || strings.ToLower(actor.RoleCode) != "owner_executive" {
		return nil, ErrForbiddenOwnerOnly
	}
	if len(req.RecipientUserIDs) == 0 {
		return nil, ErrNoActiveRecipients
	}
	return s.repo.UpdateRecipients(ctx, req.RecipientUserIDs)
}

// ListPeriods retrieves period list.
func (s *Service) ListPeriods(ctx context.Context, actor *authn.UserIdentity, status string) ([]QCBonusPeriodSummary, error) {
	if actor == nil {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.ListPeriods(ctx, status)
}

// CreatePeriod opens a new calculation period.
func (s *Service) CreatePeriod(ctx context.Context, actor *authn.UserIdentity, req CreateQCBonusPeriodRequest) (*QCBonusPeriodDetail, error) {
	if actor == nil || !isQCManagerOrOwner(actor.RoleCode) {
		return nil, ErrForbiddenQualityBonus
	}
	if req.EndDate < req.StartDate {
		return nil, ErrPeriodDatesInvalid
	}
	return s.repo.CreatePeriod(ctx, req)
}

// GetPeriodByID retrieves period details with recipient distributions.
func (s *Service) GetPeriodByID(ctx context.Context, actor *authn.UserIdentity, id string) (*QCBonusPeriodDetail, error) {
	if actor == nil {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.GetPeriodByID(ctx, id)
}

// CalculatePeriod triggers pool bonus accumulation & equal split (Skenario Uji 3).
func (s *Service) CalculatePeriod(ctx context.Context, actor *authn.UserIdentity, id string) (*QCBonusPeriodDetail, error) {
	if actor == nil || !isQCManagerOrOwner(actor.RoleCode) {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.CalculatePeriod(ctx, id)
}

// LockPeriod finalizes the bonus period so it cannot be altered.
func (s *Service) LockPeriod(ctx context.Context, actor *authn.UserIdentity, id string, req LockQCBonusPeriodRequest) (*QCBonusPeriodDetail, error) {
	if actor == nil || !isFinanceOrOwner(actor.RoleCode) {
		return nil, ErrForbiddenQualityBonus
	}
	return s.repo.LockPeriod(ctx, id, actor.ID, req.Notes)
}

func isQCManagerOrOwner(role string) bool {
	switch strings.ToLower(role) {
	case "owner_executive", "production_head", "finance_payroll":
		return true
	default:
		return false
	}
}

func isFinanceOrOwner(role string) bool {
	switch strings.ToLower(role) {
	case "owner_executive", "finance_payroll":
		return true
	default:
		return false
	}
}

