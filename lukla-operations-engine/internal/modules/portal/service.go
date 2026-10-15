package portal

import (
	"context"
	"strings"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Service provides business logic and multi-tenant security guards for client portal.
type Service struct {
	repo Repository
}

// NewService creates a new portal Service instance.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

