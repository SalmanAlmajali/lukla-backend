package notification

import (
	"context"
	"database/sql"
	"fmt"
)

// PostgresRepository mengimplementasikan antarmuka Repository menggunakan PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository membuat instance baru PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create menyimpan rekaman notifikasi baru ke tabel notifications.
func (r *PostgresRepository) Create(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (
			client_id, title, message, event_type, reference_id, is_read, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(
		ctx,
		query,
		n.ClientID,
		n.Title,
		n.Message,
		n.EventType,
		n.ReferenceID,
		n.IsRead,
	).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

// ListByClient mengambil daftar notifikasi milik klien dan menghitung jumlah notifikasi belum dibaca.
func (r *PostgresRepository) ListByClient(ctx context.Context, clientID string, unreadOnly bool, limit int) ([]Notification, int, error) {
	if limit <= 0 {
		limit = 20
	}

	// 1. Hitung total notifikasi belum dibaca milik klien
	var unreadCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications 
		WHERE client_id = $1 AND is_read = FALSE
	`, clientID).Scan(&unreadCount)
	if err != nil {
		return nil, 0, fmt.Errorf("count unread notifications: %w", err)
	}

	// 2. Ambil list notifikasi dengan filter
	query := `
		SELECT id, client_id, title, message, event_type, reference_id, is_read, read_at, created_at
		FROM notifications
		WHERE client_id = $1
		  AND ($2 = FALSE OR is_read = FALSE)
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, query, clientID, unreadOnly, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]Notification, 0)
	for rows.Next() {
		var n Notification
		var refID sql.NullString
		var readAt sql.NullTime

		if err := rows.Scan(
			&n.ID,
			&n.ClientID,
			&n.Title,
			&n.Message,
			&n.EventType,
			&refID,
			&n.IsRead,
			&readAt,
			&n.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}

		if refID.Valid {
			n.ReferenceID = &refID.String
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}

		notifications = append(notifications, n)
	}

	return notifications, unreadCount, nil
}

// MarkAsRead menandai notifikasi spesifik sebagai sudah dibaca dengan perlindungan multi-tenant.
func (r *PostgresRepository) MarkAsRead(ctx context.Context, clientID, notificationID string) error {
	query := `
		UPDATE notifications 
		SET is_read = TRUE, read_at = NOW() 
		WHERE id = $1 AND client_id = $2
	`
	res, err := r.db.ExecContext(ctx, query, notificationID, clientID)
	if err != nil {
		return fmt.Errorf("mark notification as read: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// MarkAllAsRead menandai seluruh notifikasi belum dibaca milik klien menjadi sudah dibaca.
func (r *PostgresRepository) MarkAllAsRead(ctx context.Context, clientID string) error {
	query := `
		UPDATE notifications 
		SET is_read = TRUE, read_at = NOW() 
		WHERE client_id = $1 AND is_read = FALSE
	`
	_, err := r.db.ExecContext(ctx, query, clientID)
	if err != nil {
		return fmt.Errorf("mark all notifications as read: %w", err)
	}
	return nil
}

// CountUnread menghitung jumlah notifikasi yang belum dibaca untuk klien tertentu.
func (r *PostgresRepository) CountUnread(ctx context.Context, clientID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications 
		WHERE client_id = $1 AND is_read = FALSE
	`, clientID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return count, nil
}

