package notification

import (
	"context"
	"strings"
)

// Service mengelola logika bisnis sistem notifikasi in-app.
type Service struct {
	repo Repository
}

// NewService membuat instance Service baru.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SendNotification memvalidasi dan mengirim notifikasi baru ke basis data.
func (s *Service) SendNotification(ctx context.Context, req SendNotificationRequest) (*Notification, error) {
	clientID := strings.TrimSpace(req.ClientID)
	title := strings.TrimSpace(req.Title)
	msg := strings.TrimSpace(req.Message)
	eventType := strings.TrimSpace(req.EventType)

	if clientID == "" || title == "" || msg == "" || eventType == "" {
		return nil, ErrMissingRequiredField
	}

	switch eventType {
	case EventTypeOrderCompleted, EventTypeMaterialReceived, EventTypeProductionMilestone, EventTypeAnnouncement,
		EventTypeSampleReadyForReview, EventTypeSampleApproved, EventTypeSampleRevisionRequested:
		// Valid
	default:
		return nil, ErrInvalidEventType
	}

	n := &Notification{
		ClientID:    clientID,
		Title:       title,
		Message:     msg,
		EventType:   eventType,
		ReferenceID: req.ReferenceID,
		IsRead:      false,
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

// ListClientNotifications mengambil daftar notifikasi dan jumlah belum dibaca untuk klien bersangkutan.
func (s *Service) ListClientNotifications(ctx context.Context, clientID string, unreadOnly bool, limit int) ([]Notification, int, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, 0, ErrClientTenantRequired
	}
	return s.repo.ListByClient(ctx, clientID, unreadOnly, limit)
}

// MarkAsRead menandai notifikasi spesifik milik klien sebagai sudah dibaca.
func (s *Service) MarkAsRead(ctx context.Context, clientID, notificationID string) error {
	if strings.TrimSpace(clientID) == "" {
		return ErrClientTenantRequired
	}
	if strings.TrimSpace(notificationID) == "" {
		return ErrNotificationNotFound
	}
	return s.repo.MarkAsRead(ctx, clientID, notificationID)
}

// MarkAllAsRead menandai seluruh notifikasi belum dibaca milik klien menjadi sudah dibaca.
func (s *Service) MarkAllAsRead(ctx context.Context, clientID string) error {
	if strings.TrimSpace(clientID) == "" {
		return ErrClientTenantRequired
	}
	return s.repo.MarkAllAsRead(ctx, clientID)
}

// CountUnread menghitung total notifikasi yang belum dibaca milik klien.
func (s *Service) CountUnread(ctx context.Context, clientID string) (int, error) {
	if strings.TrimSpace(clientID) == "" {
		return 0, ErrClientTenantRequired
	}
	return s.repo.CountUnread(ctx, clientID)
}

