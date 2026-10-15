package order

import (
	"context"
	"testing"
	"time"

	"github.com/lukla/operations-engine/internal/modules/notification"
)

type mockOrderRepo struct {
	orders map[string]*InternalOrderDetail
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[string]*InternalOrderDetail),
	}
}

func (m *mockOrderRepo) List(ctx context.Context, clientID, status string) ([]InternalOrderSummary, error) {
	var list []InternalOrderSummary
	for _, o := range m.orders {
		if (clientID == "" || o.ClientID == clientID) && (status == "" || status == "ALL" || o.Status == status) {
			list = append(list, o.InternalOrderSummary)
		}
	}
	return list, nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id string) (*InternalOrderDetail, error) {
	o, exists := m.orders[id]
	if !exists {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) Complete(ctx context.Context, id string, notes *string) (*InternalOrderDetail, error) {
	o, exists := m.orders[id]
	if !exists {
		return nil, ErrOrderNotFound
	}
	if o.Status == "COMPLETED" {
		return nil, ErrOrderAlreadyCompleted
	}
	o.Status = "COMPLETED"
	now := time.Now()
	o.CompletedAt = &now
	if notes != nil {
		o.Notes = notes
	}
	return o, nil
}

type mockNotificationRepo struct {
	created []*notification.Notification
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *notification.Notification) error {
	m.created = append(m.created, n)
	return nil
}
func (m *mockNotificationRepo) ListByClient(ctx context.Context, clientID string, unreadOnly bool, limit int) ([]notification.Notification, int, error) {
	return nil, 0, nil
}
func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, clientID, notificationID string) error {
	return nil
}
func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, clientID string) error {
	return nil
}
func (m *mockNotificationRepo) CountUnread(ctx context.Context, clientID string) (int, error) {
	return 0, nil
}

func TestOrderService_CompleteOrderFlow(t *testing.T) {
	repo := newMockOrderRepo()
	notifRepo := &mockNotificationRepo{}
	notifSvc := notification.NewService(notifRepo)
	svc := NewService(repo, notifSvc)
	ctx := context.Background()

	// Seed order
	orderID := "order-test-1"
	repo.orders[orderID] = &InternalOrderDetail{
		InternalOrderSummary: InternalOrderSummary{
			ID:             orderID,
			OrderNumber:    "PO-TEST-001",
			ClientID:       "client-uuid-1",
			ClientName:     "Brand Test",
			ArticleID:      "art-1",
			ArticleCode:    "ART-01",
			ArticleTitle:   "Parka Test",
			TargetQuantity: 100,
			PassedQuantity: 100,
			Status:         "IN_PRODUCTION",
		},
	}

	// 1. List Orders
	orders, err := svc.ListOrders(ctx, "", "IN_PRODUCTION")
	if err != nil || len(orders) != 1 {
		t.Fatalf("expected 1 order in list, got %d (err: %v)", len(orders), err)
	}

	// 2. Get Detail
	detail, err := svc.GetOrderDetail(ctx, orderID)
	if err != nil || detail.Status != "IN_PRODUCTION" {
		t.Fatalf("expected order in production, got %+v (err: %v)", detail, err)
	}

	// 3. Complete Order
	notes := "Verifikasi QC 100% lolos"
	completed, err := svc.CompleteOrder(ctx, orderID, &notes)
	if err != nil {
		t.Fatalf("expected complete order success, got %v", err)
	}
	if completed.Status != "COMPLETED" || completed.CompletedAt == nil {
		t.Fatalf("expected status COMPLETED and completedAt set, got %+v", completed)
	}

	// 4. Verify Notif Sent to Client
	if len(notifRepo.created) != 1 {
		t.Fatalf("expected 1 notification created for client, got %d", len(notifRepo.created))
	}
	notif := notifRepo.created[0]
	if notif.ClientID != "client-uuid-1" || notif.EventType != notification.EventTypeOrderCompleted {
		t.Fatalf("expected ORDER_COMPLETED notif for client-uuid-1, got %+v", notif)
	}

	// 5. Complete again should fail (ErrOrderAlreadyCompleted)
	_, err = svc.CompleteOrder(ctx, orderID, nil)
	if err != ErrOrderAlreadyCompleted {
		t.Fatalf("expected ErrOrderAlreadyCompleted, got %v", err)
	}
}

