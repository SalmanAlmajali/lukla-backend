package notification

import (
	"context"
	"testing"
	"time"
)

type mockNotificationRepo struct {
	notifications map[string]*Notification
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		notifications: make(map[string]*Notification),
	}
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *Notification) error {
	n.ID = "notif-uuid-1"
	n.CreatedAt = time.Now()
	m.notifications[n.ID] = n
	return nil
}

func (m *mockNotificationRepo) ListByClient(ctx context.Context, clientID string, unreadOnly bool, limit int) ([]Notification, int, error) {
	var list []Notification
	unread := 0
	for _, n := range m.notifications {
		if n.ClientID == clientID {
			if !n.IsRead {
				unread++
			}
			if !unreadOnly || !n.IsRead {
				list = append(list, *n)
			}
		}
	}
	return list, unread, nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, clientID, notificationID string) error {
	n, exists := m.notifications[notificationID]
	if !exists || n.ClientID != clientID {
		return ErrNotificationNotFound
	}
	n.IsRead = true
	now := time.Now()
	n.ReadAt = &now
	return nil
}

func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, clientID string) error {
	now := time.Now()
	for _, n := range m.notifications {
		if n.ClientID == clientID && !n.IsRead {
			n.IsRead = true
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockNotificationRepo) CountUnread(ctx context.Context, clientID string) (int, error) {
	unread := 0
	for _, n := range m.notifications {
		if n.ClientID == clientID && !n.IsRead {
			unread++
		}
	}
	return unread, nil
}

func TestNotificationService_Flow(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := NewService(repo)
	ctx := context.Background()

	// 1. Send Notification
	req := SendNotificationRequest{
		ClientID:  "client-1",
		Title:     "Pesanan Selesai",
		Message:   "Pesanan PO-001 telah selesai diproduksi.",
		EventType: EventTypeOrderCompleted,
	}
	notif, err := svc.SendNotification(ctx, req)
	if err != nil {
		t.Fatalf("expected no error sending notif, got: %v", err)
	}
	if notif.ID == "" || notif.IsRead {
		t.Fatalf("expected unread notification with ID, got %+v", notif)
	}

	// 2. Count Unread
	unread, err := svc.CountUnread(ctx, "client-1")
	if err != nil || unread != 1 {
		t.Fatalf("expected 1 unread notification, got %d (err: %v)", unread, err)
	}

	// 3. List Notifications
	list, count, err := svc.ListClientNotifications(ctx, "client-1", false, 10)
	if err != nil || len(list) != 1 || count != 1 {
		t.Fatalf("expected 1 notif in list, got len %d count %d", len(list), count)
	}

	// 4. Mark As Read (Wrong Client / Cross Tenant Test)
	err = svc.MarkAsRead(ctx, "client-other", notif.ID)
	if err != ErrNotificationNotFound {
		t.Fatalf("expected ErrNotificationNotFound on cross-tenant read attempt, got %v", err)
	}

	// 5. Mark As Read (Valid)
	err = svc.MarkAsRead(ctx, "client-1", notif.ID)
	if err != nil {
		t.Fatalf("expected no error marking read, got %v", err)
	}

	unreadAfter, _ := svc.CountUnread(ctx, "client-1")
	if unreadAfter != 0 {
		t.Fatalf("expected 0 unread after mark read, got %d", unreadAfter)
	}

	// 6. Test Mark All As Read
	_, _ = svc.SendNotification(ctx, SendNotificationRequest{
		ClientID:  "client-1",
		Title:     "Penerimaan Kain",
		Message:   "Bahan baku kain telah diterima.",
		EventType: EventTypeMaterialReceived,
	})
	_ = svc.MarkAllAsRead(ctx, "client-1")
	unreadFinal, _ := svc.CountUnread(ctx, "client-1")
	if unreadFinal != 0 {
		t.Fatalf("expected 0 unread after mark all read, got %d", unreadFinal)
	}
}

