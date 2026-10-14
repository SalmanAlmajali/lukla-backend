package ratecard

import (
	"context"
	"strings"
	"time"
)

// Service provides rate card domain operations.
type Service struct {
	repo Repository
}

// NewService instantiates a new rate card domain service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// canManageRateCards checks whether requester role has permission to create/update rate cards.
// Sesuai BPMN 10 & spesifikasi: hanya OWNER / EKSEKUTIF dan KEPALA PRODUKSI (atau ADMIN).
func canManageRateCards(role string) bool {
	norm := strings.ToLower(strings.TrimSpace(role))
	return norm == "owner_executive" || norm == "production_head" || norm == "admin" || norm == "owner" || norm == "kepala_produksi"
}

// ListRateCards retrieves rate cards matching client/article filters.
// Staf internal dapat melihat semua rate card atau memfilter per client/article.
// Klien portal hanya dapat melihat rate card milik perusahaannya sendiri.
func (s *Service) ListRateCards(ctx context.Context, filter ListRateCardsFilter, requesterRole string, requesterClientID *string) ([]RateCard, error) {
	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID == "" {
			return nil, ErrForbiddenRateCardAccess
		}
		if filter.ClientID != "" && filter.ClientID != *requesterClientID {
			return nil, ErrForbiddenRateCardAccess
		}
		filter.ClientID = *requesterClientID
	}

	return s.repo.ListRateCards(ctx, filter)
}

// GetRateCardDetail retrieves a single rate card by UUID.
func (s *Service) GetRateCardDetail(ctx context.Context, id string, requesterRole string, requesterClientID *string) (*RateCard, error) {
	rc, err := s.repo.FindRateCardByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, ErrRateCardNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != rc.ClientID {
			return nil, ErrForbiddenRateCardAccess
		}
	}

	return rc, nil
}

// CreateRateCard validates and stores a new rate card for an article component.
func (s *Service) CreateRateCard(ctx context.Context, req CreateRateCardRequest, requesterRole string, requesterUserID *string) (*RateCard, error) {
	if !canManageRateCards(requesterRole) {
		return nil, ErrForbiddenRateCardManage
	}

	if req.SewingRate < 0 || req.CuttingRate < 0 {
		return nil, ErrInvalidRateValue
	}

	// Validate client, article, component hierarchy
	compInfo, err := s.repo.ValidateComponentHierarchy(ctx, req.ClientID, req.ArticleID, req.ComponentID)
	if err != nil {
		return nil, err
	}

	// Check unique rate card for this component
	existing, err := s.repo.FindRateCardByComponent(ctx, req.ArticleID, req.ComponentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateRateCard
	}

	effectiveDate := time.Now().Format("2006-01-02")
	if req.EffectiveDate != nil && strings.TrimSpace(*req.EffectiveDate) != "" {
		effectiveDate = strings.TrimSpace(*req.EffectiveDate)
	}

	rc := RateCard{
		ClientID:      req.ClientID,
		ClientName:    compInfo.ClientName,
		ArticleID:     req.ArticleID,
		ArticleTitle:  compInfo.ArticleTitle,
		ComponentID:   req.ComponentID,
		ComponentName: compInfo.ComponentName,
		SewingRate:    req.SewingRate,
		CuttingRate:   req.CuttingRate,
		EffectiveDate: effectiveDate,
		Notes:         req.Notes,
	}

	initialHist := RateCardHistory{
		OldSewingRate:   0,
		NewSewingRate:   req.SewingRate,
		OldCuttingRate:  0,
		NewCuttingRate:  req.CuttingRate,
		ChangedByUserID: requesterUserID,
		Reason:          "Penetapan tarif awal komponen",
	}

	if err := s.repo.CreateRateCard(ctx, &rc, &initialHist); err != nil {
		return nil, err
	}

	return &rc, nil
}

// UpdateRateCard updates rate card prices and logs the change to audit trail.
func (s *Service) UpdateRateCard(ctx context.Context, id string, req UpdateRateCardRequest, requesterRole string, requesterUserID *string) (*RateCard, error) {
	if !canManageRateCards(requesterRole) {
		return nil, ErrForbiddenRateCardManage
	}

	if strings.TrimSpace(req.ChangeReason) == "" {
		return nil, ErrChangeReasonRequired
	}

	rc, err := s.repo.FindRateCardByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, ErrRateCardNotFound
	}

	newSewingRate := rc.SewingRate
	if req.SewingRate > 0 {
		newSewingRate = req.SewingRate
	}

	newCuttingRate := rc.CuttingRate
	if req.CuttingRate != nil {
		if *req.CuttingRate < 0 {
			return nil, ErrInvalidRateValue
		}
		newCuttingRate = *req.CuttingRate
	}

	if newSewingRate < 0 || newCuttingRate < 0 {
		return nil, ErrInvalidRateValue
	}

	if req.EffectiveDate != nil && strings.TrimSpace(*req.EffectiveDate) != "" {
		rc.EffectiveDate = strings.TrimSpace(*req.EffectiveDate)
	}

	history := RateCardHistory{
		RateCardID:      rc.ID,
		OldSewingRate:   rc.SewingRate,
		NewSewingRate:   newSewingRate,
		OldCuttingRate:  rc.CuttingRate,
		NewCuttingRate:  newCuttingRate,
		ChangedByUserID: requesterUserID,
		Reason:          strings.TrimSpace(req.ChangeReason),
	}

	rc.SewingRate = newSewingRate
	rc.CuttingRate = newCuttingRate

	if err := s.repo.UpdateRateCard(ctx, rc, &history); err != nil {
		return nil, err
	}

	return rc, nil
}

// ListRateCardHistories retrieves audit trail entries for a given rate card.
func (s *Service) ListRateCardHistories(ctx context.Context, rateCardID string, requesterRole string, requesterClientID *string) ([]RateCardHistory, error) {
	rc, err := s.repo.FindRateCardByID(ctx, rateCardID)
	if err != nil {
		return nil, err
	}
	if rc == nil {
		return nil, ErrRateCardNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != rc.ClientID {
			return nil, ErrForbiddenRateCardAccess
		}
	}

	return s.repo.ListRateCardHistories(ctx, rateCardID)
}

