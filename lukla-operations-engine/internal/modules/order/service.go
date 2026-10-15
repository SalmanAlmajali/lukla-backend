package order

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukla/operations-engine/internal/modules/notification"
)

// Service mengelola alur kerja siklus hidup pesanan pabrik.
type Service struct {
	repo            Repository
	notificationSvc *notification.Service
}

// NewService membuat instance Service baru.
func NewService(repo Repository, notificationSvc *notification.Service) *Service {
	return &Service{
		repo:            repo,
		notificationSvc: notificationSvc,
	}
}

// ListOrders mengambil daftar seluruh pesanan internal pabrik.
func (s *Service) ListOrders(ctx context.Context, clientID, status string) ([]InternalOrderSummary, error) {
	return s.repo.List(ctx, strings.TrimSpace(clientID), strings.TrimSpace(status))
}

// GetOrderDetail mengambil rincian mendalam pesanan pabrik.
func (s *Service) GetOrderDetail(ctx context.Context, id string) (*InternalOrderDetail, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrOrderNotFound
	}
	return s.repo.GetByID(ctx, id)
}

// CompleteOrder menandai pesanan selesai dan memicu notifikasi otomatis ke klien pemilik pesanan.
func (s *Service) CompleteOrder(ctx context.Context, id string, notes *string) (*InternalOrderDetail, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrOrderNotFound
	}

	detail, err := s.repo.Complete(ctx, id, notes)
	if err != nil {
		return nil, err
	}

	// Trigger notifikasi otomatis ke klien jika notificationSvc tersedia
	if s.notificationSvc != nil && detail.ClientID != "" {
		refID := detail.ID
		_, _ = s.notificationSvc.SendNotification(ctx, notification.SendNotificationRequest{
			ClientID:    detail.ClientID,
			Title:       fmt.Sprintf("Pesanan Selesai: %s", detail.OrderNumber),
			Message:     fmt.Sprintf("Pesanan %s (%s) telah selesai diproduksi dan siap diserahterimakan.", detail.OrderNumber, detail.ArticleTitle),
			EventType:   notification.EventTypeOrderCompleted,
			ReferenceID: &refID,
		})
	}

	return detail, nil
}

