package masterdata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository constructs a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ListClients retrieves clients matching the search filter with pagination.
func (r *PostgresRepository) ListClients(ctx context.Context, filter ListClientsFilter) ([]Client, int64, error) {
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

	offset := (page - 1) * limit
	searchTerm := strings.TrimSpace(filter.Search)

	var total int64
	var rows *sql.Rows
	var err error

	if searchTerm != "" {
		countQuery := `
			SELECT COUNT(*)
			FROM clients
			WHERE deleted_at IS NULL
			  AND (LOWER(name) LIKE $1 OR LOWER(code) LIKE $1);
		`
		pattern := "%" + strings.ToLower(searchTerm) + "%"
		if err := r.db.QueryRowContext(ctx, countQuery, pattern).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count clients: %w", err)
		}

		selectQuery := `
			SELECT
				id::text,
				code,
				name,
				contact,
				address,
				client_type,
				created_at,
				updated_at
			FROM clients
			WHERE deleted_at IS NULL
			  AND (LOWER(name) LIKE $1 OR LOWER(code) LIKE $1)
			ORDER BY created_at ASC, name ASC
			LIMIT $2 OFFSET $3;
		`
		rows, err = r.db.QueryContext(ctx, selectQuery, pattern, limit, offset)
	} else {
		countQuery := `SELECT COUNT(*) FROM clients WHERE deleted_at IS NULL;`
		if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count clients: %w", err)
		}

		selectQuery := `
			SELECT
				id::text,
				code,
				name,
				contact,
				address,
				client_type,
				created_at,
				updated_at
			FROM clients
			WHERE deleted_at IS NULL
			ORDER BY created_at ASC, name ASC
			LIMIT $1 OFFSET $2;
		`
		rows, err = r.db.QueryContext(ctx, selectQuery, limit, offset)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("query clients: %w", err)
	}
	defer rows.Close()

	clients := make([]Client, 0)
	for rows.Next() {
		var c Client
		if err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Name,
			&c.Contact,
			&c.Address,
			&c.ClientType,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan client: %w", err)
		}
		clients = append(clients, c)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("clients rows iteration: %w", err)
	}

	return clients, total, nil
}

