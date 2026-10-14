package masterdata

import (
	"context"
	"database/sql"
	"errors"
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

// FindClientByID retrieves client details by ID or unique code, including calculated articlesCount.
func (r *PostgresRepository) FindClientByID(ctx context.Context, id string) (*ClientDetail, error) {
	query := `
		SELECT
			c.id::text,
			c.code,
			c.name,
			c.contact,
			c.address,
			c.client_type,
			c.created_at,
			c.updated_at,
			(SELECT COUNT(*) FROM articles a WHERE a.client_id = c.id AND a.deleted_at IS NULL) AS articles_count
		FROM clients c
		WHERE (c.id::text = $1 OR LOWER(c.code) = LOWER($1)) AND c.deleted_at IS NULL
		LIMIT 1;
	`

	var detail ClientDetail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&detail.Code,
		&detail.Name,
		&detail.Contact,
		&detail.Address,
		&detail.ClientType,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.ArticlesCount,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find client by id: %w", err)
	}

	detail.ActiveOrdersCount = 0

	return &detail, nil
}

// ListArticles retrieves articles matching filter parameters with pagination.
func (r *PostgresRepository) ListArticles(ctx context.Context, filter ListArticlesFilter) ([]Article, int64, error) {
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

	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, "a.deleted_at IS NULL")

	if strings.TrimSpace(filter.ClientID) != "" {
		conditions = append(conditions, fmt.Sprintf("a.client_id = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.ClientID))
		argIdx++
	}

	if strings.TrimSpace(filter.Category) != "" {
		conditions = append(conditions, fmt.Sprintf("a.category = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(filter.Category)))
		argIdx++
	}

	if searchTerm := strings.TrimSpace(filter.Search); searchTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(a.title) LIKE $%d OR LOWER(a.code) LIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(searchTerm)+"%")
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM articles a WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count articles: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT
			a.id::text,
			a.client_id::text,
			c.name AS client_name,
			a.code,
			a.title,
			a.category,
			a.description,
			a.created_at,
			a.updated_at
		FROM articles a
		JOIN clients c ON a.client_id = c.id
		WHERE %s
		ORDER BY a.created_at DESC, a.title ASC
		LIMIT $%d OFFSET $%d;
	`, whereClause, argIdx, argIdx+1)

	selectArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query articles: %w", err)
	}
	defer rows.Close()

	articles := make([]Article, 0)
	for rows.Next() {
		var a Article
		if err := rows.Scan(
			&a.ID,
			&a.ClientID,
			&a.ClientName,
			&a.Code,
			&a.Title,
			&a.Category,
			&a.Description,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan article: %w", err)
		}
		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("articles rows iteration: %w", err)
	}

	return articles, total, nil
}

// FindArticleByID retrieves an article by its UUID or unique code.
func (r *PostgresRepository) FindArticleByID(ctx context.Context, id string) (*Article, error) {
	query := `
		SELECT
			a.id::text,
			a.client_id::text,
			c.name AS client_name,
			a.code,
			a.title,
			a.category,
			a.description,
			a.created_at,
			a.updated_at
		FROM articles a
		JOIN clients c ON a.client_id = c.id
		WHERE (a.id::text = $1 OR LOWER(a.code) = LOWER($1)) AND a.deleted_at IS NULL
		LIMIT 1;
	`

	var a Article
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.ClientID,
		&a.ClientName,
		&a.Code,
		&a.Title,
		&a.Category,
		&a.Description,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find article by id: %w", err)
	}

	return &a, nil
}

// CreateArticle inserts a new article record into the database.
func (r *PostgresRepository) CreateArticle(ctx context.Context, article *Article) error {
	query := `
		INSERT INTO articles (client_id, code, title, category, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at;
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		article.ClientID,
		article.Code,
		article.Title,
		article.Category,
		article.Description,
	).Scan(&article.ID, &article.CreatedAt, &article.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert article: %w", err)
	}

	clientNameQuery := `SELECT name FROM clients WHERE id::text = $1;`
	_ = r.db.QueryRowContext(ctx, clientNameQuery, article.ClientID).Scan(&article.ClientName)

	return nil
}

// ArticleCodeExists checks whether an article code is already taken.
func (r *PostgresRepository) ArticleCodeExists(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE LOWER(code) = LOWER($1) AND deleted_at IS NULL);`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(code)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check article code exists: %w", err)
	}
	return exists, nil
}

// ListComponentsByArticleID retrieves all components associated with an article ordered by sequence number.
func (r *PostgresRepository) ListComponentsByArticleID(ctx context.Context, articleID string) ([]Component, error) {
	query := `
		SELECT
			id::text,
			article_id::text,
			code,
			name,
			sequence_number,
			standard_sam,
			created_at,
			updated_at
		FROM components
		WHERE article_id::text = $1 AND deleted_at IS NULL
		ORDER BY sequence_number ASC, created_at ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil, fmt.Errorf("query components by article id: %w", err)
	}
	defer rows.Close()

	components := make([]Component, 0)
	for rows.Next() {
		var comp Component
		if err := rows.Scan(
			&comp.ID,
			&comp.ArticleID,
			&comp.Code,
			&comp.Name,
			&comp.SequenceNumber,
			&comp.StandardSAM,
			&comp.CreatedAt,
			&comp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		components = append(components, comp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("components rows iteration: %w", err)
	}

	return components, nil
}

// CreateComponent inserts a new component into the database.
func (r *PostgresRepository) CreateComponent(ctx context.Context, comp *Component) error {
	query := `
		INSERT INTO components (article_id, code, name, sequence_number, standard_sam)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at;
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		comp.ArticleID,
		comp.Code,
		comp.Name,
		comp.SequenceNumber,
		comp.StandardSAM,
	).Scan(&comp.ID, &comp.CreatedAt, &comp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert component: %w", err)
	}

	return nil
}

