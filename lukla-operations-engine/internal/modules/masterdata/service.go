package masterdata

import (
	"context"
	"math"

	"github.com/lukla/operations-engine/internal/platform/httpx"
)

// Service handles business logic for master data entities.
type Service struct {
	repo Repository
}

// NewService constructs a masterdata Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListClients retrieves a paginated list of clients along with pagination metadata.
func (s *Service) ListClients(ctx context.Context, filter ListClientsFilter) ([]Client, httpx.PaginationMeta, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	clients, total, err := s.repo.ListClients(ctx, ListClientsFilter{
		Search: filter.Search,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		return nil, httpx.PaginationMeta{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	meta := httpx.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return clients, meta, nil
}

