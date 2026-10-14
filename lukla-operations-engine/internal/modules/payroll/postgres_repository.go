package payroll

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreatePeriod(ctx context.Context, p *PayrollPeriod) error {
	query := `
		INSERT INTO payroll_periods (
			period_code, start_date, end_date, meal_allowance_percentage, status, notes
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(
		ctx,
		query,
		p.PeriodCode,
		p.StartDate,
		p.EndDate,
		p.MealAllowancePercentage,
		p.Status,
		p.Notes,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique") {
			return ErrPeriodCodeExists
		}
		return fmt.Errorf("insert payroll period: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindPeriodByID(ctx context.Context, id string) (*PayrollPeriod, error) {
	query := `
		SELECT 
			id, period_code, start_date::TEXT, end_date::TEXT, meal_allowance_percentage,
			status, total_operators_count, total_gross_piece_rate, total_meal_allowance,
			total_adjustments, total_net_payable, locked_at, locked_by_user_id, notes,
			created_at, updated_at
		FROM payroll_periods
		WHERE id = $1
	`
	var p PayrollPeriod
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.PeriodCode,
		&p.StartDate,
		&p.EndDate,
		&p.MealAllowancePercentage,
		&p.Status,
		&p.TotalOperatorsCount,
		&p.TotalGrossPieceRate,
		&p.TotalMealAllowance,
		&p.TotalAdjustments,
		&p.TotalNetPayable,
		&p.LockedAt,
		&p.LockedByUserID,
		&p.Notes,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("find payroll period by id: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) FindPeriodByCode(ctx context.Context, code string) (*PayrollPeriod, error) {
	query := `
		SELECT 
			id, period_code, start_date::TEXT, end_date::TEXT, meal_allowance_percentage,
			status, total_operators_count, total_gross_piece_rate, total_meal_allowance,
			total_adjustments, total_net_payable, locked_at, locked_by_user_id, notes,
			created_at, updated_at
		FROM payroll_periods
		WHERE period_code = $1
	`
	var p PayrollPeriod
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&p.ID,
		&p.PeriodCode,
		&p.StartDate,
		&p.EndDate,
		&p.MealAllowancePercentage,
		&p.Status,
		&p.TotalOperatorsCount,
		&p.TotalGrossPieceRate,
		&p.TotalMealAllowance,
		&p.TotalAdjustments,
		&p.TotalNetPayable,
		&p.LockedAt,
		&p.LockedByUserID,
		&p.Notes,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("find payroll period by code: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) ListPeriods(ctx context.Context, filter ListPeriodsFilter) ([]PayrollPeriod, int, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM payroll_periods WHERE %s", strings.Join(whereClauses, " AND "))
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count payroll periods: %w", err)
	}

	limit := 20
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT 
			id, period_code, start_date::TEXT, end_date::TEXT, meal_allowance_percentage,
			status, total_operators_count, total_gross_piece_rate, total_meal_allowance,
			total_adjustments, total_net_payable, locked_at, locked_by_user_id, notes,
			created_at, updated_at
		FROM payroll_periods
		WHERE %s
		ORDER BY start_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(whereClauses, " AND "), argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list payroll periods: %w", err)
	}
	defer rows.Close()

	periods := []PayrollPeriod{}
	for rows.Next() {
		var p PayrollPeriod
		if err := rows.Scan(
			&p.ID,
			&p.PeriodCode,
			&p.StartDate,
			&p.EndDate,
			&p.MealAllowancePercentage,
			&p.Status,
			&p.TotalOperatorsCount,
			&p.TotalGrossPieceRate,
			&p.TotalMealAllowance,
			&p.TotalAdjustments,
			&p.TotalNetPayable,
			&p.LockedAt,
			&p.LockedByUserID,
			&p.Notes,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan payroll period: %w", err)
		}
		periods = append(periods, p)
	}

	return periods, total, nil
}

func (r *PostgresRepository) UpdatePeriodTotalsAndStatus(
	ctx context.Context,
	periodID string,
	status string,
	opsCount int,
	gross, meal, adj, net float64,
) error {
	query := `
		UPDATE payroll_periods
		SET status = $1,
			total_operators_count = $2,
			total_gross_piece_rate = $3,
			total_meal_allowance = $4,
			total_adjustments = $5,
			total_net_payable = $6,
			updated_at = NOW()
		WHERE id = $7
	`
	res, err := r.db.ExecContext(ctx, query, status, opsCount, gross, meal, adj, net, periodID)
	if err != nil {
		return fmt.Errorf("update period totals and status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrPeriodNotFound
	}
	return nil
}

func (r *PostgresRepository) GetEligibleOperatorOutputs(
	ctx context.Context,
	startDate, endDate string,
	operatorIDs []string,
) ([]OperatorOutputAggregation, error) {
	whereClauses := []string{
		"po.work_date >= $1::DATE",
		"po.work_date <= $2::DATE",
		"op.status = 'ACTIVE'",
		"po.work_date >= op.join_date", // Skenario Uji 2: operator joining mid-week only gets credit for output on or after join_date
	}
	args := []interface{}{startDate, endDate}
	argIdx := 3

	if len(operatorIDs) > 0 {
		placeholders := make([]string, len(operatorIDs))
		for i, opID := range operatorIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, opID)
			argIdx++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("po.operator_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	query := fmt.Sprintf(`
		SELECT 
			po.operator_id,
			op.operator_code,
			op.name AS operator_name,
			op.join_date::TEXT AS operator_join_date,
			po.order_id,
			o.order_number AS order_code,
			c.name AS client_name,
			po.component_id,
			comp.name AS component_name,
			po.work_date::TEXT AS work_date,
			SUM(po.quantity) AS total_qty,
			po.snapshot_sewing_rate,
			SUM(po.quantity * po.snapshot_sewing_rate) AS subtotal
		FROM production_outputs po
		JOIN operators op ON po.operator_id = op.id
		JOIN orders o ON po.order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN components comp ON po.component_id = comp.id
		WHERE %s
		GROUP BY 
			po.operator_id, op.operator_code, op.name, op.join_date,
			po.order_id, o.order_number, c.name,
			po.component_id, comp.name,
			po.work_date, po.snapshot_sewing_rate
		ORDER BY op.name ASC, po.work_date ASC, comp.name ASC
	`, strings.Join(whereClauses, " AND "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query eligible operator outputs: %w", err)
	}
	defer rows.Close()

	aggregations := []OperatorOutputAggregation{}
	for rows.Next() {
		var a OperatorOutputAggregation
		if err := rows.Scan(
			&a.OperatorID,
			&a.OperatorCode,
			&a.OperatorName,
			&a.OperatorJoinDate,
			&a.OrderID,
			&a.OrderCode,
			&a.ClientName,
			&a.ComponentID,
			&a.ComponentName,
			&a.WorkDate,
			&a.Quantity,
			&a.SnapshotSewingRate,
			&a.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("scan operator output aggregation: %w", err)
		}
		aggregations = append(aggregations, a)
	}

	return aggregations, nil
}

func (r *PostgresRepository) SaveCalculationResults(
	ctx context.Context,
	periodID string,
	slips []PayrollSlip,
	itemsMap map[string][]PayrollSlipItem,
	totalOps int,
	totalGross, totalMeal, totalAdj, totalNet float64,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin calculation tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Lock period and verify not LOCKED
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM payroll_periods WHERE id = $1 FOR UPDATE", periodID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPeriodNotFound
		}
		return fmt.Errorf("lock period row: %w", err)
	}
	if currentStatus == string(PeriodStatusLocked) {
		return ErrPeriodAlreadyLocked
	}

	// 2. Clear old slips and cascaded items if recalculating
	_, err = tx.ExecContext(ctx, "DELETE FROM payroll_slips WHERE payroll_period_id = $1", periodID)
	if err != nil {
		return fmt.Errorf("clear previous slips: %w", err)
	}

	// 3. Insert slips and items
	slipInsertQuery := `
		INSERT INTO payroll_slips (
			payroll_period_id, operator_id, total_pieces, gross_piece_rate_total,
			meal_allowance_amount, adjustments_total, net_payable_total, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	itemInsertQuery := `
		INSERT INTO payroll_slip_items (
			payroll_slip_id, order_id, component_id, work_date,
			quantity, snapshot_sewing_rate, subtotal
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, slip := range slips {
		var slipID string
		err := tx.QueryRowContext(
			ctx,
			slipInsertQuery,
			periodID,
			slip.OperatorID,
			slip.TotalPieces,
			slip.GrossPieceRateTotal,
			slip.MealAllowanceAmount,
			slip.AdjustmentsTotal,
			slip.NetPayableTotal,
			slip.Status,
		).Scan(&slipID)
		if err != nil {
			return fmt.Errorf("insert slip for operator %s: %w", slip.OperatorID, err)
		}

		items := itemsMap[slip.OperatorID]
		for _, item := range items {
			_, err = tx.ExecContext(
				ctx,
				itemInsertQuery,
				slipID,
				item.OrderID,
				item.ComponentID,
				item.WorkDate,
				item.Quantity,
				item.SnapshotSewingRate,
				item.Subtotal,
			)
			if err != nil {
				return fmt.Errorf("insert slip item: %w", err)
			}
		}
	}

	// 4. Update period totals and status
	periodUpdateQuery := `
		UPDATE payroll_periods
		SET status = 'CALCULATED',
			total_operators_count = $1,
			total_gross_piece_rate = $2,
			total_meal_allowance = $3,
			total_adjustments = $4,
			total_net_payable = $5,
			updated_at = NOW()
		WHERE id = $6
	`
	_, err = tx.ExecContext(ctx, periodUpdateQuery, totalOps, totalGross, totalMeal, totalAdj, totalNet, periodID)
	if err != nil {
		return fmt.Errorf("update period totals: %w", err)
	}

	return tx.Commit()
}

func (r *PostgresRepository) ListSlipsByPeriod(ctx context.Context, periodID string, filter ListSlipsFilter) ([]PayrollSlip, int, error) {
	whereClauses := []string{"ps.payroll_period_id = $1"}
	args := []interface{}{periodID}
	argIdx := 2

	if filter.OperatorID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("ps.operator_id = $%d", argIdx))
		args = append(args, filter.OperatorID)
		argIdx++
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(op.name ILIKE $%d OR op.operator_code ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM payroll_slips ps
		JOIN operators op ON ps.operator_id = op.id
		WHERE %s
	`, strings.Join(whereClauses, " AND "))

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count slips: %w", err)
	}

	limit := 50
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT 
			ps.id, ps.payroll_period_id, pp.period_code, ps.operator_id, op.operator_code,
			op.name AS operator_name, op.join_date::TEXT AS operator_join_date,
			ps.total_pieces, ps.gross_piece_rate_total, ps.meal_allowance_amount,
			ps.adjustments_total, ps.net_payable_total, ps.status, ps.created_at, ps.updated_at
		FROM payroll_slips ps
		JOIN payroll_periods pp ON ps.payroll_period_id = pp.id
		JOIN operators op ON ps.operator_id = op.id
		WHERE %s
		ORDER BY op.name ASC
		LIMIT $%d OFFSET $%d
	`, strings.Join(whereClauses, " AND "), argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query slips: %w", err)
	}
	defer rows.Close()

	slips := []PayrollSlip{}
	for rows.Next() {
		var s PayrollSlip
		if err := rows.Scan(
			&s.ID,
			&s.PayrollPeriodID,
			&s.PeriodCode,
			&s.OperatorID,
			&s.OperatorCode,
			&s.OperatorName,
			&s.OperatorJoinDate,
			&s.TotalPieces,
			&s.GrossPieceRateTotal,
			&s.MealAllowanceAmount,
			&s.AdjustmentsTotal,
			&s.NetPayableTotal,
			&s.Status,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan slip: %w", err)
		}
		slips = append(slips, s)
	}

	return slips, total, nil
}

func (r *PostgresRepository) FindSlipByID(ctx context.Context, slipID string) (*PayrollSlipDetail, error) {
	query := `
		SELECT 
			ps.id, ps.payroll_period_id, pp.period_code, ps.operator_id, op.operator_code,
			op.name AS operator_name, op.join_date::TEXT AS operator_join_date,
			ps.total_pieces, ps.gross_piece_rate_total, ps.meal_allowance_amount,
			ps.adjustments_total, ps.net_payable_total, ps.status, ps.created_at, ps.updated_at,
			pp.meal_allowance_percentage
		FROM payroll_slips ps
		JOIN payroll_periods pp ON ps.payroll_period_id = pp.id
		JOIN operators op ON ps.operator_id = op.id
		WHERE ps.id = $1
	`
	var detail PayrollSlipDetail
	err := r.db.QueryRowContext(ctx, query, slipID).Scan(
		&detail.ID,
		&detail.PayrollPeriodID,
		&detail.PeriodCode,
		&detail.OperatorID,
		&detail.OperatorCode,
		&detail.OperatorName,
		&detail.OperatorJoinDate,
		&detail.TotalPieces,
		&detail.GrossPieceRateTotal,
		&detail.MealAllowanceAmount,
		&detail.AdjustmentsTotal,
		&detail.NetPayableTotal,
		&detail.Status,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.MealAllowancePercentage,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlipNotFound
		}
		return nil, fmt.Errorf("find slip by id: %w", err)
	}

	itemsQuery := `
		SELECT 
			psi.id, psi.payroll_slip_id, psi.order_id, o.order_number AS order_code, c.name AS client_name,
			psi.component_id, comp.name AS component_name, psi.work_date::TEXT AS work_date,
			psi.quantity, psi.snapshot_sewing_rate, psi.subtotal, psi.created_at
		FROM payroll_slip_items psi
		JOIN orders o ON psi.order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN components comp ON psi.component_id = comp.id
		WHERE psi.payroll_slip_id = $1
		ORDER BY psi.work_date ASC, comp.name ASC
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, slipID)
	if err != nil {
		return nil, fmt.Errorf("query slip items: %w", err)
	}
	defer rows.Close()

	items := []PayrollSlipItem{}
	for rows.Next() {
		var it PayrollSlipItem
		if err := rows.Scan(
			&it.ID,
			&it.PayrollSlipID,
			&it.OrderID,
			&it.OrderCode,
			&it.ClientName,
			&it.ComponentID,
			&it.ComponentName,
			&it.WorkDate,
			&it.Quantity,
			&it.SnapshotSewingRate,
			&it.Subtotal,
			&it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan slip item: %w", err)
		}
		items = append(items, it)
	}
	detail.Items = items

	return &detail, nil
}

func (r *PostgresRepository) LockPeriod(ctx context.Context, periodID string, lockedByUserID string, notes *string) (*PayrollPeriod, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin lock tx: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	var currentNotes *string
	err = tx.QueryRowContext(ctx, "SELECT status, notes FROM payroll_periods WHERE id = $1 FOR UPDATE", periodID).Scan(&currentStatus, &currentNotes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPeriodNotFound
		}
		return nil, fmt.Errorf("lock period query: %w", err)
	}

	if currentStatus == string(PeriodStatusLocked) {
		return nil, ErrPeriodAlreadyLocked
	}
	if currentStatus == string(PeriodStatusDraft) {
		return nil, ErrPeriodMustBeCalculatedToLock
	}

	finalNotes := currentNotes
	if notes != nil && *notes != "" {
		finalNotes = notes
	}

	updatePeriodQuery := `
		UPDATE payroll_periods
		SET status = 'LOCKED',
			locked_at = NOW(),
			locked_by_user_id = $1,
			notes = $2,
			updated_at = NOW()
		WHERE id = $3
	`
	_, err = tx.ExecContext(ctx, updatePeriodQuery, lockedByUserID, finalNotes, periodID)
	if err != nil {
		return nil, fmt.Errorf("update period to locked: %w", err)
	}

	// Update slips to LOCKED status
	_, err = tx.ExecContext(ctx, "UPDATE payroll_slips SET status = 'LOCKED', updated_at = NOW() WHERE payroll_period_id = $1", periodID)
	if err != nil {
		return nil, fmt.Errorf("update slips to locked: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lock tx: %w", err)
	}

	return r.FindPeriodByID(ctx, periodID)
}

func (r *PostgresRepository) CreateAdjustment(ctx context.Context, periodID string, adj *PayrollAdjustment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin adjustment tx: %w", err)
	}
	defer tx.Rollback()

	// Verify period exists
	var pID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM payroll_periods WHERE id = $1 FOR UPDATE", periodID).Scan(&pID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPeriodNotFound
		}
		return fmt.Errorf("query period for adjustment: %w", err)
	}

	// Verify slip exists for operator in this period
	var slipID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM payroll_slips WHERE payroll_period_id = $1 AND operator_id = $2 FOR UPDATE", periodID, adj.OperatorID).Scan(&slipID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSlipNotFound
		}
		return fmt.Errorf("query slip for adjustment: %w", err)
	}

	// Insert adjustment
	insertAdjQuery := `
		INSERT INTO payroll_adjustments (
			payroll_period_id, operator_id, adjustment_type, amount, reason, created_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err = tx.QueryRowContext(
		ctx,
		insertAdjQuery,
		periodID,
		adj.OperatorID,
		adj.AdjustmentType,
		adj.Amount,
		adj.Reason,
		adj.CreatedByUserID,
	).Scan(&adj.ID, &adj.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert adjustment: %w", err)
	}

	// Update slip totals
	updateSlipQuery := `
		UPDATE payroll_slips
		SET adjustments_total = adjustments_total + $1,
			net_payable_total = net_payable_total + $1,
			updated_at = NOW()
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updateSlipQuery, adj.Amount, slipID)
	if err != nil {
		return fmt.Errorf("update slip adjustments: %w", err)
	}

	// Update period totals
	updatePeriodQuery := `
		UPDATE payroll_periods
		SET total_adjustments = total_adjustments + $1,
			total_net_payable = total_net_payable + $1,
			updated_at = NOW()
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, updatePeriodQuery, adj.Amount, periodID)
	if err != nil {
		return fmt.Errorf("update period adjustments: %w", err)
	}

	return tx.Commit()
}

func (r *PostgresRepository) ListAdjustmentsByPeriod(ctx context.Context, periodID string) ([]PayrollAdjustment, error) {
	// Verify period exists
	if _, err := r.FindPeriodByID(ctx, periodID); err != nil {
		return nil, err
	}

	query := `
		SELECT 
			pa.id, pa.payroll_period_id, pa.operator_id, op.name AS operator_name,
			pa.adjustment_type, pa.amount, pa.reason, pa.created_by_user_id,
			COALESCE(u.full_name, 'System') AS created_by_name, pa.created_at
		FROM payroll_adjustments pa
		JOIN operators op ON pa.operator_id = op.id
		LEFT JOIN users u ON pa.created_by_user_id = u.id
		WHERE pa.payroll_period_id = $1
		ORDER BY pa.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, periodID)
	if err != nil {
		return nil, fmt.Errorf("list adjustments: %w", err)
	}
	defer rows.Close()

	adjustments := []PayrollAdjustment{}
	for rows.Next() {
		var a PayrollAdjustment
		if err := rows.Scan(
			&a.ID,
			&a.PayrollPeriodID,
			&a.OperatorID,
			&a.OperatorName,
			&a.AdjustmentType,
			&a.Amount,
			&a.Reason,
			&a.CreatedByUserID,
			&a.CreatedByName,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan adjustment: %w", err)
		}
		adjustments = append(adjustments, a)
	}
	return adjustments, nil
}

func (r *PostgresRepository) GetPayrollSummaryReport(ctx context.Context, periodID string) (*PayrollSummaryReport, error) {
	period, err := r.FindPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}

	// 1. Total pieces produced in period
	var totalPieces int
	_ = r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(total_pieces), 0) FROM payroll_slips WHERE payroll_period_id = $1", periodID).Scan(&totalPieces)

	// 2. Lines summary
	lineQuery := `
		SELECT 
			l.id AS line_id,
			l.code AS line_code,
			l.name AS line_name,
			COUNT(DISTINCT po.operator_id) AS operators_count,
			COALESCE(SUM(po.quantity), 0) AS pieces_produced,
			COALESCE(SUM(po.quantity * po.snapshot_sewing_rate), 0.00) AS gross_amount
		FROM lines l
		LEFT JOIN production_outputs po ON po.line_id = l.id 
			AND po.work_date >= $1::DATE 
			AND po.work_date <= $2::DATE
		GROUP BY l.id, l.code, l.name
		ORDER BY l.code ASC
	`
	rows, err := r.db.QueryContext(ctx, lineQuery, period.StartDate, period.EndDate)
	if err != nil {
		return nil, fmt.Errorf("query lines summary: %w", err)
	}
	defer rows.Close()

	linesSummary := []LinePayrollSummary{}
	for rows.Next() {
		var ls LinePayrollSummary
		if err := rows.Scan(
			&ls.LineID,
			&ls.LineCode,
			&ls.LineName,
			&ls.OperatorsCount,
			&ls.PiecesProduced,
			&ls.GrossAmount,
		); err != nil {
			return nil, fmt.Errorf("scan line summary: %w", err)
		}
		linesSummary = append(linesSummary, ls)
	}

	// 3. Operators summary slips
	slips, _, err := r.ListSlipsByPeriod(ctx, periodID, ListSlipsFilter{Limit: 100})
	if err != nil {
		return nil, err
	}

	report := &PayrollSummaryReport{
		PeriodID:                period.ID,
		PeriodCode:              period.PeriodCode,
		StartDate:               period.StartDate,
		EndDate:                 period.EndDate,
		MealAllowancePercentage: period.MealAllowancePercentage,
		Status:                  period.Status,
		TotalOperators:          period.TotalOperatorsCount,
		TotalPiecesProduced:     totalPieces,
		TotalGrossPieceRate:     period.TotalGrossPieceRate,
		TotalMealAllowance:      period.TotalMealAllowance,
		TotalAdjustments:        period.TotalAdjustments,
		GrandTotalPayable:       period.TotalNetPayable,
		LinesSummary:            linesSummary,
		OperatorsSummary:        slips,
	}

	return report, nil
}
