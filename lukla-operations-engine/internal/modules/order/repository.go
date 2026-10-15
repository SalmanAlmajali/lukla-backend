package order

import (
	"context"
)

// Repository mendefinisikan antarmuka abstraksi data layer untuk manajemen pesanan internal pabrik.
type Repository interface {
	// List mengambil seluruh daftar pesanan pabrik dengan filter opsional clientId dan status.
	List(ctx context.Context, clientID, status string) ([]InternalOrderSummary, error)

	// GetByID mengambil detail lengkap pesanan pabrik berdasarkan UUID.
	GetByID(ctx context.Context, id string) (*InternalOrderDetail, error)

	// Complete memperbarui status order menjadi COMPLETED dan mengisi completed_at = NOW().
	Complete(ctx context.Context, id string, notes *string) (*InternalOrderDetail, error)
}

