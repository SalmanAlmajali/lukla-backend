package ratecard

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

// NewPostgresRepository creates a new PostgresRepository for the ratecard module.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ListRateCards retrieves rate cards matching the given filter.
func (r *PostgresRepository) ListRateCards(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error) {
	conditions := []string{"rc.deleted_at IS NULL"}
	args := make([]interface{}, 0)
	argIdx := 1

	if clientID := strings.TrimSpace(filter.ClientID); clientID != "" {
		conditions = append(conditions, fmt.Sprintf("rc.client_id = $%d", argIdx))
		args = append(args, clientID)
		argIdx++
	}

	if articleID := strings.TrimSpace(filter.ArticleID); articleID != "" {
		conditions = append(conditions, fmt.Sprintf("rc.article_id = $%d", argIdx))
		args = append(args, articleID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT
			rc.id::text,
			rc.client_id::text,
			cl.name,
			rc.article_id::text,
			art.title,
			rc.component_id::text,
			cmp.name,
			rc.sewing_rate,
			rc.cutting_rate,
			rc.effective_date::text,
			rc.notes,
			rc.created_at,
			rc.updated_at
		FROM rate_cards rc
		JOIN clients cl ON rc.client_id = cl.id
		JOIN articles art ON rc.article_id = art.id
		JOIN components cmp ON rc.component_id = cmp.id
		WHERE %s
		ORDER BY cl.name ASC, art.title ASC, cmp.sequence_number ASC;
	`, strings.Join(conditions, " AND "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rate cards: %w", err)
	}
	defer rows.Close()

	rateCards := make([]RateCard, 0)
	for rows.Next() {
		var rc RateCard
		var notes sql.NullString
		if err := rows.Scan(
			&rc.ID,
			&rc.ClientID,
			&rc.ClientName,
			&rc.ArticleID,
			&rc.ArticleTitle,
			&rc.ComponentID,
			&rc.ComponentName,
			&rc.SewingRate,
			&rc.CuttingRate,
			&rc.EffectiveDate,
			&notes,
			&rc.CreatedAt,
			&rc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rate card: %w", err)
		}
		if notes.Valid {
			rc.Notes = &notes.String
		}
		rateCards = append(rateCards, rc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rate cards rows iteration: %w", err)
	}

	return rateCards, nil
}

// FindRateCardByID retrieves a rate card by its UUID.
func (r *PostgresRepository) FindRateCardByID(ctx context.Context, id string) (*RateCard, error) {
	query := `
		SELECT
			rc.id::text,
			rc.client_id::text,
			cl.name,
			rc.article_id::text,
			art.title,
			rc.component_id::text,
			cmp.name,
			rc.sewing_rate,
			rc.cutting_rate,
			rc.effective_date::text,
			rc.notes,
			rc.created_at,
			rc.updated_at
		FROM rate_cards rc
		JOIN clients cl ON rc.client_id = cl.id
		JOIN articles art ON rc.article_id = art.id
		JOIN components cmp ON rc.component_id = cmp.id
		WHERE rc.id = $1 AND rc.deleted_at IS NULL
		LIMIT 1;
	`

	var rc RateCard
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rc.ID,
		&rc.ClientID,
		&rc.ClientName,
		&rc.ArticleID,
		&rc.ArticleTitle,
		&rc.ComponentID,
		&rc.ComponentName,
		&rc.SewingRate,
		&rc.CuttingRate,
		&rc.EffectiveDate,
		&notes,
		&rc.CreatedAt,
		&rc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find rate card by id: %w", err)
	}
	if notes.Valid {
		rc.Notes = &notes.String
	}

	return &rc, nil
}

// FindRateCardByComponent retrieves an existing rate card for a specific article and component.
func (r *PostgresRepository) FindRateCardByComponent(ctx context.Context, articleID, componentID string) (*RateCard, error) {
	query := `
		SELECT
			rc.id::text,
			rc.client_id::text,
			cl.name,
			rc.article_id::text,
			art.title,
			rc.component_id::text,
			cmp.name,
			rc.sewing_rate,
			rc.cutting_rate,
			rc.effective_date::text,
			rc.notes,
			rc.created_at,
			rc.updated_at
		FROM rate_cards rc
		JOIN clients cl ON rc.client_id = cl.id
		JOIN articles art ON rc.article_id = art.id
		JOIN components cmp ON rc.component_id = cmp.id
		WHERE rc.article_id = $1 AND rc.component_id = $2 AND rc.deleted_at IS NULL
		LIMIT 1;
	`

	var rc RateCard
	var notes sql.NullString
	err := r.db.QueryRowContext(ctx, query, articleID, componentID).Scan(
		&rc.ID,
		&rc.ClientID,
		&rc.ClientName,
		&rc.ArticleID,
		&rc.ArticleTitle,
		&rc.ComponentID,
		&rc.ComponentName,
		&rc.SewingRate,
		&rc.CuttingRate,
		&rc.EffectiveDate,
		&notes,
		&rc.CreatedAt,
		&rc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find rate card by component: %w", err)
	}
	if notes.Valid {
		rc.Notes = &notes.String
	}

	return &rc, nil
}

// ValidateComponentHierarchy checks that client, article, and component exist and are properly parented.
func (r *PostgresRepository) ValidateComponentHierarchy(ctx context.Context, clientID, articleID, componentID string) (*ComponentValidationResult, error) {
	var clientName string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM clients WHERE id = $1 AND deleted_at IS NULL LIMIT 1;", clientID).Scan(&clientName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("validate client: %w", err)
	}

	var articleTitle string
	var artClientID string
	err = r.db.QueryRowContext(ctx, "SELECT title, client_id::text FROM articles WHERE id = $1 AND deleted_at IS NULL LIMIT 1;", articleID).Scan(&articleTitle, &artClientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, fmt.Errorf("validate article: %w", err)
	}
	if artClientID != clientID {
		return nil, ErrComponentMismatch
	}

	var componentName string
	var cmpArticleID string
	err = r.db.QueryRowContext(ctx, "SELECT name, article_id::text FROM components WHERE id = $1 AND deleted_at IS NULL LIMIT 1;", componentID).Scan(&componentName, &cmpArticleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrComponentMismatch
		}
		return nil, fmt.Errorf("validate component: %w", err)
	}
	if cmpArticleID != articleID {
		return nil, ErrComponentMismatch
	}

	return &ComponentValidationResult{
		ClientName:    clientName,
		ArticleTitle:  articleTitle,
		ComponentName: componentName,
	}, nil
}

// CreateRateCard persists a new rate card record, and optionally logs its initial audit history.
func (r *PostgresRepository) CreateRateCard(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO rate_cards (client_id, article_id, component_id, sewing_rate, cutting_rate, effective_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, created_at, updated_at;
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		rc.ClientID,
		rc.ArticleID,
		rc.ComponentID,
		rc.SewingRate,
		rc.CuttingRate,
		rc.EffectiveDate,
		rc.Notes,
	).Scan(&rc.ID, &rc.CreatedAt, &rc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert rate card: %w", err)
	}

	if initialHistory != nil {
		histQuery := `
			INSERT INTO rate_card_histories (rate_card_id, old_sewing_rate, new_sewing_rate, old_cutting_rate, new_cutting_rate, changed_by_user_id, reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id::text, created_at;
		`
		err = tx.QueryRowContext(
			ctx,
			histQuery,
			rc.ID,
			initialHistory.OldSewingRate,
			initialHistory.NewSewingRate,
			initialHistory.OldCuttingRate,
			initialHistory.NewCuttingRate,
			initialHistory.ChangedByUserID,
			initialHistory.Reason,
		).Scan(&initialHistory.ID, &initialHistory.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert initial rate card history: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateRateCard updates rate card prices and logs the change to rate_card_histories.
func (r *PostgresRepository) UpdateRateCard(ctx context.Context, rc *RateCard, history *RateCardHistory) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE rate_cards
		SET sewing_rate = $1, cutting_rate = $2, effective_date = $3, updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING updated_at;
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		rc.SewingRate,
		rc.CuttingRate,
		rc.EffectiveDate,
		rc.ID,
	).Scan(&rc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update rate card: %w", err)
	}

	if history != nil {
		histQuery := `
			INSERT INTO rate_card_histories (rate_card_id, old_sewing_rate, new_sewing_rate, old_cutting_rate, new_cutting_rate, changed_by_user_id, reason)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id::text, created_at;
		`
		err = tx.QueryRowContext(
			ctx,
			histQuery,
			rc.ID,
			history.OldSewingRate,
			history.NewSewingRate,
			history.OldCuttingRate,
			history.NewCuttingRate,
			history.ChangedByUserID,
			history.Reason,
		).Scan(&history.ID, &history.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert rate card history: %w", err)
		}
	}

	return tx.Commit()
}

// ListRateCardHistories retrieves audit trail entries for a given rate card.
func (r *PostgresRepository) ListRateCardHistories(ctx context.Context, rateCardID string) ([]RateCardHistory, error) {
	query := `
		SELECT
			rch.id::text,
			rch.rate_card_id::text,
			rch.old_sewing_rate,
			rch.new_sewing_rate,
			rch.old_cutting_rate,
			rch.new_cutting_rate,
			rch.changed_by_user_id::text,
			COALESCE(u.full_name, 'System') as changed_by_name,
			rch.reason,
			rch.created_at
		FROM rate_card_histories rch
		LEFT JOIN users u ON rch.changed_by_user_id = u.id
		WHERE rch.rate_card_id = $1
		ORDER BY rch.created_at DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, rateCardID)
	if err != nil {
		return nil, fmt.Errorf("query rate card histories: %w", err)
	}
	defer rows.Close()

	histories := make([]RateCardHistory, 0)
	for rows.Next() {
		var h RateCardHistory
		var changedByUserID sql.NullString
		if err := rows.Scan(
			&h.ID,
			&h.RateCardID,
			&h.OldSewingRate,
			&h.NewSewingRate,
			&h.OldCuttingRate,
			&h.NewCuttingRate,
			&changedByUserID,
			&h.ChangedByName,
			&h.Reason,
			&h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rate card history: %w", err)
		}
		if changedByUserID.Valid {
			h.ChangedByUserID = &changedByUserID.String
		}
		histories = append(histories, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rate card histories iteration: %w", err)
	}

	return histories, nil
}

