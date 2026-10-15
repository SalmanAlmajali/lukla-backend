package portal

import (
	"context"
	"testing"
	"time"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

type mockPortalRepo struct {
	dashboard     *ClientPortalDashboard
	orders        []ClientPortalOrderSummary
	totalOrders   int
	orderDetail   *ClientPortalOrderDetail
	orderQuality  *ClientPortalOrderQuality
	materialStock []ClientPortalMaterialStock
	finishedGoods []ClientPortalFinishedGoodsStock
	err           error
}

func (m *mockPortalRepo) GetDashboard(ctx context.Context, clientID string) (*ClientPortalDashboard, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dashboard, nil
}

func (m *mockPortalRepo) ListOrders(ctx context.Context, clientID string, filter ListClientOrdersFilter) ([]ClientPortalOrderSummary, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.orders, m.totalOrders, nil
}

func (m *mockPortalRepo) GetOrderDetail(ctx context.Context, clientID, orderID string) (*ClientPortalOrderDetail, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orderDetail, nil
}

func (m *mockPortalRepo) GetOrderQuality(ctx context.Context, clientID, orderID string) (*ClientPortalOrderQuality, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orderQuality, nil
}

func (m *mockPortalRepo) ListMaterialStocks(ctx context.Context, clientID, category string) ([]ClientPortalMaterialStock, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.materialStock, nil
}

func (m *mockPortalRepo) ListFinishedGoodsStocks(ctx context.Context, clientID string) ([]ClientPortalFinishedGoodsStock, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.finishedGoods, nil
}

func TestPortalService_MultiTenantValidation(t *testing.T) {
	ctx := context.Background()
	repo := &mockPortalRepo{}
	svc := NewService(repo)

	// 1. Nil actor
	_, err := svc.GetDashboard(ctx, nil)
	if err != ErrUnauthorizedPortalAccess {
		t.Errorf("expected ErrUnauthorizedPortalAccess for nil actor, got %v", err)
	}

	// 2. Non-client role (e.g. tailor)
	tailorID := "user-tailor-01"
	clientID := "client-sno-01"
	nonClientActor := &authn.UserIdentity{
		ID:       tailorID,
		RoleCode: "operator_tailor",
		ClientID: &clientID,
	}
	_, err = svc.GetDashboard(ctx, nonClientActor)
	if err != ErrUnauthorizedPortalAccess {
		t.Errorf("expected ErrUnauthorizedPortalAccess for operator_tailor, got %v", err)
	}

	// 3. Client role without ClientID
	clientNoTenant := &authn.UserIdentity{
		ID:       "user-client-01",
		RoleCode: "client",
		ClientID: nil,
	}
	_, err = svc.GetDashboard(ctx, clientNoTenant)
	if err != ErrClientTenantRequired {
		t.Errorf("expected ErrClientTenantRequired when clientID is nil, got %v", err)
	}

	// 4. Valid client actor
	validClient := &authn.UserIdentity{
		ID:       "user-client-01",
		RoleCode: "client",
		ClientID: &clientID,
	}
	repo.dashboard = &ClientPortalDashboard{
		Client: ClientInfo{
			ID:   clientID,
			Name: "Snøhund Brand",
			Code: "CLIENT-SNO-01",
		},
		ActiveOrdersCount: 2,
	}

	dash, err := svc.GetDashboard(ctx, validClient)
	if err != nil {
		t.Fatalf("unexpected error for valid client: %v", err)
	}
	if dash.Client.Name != "Snøhund Brand" {
		t.Errorf("expected client name 'Snøhund Brand', got %s", dash.Client.Name)
	}
}

func TestPortalService_OrderDrillDownAndStocks(t *testing.T) {
	ctx := context.Background()
	clientID := "client-sno-01"
	orderID := "order-po-01"

	repo := &mockPortalRepo{
		orderDetail: &ClientPortalOrderDetail{
			ID:             orderID,
			OrderNumber:    "PO-2026-10-001",
			TargetQuantity: 3000,
			Status:         "IN_PRODUCTION",
			Progress: ClientOrderProgress{
				CuttingQuantity:   3000,
				SewingQuantity:    2500,
				PassedQCQuantity:  2000,
				OverallPercentage: 66.7,
			},
		},
		orderQuality: &ClientPortalOrderQuality{
			OrderID:        orderID,
			OrderNumber:    "PO-2026-10-001",
			TargetQuantity: 3000,
			TotalChecked:   2100,
			PassedQuantity: 2000,
			RepairQuantity: 80,
			RejectQuantity: 20,
			PassRate:       95.2,
		},
		materialStock: []ClientPortalMaterialStock{
			{
				MaterialID:             "mat-fab-01",
				MaterialName:           "Kain Cordura 500D",
				TotalAvailableQuantity: 250.0,
				Lots: []ClientPortalLotStock{
					{LotNumber: "LOT-01", AvailableQuantity: 250.0},
				},
			},
		},
		finishedGoods: []ClientPortalFinishedGoodsStock{
			{
				OrderID:       orderID,
				OrderNumber:   "PO-2026-10-001",
				ReadyQuantity: 2000,
				Status:        "READY_FOR_DISPATCH",
			},
		},
	}
	svc := NewService(repo)

	clientActor := &authn.UserIdentity{
		ID:       "user-client-01",
		RoleCode: "client",
		ClientID: &clientID,
	}

	// Test GetOrderDetail
	detail, err := svc.GetOrderDetail(ctx, clientActor, orderID)
	if err != nil {
		t.Fatalf("unexpected error getting order detail: %v", err)
	}
	if detail.Progress.PassedQCQuantity != 2000 {
		t.Errorf("expected 2000 passed QC, got %d", detail.Progress.PassedQCQuantity)
	}

	// Test GetOrderQuality
	quality, err := svc.GetOrderQuality(ctx, clientActor, orderID)
	if err != nil {
		t.Fatalf("unexpected error getting order quality: %v", err)
	}
	if quality.PassRate != 95.2 {
		t.Errorf("expected pass rate 95.2, got %f", quality.PassRate)
	}

	// Test ListMaterialStocks
	stocks, err := svc.ListMaterialStocks(ctx, clientActor, "FABRIC")
	if err != nil {
		t.Fatalf("unexpected error listing material stocks: %v", err)
	}
	if len(stocks) != 1 || stocks[0].TotalAvailableQuantity != 250.0 {
		t.Errorf("unexpected material stock result: %+v", stocks)
	}

	// Test ListFinishedGoodsStocks
	fg, err := svc.ListFinishedGoodsStocks(ctx, clientActor)
	if err != nil {
		t.Fatalf("unexpected error listing finished goods stocks: %v", err)
	}
	if len(fg) != 1 || fg[0].ReadyQuantity != 2000 {
		t.Errorf("unexpected finished goods result: %+v", fg)
	}
}

// Verify date parsing / timestamps works cleanly
func TestPortal_ModelStructs(t *testing.T) {
	now := time.Now()
	summary := ClientPortalOrderSummary{
		ID:                 "id",
		OrderNumber:        "PO-01",
		TargetQuantity:     100,
		PassedQuantity:     100,
		ProgressPercentage: 100.0,
		Status:             "COMPLETED",
		CompletedAt:        &now,
		CreatedAt:          now,
	}
	if summary.ProgressPercentage != 100.0 {
		t.Errorf("expected 100.0, got %f", summary.ProgressPercentage)
	}
}

