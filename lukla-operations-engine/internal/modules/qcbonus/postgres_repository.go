package qcbonus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// PostgresRepository implements qcbonus.Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository instance.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetBonusConfig retrieves current active bonus rate configuration and history.
func (r *PostgresRepository) GetBonusConfig(ctx context.Context) (*QCBonusConfig, error) {
	query := `
		SELECT 
			c.rate_per_pcs,
			c.currency,
			c.updated_at,
			COALESCE(u.full_name, 'Owner') AS updated_by
		FROM qc_bonus_configs c
		LEFT JOIN users u ON c.updated_by_user_id = u.id
		ORDER BY c.updated_at DESC
		LIMIT 1
	`
	var cfg QCBonusConfig
	err := r.db.QueryRowContext(ctx, query).Scan(
		&cfg.CurrentRatePerPcs,
		&cfg.Currency,
		&cfg.UpdatedAt,
		&cfg.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("query bonus config: %w", err)
	}

	// Fetch history
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			h.rate_per_pcs,
			h.created_at,
			COALESCE(u.full_name, 'Owner') AS changed_by,
			h.notes
		FROM qc_bonus_rate_history h
		LEFT JOIN users u ON h.changed_by_user_id = u.id
		ORDER BY h.created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return nil, fmt.Errorf("query bonus rate history: %w", err)
	}
	defer rows.Close()

	cfg.RateHistory = []QCBonusRateHistory{}
	for rows.Next() {
		var h QCBonusRateHistory
		if err := rows.Scan(
			&h.RatePerPcs,
			&h.EffectiveFrom,
			&h.ChangedBy,
			&h.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan rate history: %w", err)
		}
		cfg.RateHistory = append(cfg.RateHistory, h)
	}

	return &cfg, nil
}

// UpdateBonusRate updates the nominal bonus rate per pcs (exclusive to Owner).
func (r *PostgresRepository) UpdateBonusRate(ctx context.Context, rate float64, ownerID string, notes *string) (*QCBonusConfig, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update bonus rate tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO qc_bonus_configs (rate_per_pcs, currency, updated_by_user_id, notes)
		VALUES ($1, 'IDR', $2, $3)
	`, rate, ownerID, notes)
	if err != nil {
		return nil, fmt.Errorf("insert bonus config: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO qc_bonus_rate_history (rate_per_pcs, changed_by_user_id, notes)
		VALUES ($1, $2, $3)
	`, rate, ownerID, notes)
	if err != nil {
		return nil, fmt.Errorf("insert rate history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit bonus rate tx: %w", err)
	}

	return r.GetBonusConfig(ctx)
}

// ListRecipients retrieves all configured pool bonus recipients and their ratios.
func (r *PostgresRepository) ListRecipients(ctx context.Context) ([]QCBonusRecipient, error) {
	query := `
		SELECT 
			qbr.id,
			qbr.user_id,
			u.full_name,
			r.name AS role_name,
			COALESCE(u.employee_code, '') AS title,
			qbr.is_active
		FROM qc_bonus_recipients qbr
		JOIN users u ON qbr.user_id = u.id
		JOIN roles r ON u.role_id = r.id
		ORDER BY qbr.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list bonus recipients: %w", err)
	}
	defer rows.Close()

	var recipients []QCBonusRecipient
	activeCount := 0
	for rows.Next() {
		var rec QCBonusRecipient
		if err := rows.Scan(
			&rec.ID,
			&rec.UserID,
			&rec.FullName,
			&rec.Role,
			&rec.Title,
			&rec.IsActive,
		); err != nil {
			return nil, fmt.Errorf("scan bonus recipient: %w", err)
		}
		if rec.IsActive {
			activeCount++
		}
		recipients = append(recipients, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipients: %w", err)
	}

	// Calculate share ratio (1 / N)
	for i := range recipients {
		if recipients[i].IsActive && activeCount > 0 {
			recipients[i].ShareRatio = 1.0 / float64(activeCount)
		} else {
			recipients[i].ShareRatio = 0.0
		}
	}

	return recipients, nil
}

// UpdateRecipients updates the list of recipient user IDs (exclusive to Owner).
func (r *PostgresRepository) UpdateRecipients(ctx context.Context, userIDs []string) ([]QCBonusRecipient, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update recipients tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM qc_bonus_recipients`)
	if err != nil {
		return nil, fmt.Errorf("delete old recipients: %w", err)
	}

	for _, uid := range userIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO qc_bonus_recipients (user_id, is_active)
			VALUES ($1, TRUE)
		`, uid)
		if err != nil {
			return nil, fmt.Errorf("insert recipient %s: %w", uid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update recipients tx: %w", err)
	}

	return r.ListRecipients(ctx)
}

// ListPeriods retrieves bonus periods with optional status filter.
func (r *PostgresRepository) ListPeriods(ctx context.Context, status string) ([]QCBonusPeriodSummary, error) {
	query := `
		SELECT 
			id,
			name,
			start_date::TEXT,
			end_date::TEXT,
			status,
			total_passed_pcs,
			rate_per_pcs_applied,
			total_pool_amount,
			recipient_count,
			amount_per_recipient,
			created_at
		FROM qc_bonus_periods
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("list bonus periods: %w", err)
	}
	defer rows.Close()

	var summaries []QCBonusPeriodSummary
	for rows.Next() {
		var s QCBonusPeriodSummary
		if err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.StartDate,
			&s.EndDate,
			&s.Status,
			&s.TotalPassedPcs,
			&s.RatePerPcs,
			&s.TotalPoolAmount,
			&s.RecipientCount,
			&s.AmountPerRecipient,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan period summary: %w", err)
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

// CreatePeriod opens a new bonus period.
func (r *PostgresRepository) CreatePeriod(ctx context.Context, req CreateQCBonusPeriodRequest) (*QCBonusPeriodDetail, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO qc_bonus_periods (name, start_date, end_date, notes)
		VALUES ($1, $2::DATE, $3::DATE, $4)
		RETURNING id
	`, req.Name, req.StartDate, req.EndDate, req.Notes).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert bonus period: %w", err)
	}

	return r.GetPeriodByID(ctx, id)
}

// GetPeriodByID retrieves period details with its distribution list.
func (r *PostgresRepository) GetPeriodByID(ctx context.Context, id string) (*QCBonusPeriodDetail, error) {
	query := `
		SELECT 
			id,
			name,
			start_date::TEXT,
			end_date::TEXT,
			status,
			rate_per_pcs_applied,
			total_passed_pcs,
			total_pool_amount,
			recipient_count,
			amount_per_recipient,
			notes,
			created_at
		FROM qc_bonus_periods
		WHERE id = $1
	`
	var detail QCBonusPeriodDetail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&detail.Name,
		&detail.StartDate,
		&detail.EndDate,
		&detail.Status,
		&detail.RatePerPcsApplied,
		&detail.TotalPassedPcs,
		&detail.TotalPoolAmount,
		&detail.RecipientCount,
		&detail.AmountPerRecipient,
		&detail.Notes,
		&detail.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("get period by id: %w", err)
	}

	// Fetch distributions
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			qbd.user_id,
			u.full_name,
			r.name AS role_name,
			qbd.share_amount,
			qbd.payout_status
		FROM qc_bonus_distributions qbd
		JOIN users u ON qbd.user_id = u.id
		JOIN roles r ON u.role_id = r.id
		WHERE qbd.period_id = $1
		ORDER BY qbd.share_amount DESC, u.full_name ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query distributions: %w", err)
	}
	defer rows.Close()

	detail.Distributions = []QCBonusRecipientShare{}
	for rows.Next() {
		var s QCBonusRecipientShare
		if err := rows.Scan(
			&s.UserID,
			&s.FullName,
			&s.Role,
			&s.ShareAmount,
			&s.PayoutStatus,
		); err != nil {
			return nil, fmt.Errorf("scan distribution: %w", err)
		}
		detail.Distributions = append(detail.Distributions, s)
	}

	return &detail, nil
}

// CalculatePeriod executes pool bonus accumulation & 3-way equal split (Skenario Uji 3).
func (r *PostgresRepository) CalculatePeriod(ctx context.Context, id string) (*QCBonusPeriodDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin calculate tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Lock period row
	var startDate, endDate, status string
	err = tx.QueryRowContext(ctx, `
		SELECT start_date::TEXT, end_date::TEXT, status
		FROM qc_bonus_periods
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&startDate, &endDate, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("lock period: %w", err)
	}

	if status == "LOCKED" {
		return nil, ErrPeriodAlreadyLocked
	}

	// 2. Fetch current active bonus rate per pcs
	var currentRate float64
	err = tx.QueryRowContext(ctx, `
		SELECT rate_per_pcs
		FROM qc_bonus_configs
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&currentRate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			currentRate = 1500.00
		} else {
			return nil, fmt.Errorf("query active rate: %w", err)
		}
	}

	// 3. Count total passed QC pieces during period
	var totalPassedPcs int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(passed_quantity), 0)
		FROM qc_inspections
		WHERE inspection_date >= $1::DATE AND inspection_date <= $2::DATE
	`, startDate, endDate).Scan(&totalPassedPcs)
	if err != nil {
		return nil, fmt.Errorf("count passed pcs: %w", err)
	}

	totalPoolAmount := float64(totalPassedPcs) * currentRate

	// 4. Fetch active recipients (default: 3 orang pengawas)
	type activeRec struct {
		userID string
	}
	recRows, err := tx.QueryContext(ctx, `
		SELECT user_id
		FROM qc_bonus_recipients
		WHERE is_active = TRUE
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query active recipients: %w", err)
	}
	defer recRows.Close()

	var activeRecs []activeRec
	for recRows.Next() {
		var ar activeRec
		if err := recRows.Scan(&ar.userID); err != nil {
			return nil, fmt.Errorf("scan active recipient: %w", err)
		}
		activeRecs = append(activeRecs, ar)
	}
	if len(activeRecs) == 0 {
		return nil, ErrNoActiveRecipients
	}

	recipientCount := len(activeRecs)

	// 5. Calculate Equal Split with Penny Allocation (Skenario Uji 3: zero remainder loss)
	totalCents := int64(math.Round(totalPoolAmount * 100))
	baseCents := totalCents / int64(recipientCount)
	remainderCents := totalCents % int64(recipientCount)

	// 6. Delete old distributions and insert calculated shares
	_, err = tx.ExecContext(ctx, `DELETE FROM qc_bonus_distributions WHERE period_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("clear old distributions: %w", err)
	}

	for i, ar := range activeRecs {
		cents := baseCents
		if int64(i) < remainderCents {
			cents++
		}
		shareAmount := float64(cents) / 100.0

		_, err = tx.ExecContext(ctx, `
			INSERT INTO qc_bonus_distributions (period_id, user_id, share_amount, payout_status)
			VALUES ($1, $2, $3, 'PENDING_DISBURSEMENT')
		`, id, ar.userID, shareAmount)
		if err != nil {
			return nil, fmt.Errorf("insert distribution for user %s: %w", ar.userID, err)
		}
	}

	amountPerRecipient := 0.0
	if recipientCount > 0 {
		amountPerRecipient = totalPoolAmount / float64(recipientCount)
	}

	// 7. Update period to CALCULATED
	_, err = tx.ExecContext(ctx, `
		UPDATE qc_bonus_periods
		SET status = 'CALCULATED',
			rate_per_pcs_applied = $1,
			total_passed_pcs = $2,
			total_pool_amount = $3,
			recipient_count = $4,
			amount_per_recipient = $5,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`, currentRate, totalPassedPcs, totalPoolAmount, recipientCount, amountPerRecipient, id)
	if err != nil {
		return nil, fmt.Errorf("update period status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit calculate tx: %w", err)
	}

	return r.GetPeriodByID(ctx, id)
}

// LockPeriod finalizes and locks the bonus period so it cannot be altered.
func (r *PostgresRepository) LockPeriod(ctx context.Context, id string, userID string, notes *string) (*QCBonusPeriodDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin lock tx: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT status
		FROM qc_bonus_periods
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("lock period check: %w", err)
	}

	if status == "LOCKED" {
		return nil, ErrPeriodAlreadyLocked
	}

	// If period is in DRAFT status, calculate it first inside this transaction
	if status == "DRAFT" {
		_ = tx.Rollback()
		_, err := r.CalculatePeriod(ctx, id)
		if err != nil {
			return nil, err
		}
		tx, err = r.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE qc_bonus_periods
		SET status = 'LOCKED',
			locked_at = CURRENT_TIMESTAMP,
			locked_by_user_id = $2,
			notes = COALESCE($3, notes),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, userID, notes)
	if err != nil {
		return nil, fmt.Errorf("update lock status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lock tx: %w", err)
	}

	return r.GetPeriodByID(ctx, id)
}
