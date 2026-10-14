package masterdata

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/lukla/operations-engine/internal/platform/httpx"
)

var (
	ErrClientNotFound         = errors.New("Klien tidak ditemukan")
	ErrForbiddenClientAccess  = errors.New("Akses ditolak: Anda hanya dapat mengakses data klien milik akun Anda sendiri")
	ErrForbiddenGeneralAccess = errors.New("Akses ditolak: akun klien eksternal tidak diizinkan mengakses data artikel umum pabrik. Harap sertakan parameter clientId milik akun Anda.")
	ErrArticleNotFound        = errors.New("Artikel tidak ditemukan")
	ErrDuplicateArticleCode   = errors.New("Kode artikel sudah digunakan")
	ErrInvalidCategory        = errors.New("Kategori produk tidak valid, harus 'GARMENT' atau 'BAG'")
	ErrForbiddenArticleAccess = errors.New("Akses ditolak: Anda hanya dapat mengakses data artikel milik klien Anda sendiri")
	ErrInvalidOperatorStatus  = errors.New("Status operator tidak valid, harus 'ACTIVE', 'INACTIVE', atau 'ON_LEAVE'")
	ErrForbiddenOperatorAccess = errors.New("Akses ditolak: akun klien eksternal tidak diizinkan mengakses data master operator pabrik")
	ErrOperatorNotFound       = errors.New("Operator jahit tidak ditemukan")
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

// ListArticles retrieves a paginated list of articles with multi-tenant scoping and category filters.
// Staf internal berlaku umum (dapat mengakses seluruh artikel atau memfilter per clientId).
// Pengguna peran klien (client / CLIENT_PORTAL) tidak dapat mengakses data umum pabrik (wajib menyertakan clientId miliknya).
func (s *Service) ListArticles(ctx context.Context, filter ListArticlesFilter, requesterRole string, requesterClientID *string) ([]Article, httpx.PaginationMeta, error) {
	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID == "" {
			return nil, httpx.PaginationMeta{}, ErrForbiddenArticleAccess
		}
		if filter.ClientID == "" {
			return nil, httpx.PaginationMeta{}, ErrForbiddenGeneralAccess
		}
		if filter.ClientID != *requesterClientID {
			return nil, httpx.PaginationMeta{}, ErrForbiddenArticleAccess
		}
	}

	if filter.Category != "" {
		cat := strings.ToUpper(strings.TrimSpace(filter.Category))
		if cat != "GARMENT" && cat != "BAG" {
			return nil, httpx.PaginationMeta{}, ErrInvalidCategory
		}
		filter.Category = cat
	}

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

	articles, total, err := s.repo.ListArticles(ctx, ListArticlesFilter{
		ClientID: filter.ClientID,
		Category: filter.Category,
		Search:   filter.Search,
		Page:     page,
		Limit:    limit,
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

	return articles, meta, nil
}

// GetArticleDetail retrieves the detailed information of an article by ID, including its components.
func (s *Service) GetArticleDetail(ctx context.Context, id string, requesterRole string, requesterClientID *string) (*ArticleDetail, error) {
	art, err := s.repo.FindArticleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if art == nil {
		return nil, ErrArticleNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != art.ClientID {
			return nil, ErrForbiddenArticleAccess
		}
	}

	comps, err := s.repo.ListComponentsByArticleID(ctx, art.ID)
	if err != nil {
		return nil, err
	}

	return &ArticleDetail{
		Article:    *art,
		Components: comps,
	}, nil
}

// CreateArticle validates and creates a new article record.
func (s *Service) CreateArticle(ctx context.Context, req CreateArticleRequest, requesterRole string, requesterClientID *string) (*Article, error) {
	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != req.ClientID {
			return nil, ErrForbiddenArticleAccess
		}
	}

	category := strings.ToUpper(strings.TrimSpace(req.Category))
	if category != "GARMENT" && category != "BAG" {
		return nil, ErrInvalidCategory
	}

	client, err := s.repo.FindClientByID(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}

	exists, err := s.repo.ArticleCodeExists(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateArticleCode
	}

	article := Article{
		ClientID:    req.ClientID,
		Code:        strings.TrimSpace(req.Code),
		Title:       strings.TrimSpace(req.Title),
		Category:    category,
		Description: req.Description,
	}

	if err := s.repo.CreateArticle(ctx, &article); err != nil {
		return nil, err
	}

	return &article, nil
}

// ListComponents retrieves the operational components of an article.
func (s *Service) ListComponents(ctx context.Context, articleID string, requesterRole string, requesterClientID *string) ([]Component, error) {
	art, err := s.repo.FindArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if art == nil {
		return nil, ErrArticleNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != art.ClientID {
			return nil, ErrForbiddenArticleAccess
		}
	}

	return s.repo.ListComponentsByArticleID(ctx, art.ID)
}

// CreateComponent adds an operational component to an article with optional Standard SAM.
func (s *Service) CreateComponent(ctx context.Context, articleID string, req CreateComponentRequest, requesterRole string, requesterClientID *string) (*Component, error) {
	art, err := s.repo.FindArticleByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if art == nil {
		return nil, ErrArticleNotFound
	}

	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		if requesterClientID == nil || *requesterClientID != art.ClientID {
			return nil, ErrForbiddenArticleAccess
		}
	}

	seq := 1
	if req.SequenceNumber != nil && *req.SequenceNumber > 0 {
		seq = *req.SequenceNumber
	} else {
		comps, err := s.repo.ListComponentsByArticleID(ctx, art.ID)
		if err == nil {
			seq = len(comps) + 1
		}
	}

	comp := Component{
		ArticleID:      art.ID,
		Name:           strings.TrimSpace(req.Name),
		Code:           req.Code,
		SequenceNumber: seq,
		StandardSAM:    req.StandardSAM,
	}

	if err := s.repo.CreateComponent(ctx, &comp); err != nil {
		return nil, err
	}

	return &comp, nil
}

// ListOperators retrieves the list of sewing operators / tailors in the factory.
// Hanya staf internal yang diizinkan mengakses data master operator pabrik.
// Pengguna dengan peran klien (client / CLIENT_PORTAL) akan ditolak (403 Forbidden).
func (s *Service) ListOperators(ctx context.Context, filter ListOperatorsFilter, requesterRole string) ([]Operator, error) {
	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		return nil, ErrForbiddenOperatorAccess
	}

	if filter.Status != "" {
		st := strings.ToUpper(strings.TrimSpace(filter.Status))
		if st != string(OperatorStatusActive) && st != string(OperatorStatusInactive) && st != string(OperatorStatusOnLeave) {
			return nil, ErrInvalidOperatorStatus
		}
		filter.Status = st
	}

	if filter.Line != "" {
		filter.Line = strings.ToUpper(strings.TrimSpace(filter.Line))
	}

	return s.repo.ListOperators(ctx, filter)
}

// GetOperatorDetail retrieves operator details by ID or operator code.
func (s *Service) GetOperatorDetail(ctx context.Context, id string, requesterRole string) (*Operator, error) {
	normRole := strings.ToLower(strings.TrimSpace(requesterRole))
	if normRole == "client" || normRole == "client_portal" {
		return nil, ErrForbiddenOperatorAccess
	}

	op, err := s.repo.FindOperatorByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, ErrOperatorNotFound
	}

	return op, nil
}


