package masterdata

import "context"

// Repository defines data operations for master data entities.
type Repository interface {
	ListClients(ctx context.Context, filter ListClientsFilter) ([]Client, int64, error)
}

