package production

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository initializes a new production PostgreSQL repository.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// ListLines retrieves all production lines along with their active operator counts.
func (r *postgresRepository) ListLines(ctx context.Context) ([]ProductionLine, error) {
	query := `
		SELECT l.id, l.code, l.name, l.max_capacity,
		       COALESCE(COUNT(a.id) FILTER (WHERE a.status = 'ACTIVE'), 0) AS active_operators_count,
		       l.supervisor_id, u.full_name AS supervisor_name,
		       l.status, l.created_at, l.updated_at, l.deleted_at
		FROM lines l
		LEFT JOIN users u ON l.supervisor_id = u.id
		LEFT JOIN line_allocations a ON l.id = a.line_id AND a.status = 'ACTIVE'
		WHERE l.deleted_at IS NULL
		GROUP BY l.id, u.full_name
		ORDER BY l.code ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query lines: %w", err)
	}
	defer rows.Close()

	var lines []ProductionLine
	for rows.Next() {
		var l ProductionLine
		var supervisorID, supervisorName sql.NullString
		if err := rows.Scan(
			&l.ID,
			&l.Code,
			&l.Name,
			&l.MaxCapacity,
			&l.ActiveOperatorsCount,
			&supervisorID,
			&supervisorName,
			&l.Status,
			&l.CreatedAt,
			&l.UpdatedAt,
			&l.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}
		if supervisorID.Valid {
			l.SupervisorID = &supervisorID.String
		}
		if supervisorName.Valid {
			l.SupervisorName = &supervisorName.String
		}
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []ProductionLine{}
	}
	return lines, rows.Err()
}

// FindLineByID retrieves detailed information of a production line including active allocations and orders.
func (r *postgresRepository) FindLineByID(ctx context.Context, id string) (*ProductionLineDetail, error) {
	queryLine := `
		SELECT l.id, l.code, l.name, l.max_capacity,
		       COALESCE(COUNT(a.id) FILTER (WHERE a.status = 'ACTIVE'), 0) AS active_operators_count,
		       l.supervisor_id, u.full_name AS supervisor_name,
		       l.status, l.created_at, l.updated_at, l.deleted_at
		FROM lines l
		LEFT JOIN users u ON l.supervisor_id = u.id
		LEFT JOIN line_allocations a ON l.id = a.line_id AND a.status = 'ACTIVE'
		WHERE l.id = $1 AND l.deleted_at IS NULL
		GROUP BY l.id, u.full_name
	`
	var detail ProductionLineDetail
	var supervisorID, supervisorName sql.NullString
	err := r.db.QueryRowContext(ctx, queryLine, id).Scan(
		&detail.ID,
		&detail.Code,
		&detail.Name,
		&detail.MaxCapacity,
		&detail.ActiveOperatorsCount,
		&supervisorID,
		&supervisorName,
		&detail.Status,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.DeletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find line by id: %w", err)
	}
	if supervisorID.Valid {
		detail.SupervisorID = &supervisorID.String
	}
	if supervisorName.Valid {
		detail.SupervisorName = &supervisorName.String
	}

	// 2. Fetch current active allocations
	queryAllocs := `
		SELECT a.id, a.line_id, l.code AS line_code, a.order_id, o.order_number AS order_code,
		       a.operator_id, op.operator_code AS operator_code, op.name AS operator_name, op.skill_tier,
		       a.assigned_component_id, c.name AS assigned_component_name,
		       a.status, a.assigned_at, a.completed_at
		FROM line_allocations a
		JOIN lines l ON a.line_id = l.id
		JOIN orders o ON a.order_id = o.id
		JOIN operators op ON a.operator_id = op.id
		LEFT JOIN components c ON a.assigned_component_id = c.id
		WHERE a.line_id = $1 AND a.status = 'ACTIVE'
		ORDER BY a.assigned_at DESC
	`
	allocRows, err := r.db.QueryContext(ctx, queryAllocs, id)
	if err != nil {
		return nil, fmt.Errorf("query line active allocations: %w", err)
	}
	defer allocRows.Close()

	detail.CurrentAllocations = []LineAllocation{}
	for allocRows.Next() {
		var alloc LineAllocation
		var compID, compName sql.NullString
		if err := allocRows.Scan(
			&alloc.ID,
			&alloc.LineID,
			&alloc.LineCode,
			&alloc.OrderID,
			&alloc.OrderCode,
			&alloc.OperatorID,
			&alloc.OperatorCode,
			&alloc.OperatorName,
			&alloc.SkillTier,
			&compID,
			&compName,
			&alloc.Status,
			&alloc.AssignedAt,
			&alloc.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line allocation: %w", err)
		}
		if compID.Valid {
			alloc.AssignedComponentID = &compID.String
		}
		if compName.Valid {
			alloc.AssignedComponentName = &compName.String
		}
		detail.CurrentAllocations = append(detail.CurrentAllocations, alloc)
	}

	// 3. Fetch active orders running on this line
	queryOrders := `
		SELECT DISTINCT o.id, o.order_number, cl.name AS client_name, ar.title AS article_title
		FROM line_allocations a
		JOIN orders o ON a.order_id = o.id
		JOIN clients cl ON o.client_id = cl.id
		JOIN articles ar ON o.article_id = ar.id
		WHERE a.line_id = $1 AND a.status = 'ACTIVE'
		ORDER BY o.order_number ASC
	`
	orderRows, err := r.db.QueryContext(ctx, queryOrders, id)
	if err != nil {
		return nil, fmt.Errorf("query line active orders: %w", err)
	}
	defer orderRows.Close()

	detail.ActiveOrders = []LineActiveOrder{}
	for orderRows.Next() {
		var ord LineActiveOrder
		if err := orderRows.Scan(&ord.OrderID, &ord.OrderCode, &ord.ClientName, &ord.ArticleTitle); err != nil {
			return nil, fmt.Errorf("scan line active order: %w", err)
		}
		detail.ActiveOrders = append(detail.ActiveOrders, ord)
	}

	return &detail, nil
}

// FindLineByCode finds a line by its unique code.
func (r *postgresRepository) FindLineByCode(ctx context.Context, code string) (*ProductionLine, error) {
	query := `
		SELECT id, code, name, max_capacity, supervisor_id, status, created_at, updated_at
		FROM lines
		WHERE code = $1 AND deleted_at IS NULL
	`
	var l ProductionLine
	var supervisorID sql.NullString
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&l.ID,
		&l.Code,
		&l.Name,
		&l.MaxCapacity,
		&supervisorID,
		&l.Status,
		&l.CreatedAt,
		&l.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find line by code: %w", err)
	}
	if supervisorID.Valid {
		l.SupervisorID = &supervisorID.String
	}
	return &l, nil
}

// CreateLine inserts a new production line.
func (r *postgresRepository) CreateLine(ctx context.Context, line *ProductionLine) error {
	query := `
		INSERT INTO lines (code, name, max_capacity, supervisor_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	var supervisorID *string
	if line.SupervisorID != nil && *line.SupervisorID != "" {
		supervisorID = line.SupervisorID
	}
	return r.db.QueryRowContext(
		ctx,
		query,
		line.Code,
		line.Name,
		line.MaxCapacity,
		supervisorID,
		line.Status,
	).Scan(&line.ID, &line.CreatedAt, &line.UpdatedAt)
}

// UpdateLine updates line attributes.
func (r *postgresRepository) UpdateLine(ctx context.Context, line *ProductionLine) error {
	query := `
		UPDATE lines
		SET name = $1, max_capacity = $2, supervisor_id = $3, status = $4, updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING updated_at
	`
	var supervisorID *string
	if line.SupervisorID != nil && *line.SupervisorID != "" {
		supervisorID = line.SupervisorID
	}
	res, err := r.db.ExecContext(ctx, query, line.Name, line.MaxCapacity, supervisorID, line.Status, line.ID)
	if err != nil {
		return fmt.Errorf("update line: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrLineNotFound
	}
	return nil
}

// GetTotalFactoryCapacity calculates total active machine capacity across all lines except excludeLineID.
func (r *postgresRepository) GetTotalFactoryCapacity(ctx context.Context, excludeLineID string) (int, error) {
	query := `
		SELECT COALESCE(SUM(max_capacity), 0)
		FROM lines
		WHERE deleted_at IS NULL AND status = 'ACTIVE'
	`
	var total int
	if excludeLineID != "" {
		query += " AND id != $1::uuid"
		err := r.db.QueryRowContext(ctx, query, excludeLineID).Scan(&total)
		return total, err
	}
	err := r.db.QueryRowContext(ctx, query).Scan(&total)
	return total, err
}

// ListAllocations lists operator allocations based on filtering criteria.
func (r *postgresRepository) ListAllocations(ctx context.Context, filter ListAllocationsFilter) ([]LineAllocation, error) {
	query := `
		SELECT a.id, a.line_id, l.code AS line_code, a.order_id, o.order_number AS order_code,
		       a.operator_id, op.operator_code AS operator_code, op.name AS operator_name, op.skill_tier,
		       a.assigned_component_id, c.name AS assigned_component_name,
		       a.status, a.assigned_at, a.completed_at
		FROM line_allocations a
		JOIN lines l ON a.line_id = l.id
		JOIN orders o ON a.order_id = o.id
		JOIN operators op ON a.operator_id = op.id
		LEFT JOIN components c ON a.assigned_component_id = c.id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if filter.LineID != "" {
		query += fmt.Sprintf(" AND a.line_id = $%d", argIdx)
		args = append(args, filter.LineID)
		argIdx++
	}
	if filter.OrderID != "" {
		query += fmt.Sprintf(" AND a.order_id = $%d", argIdx)
		args = append(args, filter.OrderID)
		argIdx++
	}
	if filter.OperatorID != "" {
		query += fmt.Sprintf(" AND a.operator_id = $%d", argIdx)
		args = append(args, filter.OperatorID)
		argIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND a.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	query += " ORDER BY a.assigned_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list allocations: %w", err)
	}
	defer rows.Close()

	var allocs []LineAllocation
	for rows.Next() {
		var alloc LineAllocation
		var compID, compName sql.NullString
		if err := rows.Scan(
			&alloc.ID,
			&alloc.LineID,
			&alloc.LineCode,
			&alloc.OrderID,
			&alloc.OrderCode,
			&alloc.OperatorID,
			&alloc.OperatorCode,
			&alloc.OperatorName,
			&alloc.SkillTier,
			&compID,
			&compName,
			&alloc.Status,
			&alloc.AssignedAt,
			&alloc.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan allocation: %w", err)
		}
		if compID.Valid {
			alloc.AssignedComponentID = &compID.String
		}
		if compName.Valid {
			alloc.AssignedComponentName = &compName.String
		}
		allocs = append(allocs, alloc)
	}
	if allocs == nil {
		allocs = []LineAllocation{}
	}
	return allocs, rows.Err()
}

// FindAllocationByID finds a single allocation by ID.
func (r *postgresRepository) FindAllocationByID(ctx context.Context, id string) (*LineAllocation, error) {
	query := `
		SELECT a.id, a.line_id, l.code AS line_code, a.order_id, o.order_number AS order_code,
		       a.operator_id, op.operator_code AS operator_code, op.name AS operator_name, op.skill_tier,
		       a.assigned_component_id, c.name AS assigned_component_name,
		       a.status, a.assigned_at, a.completed_at
		FROM line_allocations a
		JOIN lines l ON a.line_id = l.id
		JOIN orders o ON a.order_id = o.id
		JOIN operators op ON a.operator_id = op.id
		LEFT JOIN components c ON a.assigned_component_id = c.id
		WHERE a.id = $1
	`
	var alloc LineAllocation
	var compID, compName sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&alloc.ID,
		&alloc.LineID,
		&alloc.LineCode,
		&alloc.OrderID,
		&alloc.OrderCode,
		&alloc.OperatorID,
		&alloc.OperatorCode,
		&alloc.OperatorName,
		&alloc.SkillTier,
		&compID,
		&compName,
		&alloc.Status,
		&alloc.AssignedAt,
		&alloc.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAllocationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find allocation by id: %w", err)
	}
	if compID.Valid {
		alloc.AssignedComponentID = &compID.String
	}
	if compName.Valid {
		alloc.AssignedComponentName = &compName.String
	}
	return &alloc, nil
}

// FindActiveAllocationByOperator checks if an operator is currently active on any line.
func (r *postgresRepository) FindActiveAllocationByOperator(ctx context.Context, operatorID string) (*LineAllocation, error) {
	query := `
		SELECT a.id, a.line_id, l.code AS line_code, a.order_id, o.order_number AS order_code,
		       a.operator_id, op.operator_code AS operator_code, op.name AS operator_name, op.skill_tier,
		       a.assigned_component_id, c.name AS assigned_component_name,
		       a.status, a.assigned_at, a.completed_at
		FROM line_allocations a
		JOIN lines l ON a.line_id = l.id
		JOIN orders o ON a.order_id = o.id
		JOIN operators op ON a.operator_id = op.id
		LEFT JOIN components c ON a.assigned_component_id = c.id
		WHERE a.operator_id = $1 AND a.status = 'ACTIVE'
		LIMIT 1
	`
	var alloc LineAllocation
	var compID, compName sql.NullString
	err := r.db.QueryRowContext(ctx, query, operatorID).Scan(
		&alloc.ID,
		&alloc.LineID,
		&alloc.LineCode,
		&alloc.OrderID,
		&alloc.OrderCode,
		&alloc.OperatorID,
		&alloc.OperatorCode,
		&alloc.OperatorName,
		&alloc.SkillTier,
		&compID,
		&compName,
		&alloc.Status,
		&alloc.AssignedAt,
		&alloc.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active allocation by operator: %w", err)
	}
	if compID.Valid {
		alloc.AssignedComponentID = &compID.String
	}
	if compName.Valid {
		alloc.AssignedComponentName = &compName.String
	}
	return &alloc, nil
}

// CreateAllocation assigns an operator to a line and order.
func (r *postgresRepository) CreateAllocation(ctx context.Context, alloc *LineAllocation) error {
	query := `
		INSERT INTO line_allocations (line_id, order_id, operator_id, assigned_component_id, status, assigned_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, assigned_at
	`
	var compID *string
	if alloc.AssignedComponentID != nil && *alloc.AssignedComponentID != "" {
		compID = alloc.AssignedComponentID
	}
	assignedAt := alloc.AssignedAt
	if assignedAt.IsZero() {
		assignedAt = time.Now()
	}
	return r.db.QueryRowContext(
		ctx,
		query,
		alloc.LineID,
		alloc.OrderID,
		alloc.OperatorID,
		compID,
		alloc.Status,
		assignedAt,
	).Scan(&alloc.ID, &alloc.AssignedAt)
}

// TransferOperator executes BPMN 04 operator line transfer within an atomic database transaction.
func (r *postgresRepository) TransferOperator(
	ctx context.Context,
	oldAllocID string,
	newAlloc *LineAllocation,
	transferLog *AllocationTransferLog,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Mark old allocation as TRANSFERRED
	queryOld := `
		UPDATE line_allocations
		SET status = 'TRANSFERRED', completed_at = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'ACTIVE'
	`
	resOld, err := tx.ExecContext(ctx, queryOld, now, oldAllocID)
	if err != nil {
		return fmt.Errorf("update old allocation status: %w", err)
	}
	rowsAffected, _ := resOld.RowsAffected()
	if rowsAffected == 0 {
		return ErrAllocationNotActive
	}

	// 2. Insert new allocation on destination line
	queryNew := `
		INSERT INTO line_allocations (line_id, order_id, operator_id, assigned_component_id, status, assigned_at)
		VALUES ($1, $2, $3, $4, 'ACTIVE', $5)
		RETURNING id, assigned_at
	`
	var compID *string
	if newAlloc.AssignedComponentID != nil && *newAlloc.AssignedComponentID != "" {
		compID = newAlloc.AssignedComponentID
	}
	assignedAt := newAlloc.AssignedAt
	if assignedAt.IsZero() {
		assignedAt = now
	}
	if err := tx.QueryRowContext(
		ctx,
		queryNew,
		newAlloc.LineID,
		newAlloc.OrderID,
		newAlloc.OperatorID,
		compID,
		assignedAt,
	).Scan(&newAlloc.ID, &newAlloc.AssignedAt); err != nil {
		return fmt.Errorf("insert new allocation: %w", err)
	}

	// 3. Insert audit log in line_transfers
	queryLog := `
		INSERT INTO line_transfers (allocation_id, operator_id, from_line_id, to_line_id, reason, transferred_by_user_id, transferred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, transferred_at
	`
	var transferredByUserID *string
	if transferLog.TransferredByUserID != nil && *transferLog.TransferredByUserID != "" {
		transferredByUserID = transferLog.TransferredByUserID
	}
	transferredAt := transferLog.TransferredAt
	if transferredAt.IsZero() {
		transferredAt = now
	}
	if err := tx.QueryRowContext(
		ctx,
		queryLog,
		newAlloc.ID, // associate log with the new allocation
		newAlloc.OperatorID,
		transferLog.FromLineID,
		transferLog.ToLineID,
		transferLog.Reason,
		transferredByUserID,
		transferredAt,
	).Scan(&transferLog.ID, &transferLog.TransferredAt); err != nil {
		return fmt.Errorf("insert line transfer audit log: %w", err)
	}

	return tx.Commit()
}

// ListTransferHistoriesByAllocation retrieves transfer audit trail records for an allocation.
func (r *postgresRepository) ListTransferHistoriesByAllocation(ctx context.Context, allocationID string) ([]AllocationTransferLog, error) {
	query := `
		SELECT t.id, t.allocation_id, t.operator_id,
		       t.from_line_id, fl.code AS from_line_code,
		       t.to_line_id, tl.code AS to_line_code,
		       t.reason, t.transferred_by_user_id,
		       COALESCE(u.full_name, 'System') AS transferred_by_name,
		       t.transferred_at
		FROM line_transfers t
		JOIN lines fl ON t.from_line_id = fl.id
		JOIN lines tl ON t.to_line_id = tl.id
		LEFT JOIN users u ON t.transferred_by_user_id = u.id
		WHERE t.allocation_id = $1
		ORDER BY t.transferred_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, allocationID)
	if err != nil {
		return nil, fmt.Errorf("query transfer histories: %w", err)
	}
	defer rows.Close()

	var logs []AllocationTransferLog
	for rows.Next() {
		var l AllocationTransferLog
		var transferredByUserID sql.NullString
		if err := rows.Scan(
			&l.ID,
			&l.AllocationID,
			&l.OperatorID,
			&l.FromLineID,
			&l.FromLineCode,
			&l.ToLineID,
			&l.ToLineCode,
			&l.Reason,
			&transferredByUserID,
			&l.TransferredByName,
			&l.TransferredAt,
		); err != nil {
			return nil, fmt.Errorf("scan transfer log: %w", err)
		}
		if transferredByUserID.Valid {
			l.TransferredByUserID = &transferredByUserID.String
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AllocationTransferLog{}
	}
	return logs, rows.Err()
}

// ValidateOperator verifies operator existence and active status.
func (r *postgresRepository) ValidateOperator(ctx context.Context, operatorID string) (string, string, string, error) {
	query := `SELECT name, operator_code, skill_tier FROM operators WHERE id = $1 AND deleted_at IS NULL AND status = 'ACTIVE'`
	var name, code, skillTier string
	err := r.db.QueryRowContext(ctx, query, operatorID).Scan(&name, &code, &skillTier)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", ErrOperatorNotFound
	}
	if err != nil {
		return "", "", "", fmt.Errorf("validate operator: %w", err)
	}
	return name, code, skillTier, nil
}

// ValidateOrder verifies order existence and returns orderCode, clientID, articleID.
func (r *postgresRepository) ValidateOrder(ctx context.Context, orderID string) (string, string, string, error) {
	query := `SELECT order_number, client_id, article_id FROM orders WHERE id = $1 AND deleted_at IS NULL`
	var orderCode, clientID, articleID string
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(&orderCode, &clientID, &articleID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", ErrOrderNotFound
	}
	if err != nil {
		return "", "", "", fmt.Errorf("validate order: %w", err)
	}
	return orderCode, clientID, articleID, nil
}

// ValidateComponent verifies component existence and returns component name.
func (r *postgresRepository) ValidateComponent(ctx context.Context, componentID string) (string, error) {
	query := `SELECT name FROM components WHERE id = $1 AND deleted_at IS NULL`
	var name string
	err := r.db.QueryRowContext(ctx, query, componentID).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("komponen tidak ditemukan")
	}
	if err != nil {
		return "", fmt.Errorf("validate component: %w", err)
	}
	return name, nil
}
