package portal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lukla/operations-engine/internal/modules/notification"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Service provides business logic and multi-tenant security guards for client portal.
type Service struct {
	repo            Repository
	notificationSvc *notification.Service
}

// NewService creates a new portal Service instance.
func NewService(repo Repository, notificationSvc ...*notification.Service) *Service {
	var notif *notification.Service
	if len(notificationSvc) > 0 {
		notif = notificationSvc[0]
	}
	return &Service{repo: repo, notificationSvc: notif}
}

// validateClientTenant ensures actor is authorized and bound to a valid client tenant.
func (s *Service) validateClientTenant(actor *authn.UserIdentity) (string, error) {
	if actor == nil {
		return "", ErrUnauthorizedPortalAccess
	}

	role := strings.ToLower(actor.RoleCode)
	if role != "client" && role != "owner_executive" {
		return "", ErrUnauthorizedPortalAccess
	}

	if actor.ClientID == nil || *actor.ClientID == "" {
		return "", ErrClientTenantRequired
	}

	return *actor.ClientID, nil
}

// GetDashboard retrieves dashboard metrics for authenticated client.
func (s *Service) GetDashboard(ctx context.Context, actor *authn.UserIdentity) (*ClientPortalDashboard, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDashboard(ctx, clientID)
}

// ListOrders retrieves orders belonging to authenticated client.
func (s *Service) ListOrders(ctx context.Context, actor *authn.UserIdentity, filter ListClientOrdersFilter) ([]ClientPortalOrderSummary, int, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListOrders(ctx, clientID, filter)
}

// GetOrderDetail retrieves detailed order progress.
func (s *Service) GetOrderDetail(ctx context.Context, actor *authn.UserIdentity, orderID string) (*ClientPortalOrderDetail, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, ErrOrderNotFound
	}
	return s.repo.GetOrderDetail(ctx, clientID, orderID)
}

// GetOrderQuality retrieves quality analytics for client order.
func (s *Service) GetOrderQuality(ctx context.Context, actor *authn.UserIdentity, orderID string) (*ClientPortalOrderQuality, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, ErrOrderNotFound
	}
	return s.repo.GetOrderQuality(ctx, clientID, orderID)
}

// ListMaterialStocks retrieves consigned raw materials for authenticated client.
func (s *Service) ListMaterialStocks(ctx context.Context, actor *authn.UserIdentity, category string) ([]ClientPortalMaterialStock, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMaterialStocks(ctx, clientID, category)
}

// ListFinishedGoodsStocks retrieves ready garments for authenticated client.
func (s *Service) ListFinishedGoodsStocks(ctx context.Context, actor *authn.UserIdentity) ([]ClientPortalFinishedGoodsStock, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	return s.repo.ListFinishedGoodsStocks(ctx, clientID)
}

// ListSamples retrieves sample requests for authenticated client.
func (s *Service) ListSamples(ctx context.Context, actor *authn.UserIdentity, status string) ([]ClientPortalSampleSummary, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	return s.repo.ListClientSamples(ctx, clientID, status)
}

// GetSampleDetail retrieves comprehensive drill-down of a sample for authenticated client.
func (s *Service) GetSampleDetail(ctx context.Context, actor *authn.UserIdentity, sampleID string) (*ClientPortalSampleDetail, error) {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	return s.repo.GetClientSampleDetail(ctx, clientID, strings.TrimSpace(sampleID))
}

// ApproveSample handles client approval of a sample version.
func (s *Service) ApproveSample(ctx context.Context, actor *authn.UserIdentity, sampleID string, feedbackNotes *string) error {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return err
	}
	if strings.TrimSpace(sampleID) == "" {
		return ErrSampleNotFound
	}

	err = s.repo.ApproveSample(ctx, clientID, strings.TrimSpace(sampleID), feedbackNotes)
	if err != nil {
		return err
	}

	if s.notificationSvc != nil {
		refID := sampleID
		msg := "Klien telah menyetujui versi fisik dan penawaran harga komersial sample. Siap untuk penerbitan PO produksi massal."
		if feedbackNotes != nil && strings.TrimSpace(*feedbackNotes) != "" {
			msg = fmt.Sprintf("%s Catatan klien: %s", msg, strings.TrimSpace(*feedbackNotes))
		}
		_, _ = s.notificationSvc.SendNotification(ctx, notification.SendNotificationRequest{
			ClientID:    clientID,
			Title:       "Sample Disetujui Klien",
			Message:     msg,
			EventType:   notification.EventTypeSampleApproved,
			ReferenceID: &refID,
		})
	}

	return nil
}

// RequestSampleRevision handles client request for sample revision.
func (s *Service) RequestSampleRevision(ctx context.Context, actor *authn.UserIdentity, sampleID string, revisionNotes string) error {
	clientID, err := s.validateClientTenant(actor)
	if err != nil {
		return err
	}
	if strings.TrimSpace(sampleID) == "" {
		return ErrSampleNotFound
	}
	if strings.TrimSpace(revisionNotes) == "" {
		return errors.New("catatan revisi wajib diisi")
	}

	err = s.repo.RequestSampleRevision(ctx, clientID, strings.TrimSpace(sampleID), strings.TrimSpace(revisionNotes))
	if err != nil {
		return err
	}

	if s.notificationSvc != nil {
		refID := sampleID
		msg := fmt.Sprintf("Klien meminta perbaikan teknis pada sample prototipe. Catatan revisi: %s", strings.TrimSpace(revisionNotes))
		_, _ = s.notificationSvc.SendNotification(ctx, notification.SendNotificationRequest{
			ClientID:    clientID,
			Title:       "Permintaan Revisi Sample dari Klien",
			Message:     msg,
			EventType:   notification.EventTypeSampleRevisionRequested,
			ReferenceID: &refID,
		})
	}

	return nil
}

