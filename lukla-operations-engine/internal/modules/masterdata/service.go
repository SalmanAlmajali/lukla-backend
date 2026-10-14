package masterdata

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/lukla/operations-engine/internal/platform/httpx"
)

var (
	ErrClientNotFound        = errors.New("Klien tidak ditemukan")
	ErrForbiddenClientAccess = errors.New("Akses ditolak: Anda hanya dapat mengakses data klien milik akun Anda sendiri")
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

// GetClientDetail retrieves the details of a client by ID.
// Staf internal dapat melihat semua klien.
// Pengguna dengan peran klien (client / CLIENT_PORTAL) dibatasi hanya pada client_id miliknya.
func (s *Service) GetClientDetail(ctx context.Context, clientID string, requesterRole string, requesterClientID *string) (*ClientDetail, error) {
	detail, err := s.repo.FindClientByID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, ErrClientNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != detail.ID {
			return nil, ErrForbiddenClientAccess
		}
	}

	return detail, nil
}


