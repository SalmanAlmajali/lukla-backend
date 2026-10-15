package notification

import (
	"context"
)

// Repository mendefinisikan antarmuka abstraksi data layer untuk modul notifikasi.
type Repository interface {
	// Create menyimpan notifikasi baru ke basis data.
	Create(ctx context.Context, n *Notification) error

	// ListByClient mengambil daftar notifikasi milik klien dengan opsi filter hanya yang belum dibaca.
	ListByClient(ctx context.Context, clientID string, unreadOnly bool, limit int) ([]Notification, int, error)

	// MarkAsRead menandai notifikasi spesifik sebagai sudah dibaca (is_read = true, read_at = now).
	MarkAsRead(ctx context.Context, clientID, notificationID string) error

	// MarkAllAsRead menandai seluruh notifikasi belum dibaca milik klien menjadi sudah dibaca.
	MarkAllAsRead(ctx context.Context, clientID string) error

	// CountUnread menghitung jumlah notifikasi yang belum dibaca untuk klien tertentu.
	CountUnread(ctx context.Context, clientID string) (int, error)
}

