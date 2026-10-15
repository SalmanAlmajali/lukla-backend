package portal

import (
	"context"
)

// Repository defines data access contract for client portal operations.
type Repository interface {
	GetDashboard(ctx context.Context, clientID string) (*ClientPortalDashboard, error)
	ListOrders(ctx context.Context, clientID string, filter ListClientOrdersFilter) ([]ClientPortalOrderSummary, int, error)
	GetOrderDetail(ctx context.Context, clientID, orderID string) (*ClientPortalOrderDetail, error)
	GetOrderQuality(ctx context.Context, clientID, orderID string) (*ClientPortalOrderQuality, error)
	ListMaterialStocks(ctx context.Context, clientID, category string) ([]ClientPortalMaterialStock, error)
	ListFinishedGoodsStocks(ctx context.Context, clientID string) ([]ClientPortalFinishedGoodsStock, error)
}

