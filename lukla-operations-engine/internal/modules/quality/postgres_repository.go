package quality

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PostgresRepository implements quality.Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository instance.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ListDefectCategories retrieves all defect categories with optional isActive filter.
func (r *PostgresRepository) ListDefectCategories(ctx context.Context, isActive *bool) ([]DefectCategory, error) {
	query := `
		SELECT id, code, name, severity, description, is_active, created_at, updated_at
		FROM qc_defect_categories
		WHERE ($1::BOOLEAN IS NULL OR is_active = $1)
		ORDER BY code ASC
	`
	rows, err := r.db.QueryContext(ctx, query, isActive)
	if err != nil {
		return nil, fmt.Errorf("list defect categories: %w", err)
	}
	defer rows.Close()

	var categories []DefectCategory
	for rows.Next() {
		var cat DefectCategory
		if err := rows.Scan(
			&cat.ID,
			&cat.Code,
			&cat.Name,
			&cat.Severity,
			&cat.Description,
			&cat.IsActive,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan defect category: %w", err)
		}
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate defect categories: %w", err)
	}
	return categories, nil
}

// CreateDefectCategory inserts a new defect category into the master catalog.
func (r *PostgresRepository) CreateDefectCategory(ctx context.Context, cat *DefectCategory) error {
	query := `
		INSERT INTO qc_defect_categories (code, name, severity, description, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(
		ctx,
		query,
		cat.Code,
		cat.Name,
		cat.Severity,
		cat.Description,
		cat.IsActive,
	).Scan(&cat.ID, &cat.CreatedAt, &cat.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "23505") {
			return ErrDefectCategoryCodeExists
		}
		return fmt.Errorf("create defect category: %w", err)
	}
	return nil
}

// FindDefectCategoryByID finds a defect category by ID.
func (r *PostgresRepository) FindDefectCategoryByID(ctx context.Context, id string) (*DefectCategory, error) {
	query := `
		SELECT id, code, name, severity, description, is_active, created_at, updated_at
		FROM qc_defect_categories
		WHERE id = $1
	`
	var cat DefectCategory
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cat.ID,
		&cat.Code,
		&cat.Name,
		&cat.Severity,
		&cat.Description,
		&cat.IsActive,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDefectCategoryNotFound
		}
		return nil, fmt.Errorf("find defect category by id: %w", err)
	}
	return &cat, nil
}

// ListQCQueue retrieves assemblies waiting for inspection.
func (r *PostgresRepository) ListQCQueue(ctx context.Context, filter ListQCQueueFilter) ([]QCQueueItem, int, error) {
	whereClauses := []string{"a.status IN ('MENUNGGU_QC', 'WAITING_QC', 'REPAIR')"}
	args := []interface{}{}
	argIdx := 1

	if filter.OrderID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.order_id = $%d", argIdx))
		args = append(args, filter.OrderID)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM assemblies a
		WHERE %s
	`, whereSQL)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count qc queue: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			a.id AS assembly_id,
			a.order_id,
			o.order_number,
			c.name AS client_name,
			art.title AS article_name,
			l.id AS line_id,
			l.code AS line_code,
			a.quantity_assembled,
			a.status,
			a.created_at
		FROM assemblies a
		JOIN orders o ON a.order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN articles art ON a.article_id = art.id
		LEFT JOIN LATERAL (
			SELECT la.line_id 
			FROM line_allocations la 
			WHERE la.order_id = a.order_id 
			ORDER BY la.assigned_at DESC 
			LIMIT 1
		) last_alloc ON TRUE
		LEFT JOIN lines l ON last_alloc.line_id = l.id
		WHERE %s
		ORDER BY a.created_at ASC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query qc queue: %w", err)
	}
	defer rows.Close()

	var items []QCQueueItem
	for rows.Next() {
		var item QCQueueItem
		if err := rows.Scan(
			&item.AssemblyID,
			&item.OrderID,
			&item.OrderCode,
			&item.ClientName,
			&item.ArticleName,
			&item.LineID,
			&item.LineCode,
			&item.CompletedQuantity,
			&item.Status,
			&item.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan qc queue item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate qc queue rows: %w", err)
	}

	return items, total, nil
}

// CreateInspection records an inspection transaction and updates assembly/order status.
func (r *PostgresRepository) CreateInspection(ctx context.Context, req *CreateQCInspectionRequest, inspectorID string) (*QCInspectionDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify assembly exists and lock row
	var assemblyQty int
	var assemblyStatus string
	var assemblyOrderID string
	err = tx.QueryRowContext(ctx, `
		SELECT quantity_assembled, status, order_id
		FROM assemblies
		WHERE id = $1
		FOR UPDATE
	`, req.AssemblyID).Scan(&assemblyQty, &assemblyStatus, &assemblyOrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssemblyNotFound
		}
		return nil, fmt.Errorf("lock assembly: %w", err)
	}

	if req.OrderID != assemblyOrderID {
		return nil, fmt.Errorf("%w: orderId tidak cocok dengan assembly", ErrInvalidQuantity)
	}

	totalChecked := req.PassedQuantity + req.RepairQuantity + req.RejectQuantity
	if totalChecked <= 0 {
		return nil, fmt.Errorf("%w: total checked harus lebih dari 0", ErrInvalidQuantity)
	}
	if totalChecked > assemblyQty {
		return nil, fmt.Errorf("%w: total diperiksa (%d) melebihi kuantitas assembly (%d)", ErrInvalidQuantity, totalChecked, assemblyQty)
	}

	// 2. Determine inspection date
	inspectionDate := req.InspectionDate
	if inspectionDate == "" {
		inspectionDate = time.Now().Format("2006-01-02")
	}

	// 3. Generate inspection number: QC-YYYYMMDD-XXXX
	cleanDate := strings.ReplaceAll(inspectionDate, "-", "")
	var countToday int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM qc_inspections
		WHERE inspection_date = $1::DATE
	`, inspectionDate).Scan(&countToday)
	if err != nil {
		return nil, fmt.Errorf("count inspections today: %w", err)
	}

	inspectionNumber := fmt.Sprintf("QC-%s-%04d", cleanDate, countToday+1)

	// 4. Insert qc_inspections
	var inspectionID string
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO qc_inspections (
			inspection_number, assembly_id, order_id, inspector_id, inspection_date,
			total_checked, passed_quantity, repair_quantity, reject_quantity, notes
		) VALUES ($1, $2, $3, $4, $5::DATE, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`,
		inspectionNumber,
		req.AssemblyID,
		req.OrderID,
		inspectorID,
		inspectionDate,
		totalChecked,
		req.PassedQuantity,
		req.RepairQuantity,
		req.RejectQuantity,
		req.Notes,
	).Scan(&inspectionID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("insert qc_inspections: %w", err)
	}

	// 5. Insert defects if any
	for _, defect := range req.Defects {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO qc_inspection_defects (
				inspection_id, defect_category_id, component_id, quantity, disposition, notes
			) VALUES ($1, $2, $3, $4, $5, $6)
		`,
			inspectionID,
			defect.DefectCategoryID,
			defect.ComponentID,
			defect.Quantity,
			defect.Disposition,
			defect.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("insert qc_inspection_defects: %w", err)
		}
	}

	// 6. Update assembly status
	var newAssemblyStatus string
	if req.PassedQuantity == totalChecked {
		newAssemblyStatus = "PASSED"
	} else if req.RejectQuantity == totalChecked {
		newAssemblyStatus = "REJECTED"
	} else if req.RepairQuantity > 0 && req.PassedQuantity == 0 {
		newAssemblyStatus = "REPAIR"
	} else {
		newAssemblyStatus = "PARTIALLY_PASSED"
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE assemblies
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, newAssemblyStatus, req.AssemblyID)
	if err != nil {
		return nil, fmt.Errorf("update assembly status: %w", err)
	}

	// 7. Check if order completion target is achieved
	var targetQty int
	err = tx.QueryRowContext(ctx, `
		SELECT target_quantity
		FROM orders
		WHERE id = $1
	`, req.OrderID).Scan(&targetQty)
	if err == nil && targetQty > 0 {
		var cumPassed int
		_ = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(passed_quantity), 0)
			FROM qc_inspections
			WHERE order_id = $1
		`, req.OrderID).Scan(&cumPassed)

		if cumPassed >= targetQty {
			_, _ = tx.ExecContext(ctx, `
				UPDATE orders
				SET status = 'COMPLETED', updated_at = CURRENT_TIMESTAMP
				WHERE id = $1 AND status != 'COMPLETED'
			`, req.OrderID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit inspection tx: %w", err)
	}

	return r.FindInspectionByID(ctx, inspectionID)
}

// ListInspections retrieves inspection summaries based on criteria.
func (r *PostgresRepository) ListInspections(ctx context.Context, filter ListQCInspectionsFilter) ([]QCInspectionSummary, int, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.OrderID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("qi.order_id = $%d", argIdx))
		args = append(args, filter.OrderID)
		argIdx++
	}

	if filter.Status != "" {
		switch strings.ToUpper(filter.Status) {
		case "LOLOS", "PASSED":
			whereClauses = append(whereClauses, "qi.passed_quantity > 0")
		case "REPAIR":
			whereClauses = append(whereClauses, "qi.repair_quantity > 0")
		case "REJECT", "REJECTED":
			whereClauses = append(whereClauses, "qi.reject_quantity > 0")
		}
	}

	if filter.StartDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("qi.inspection_date >= $%d::DATE", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}

	if filter.EndDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("qi.inspection_date <= $%d::DATE", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM qc_inspections qi
		WHERE %s
	`, whereSQL)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inspections: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			qi.id,
			qi.inspection_number,
			o.order_number,
			c.name AS client_name,
			u.full_name AS inspector_name,
			qi.inspection_date::TEXT,
			qi.passed_quantity,
			qi.repair_quantity,
			qi.reject_quantity,
			qi.total_checked,
			qi.created_at
		FROM qc_inspections qi
		JOIN orders o ON qi.order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN users u ON qi.inspector_id = u.id
		WHERE %s
		ORDER BY qi.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query inspections: %w", err)
	}
	defer rows.Close()

	var summaries []QCInspectionSummary
	for rows.Next() {
		var s QCInspectionSummary
		if err := rows.Scan(
			&s.ID,
			&s.InspectionNumber,
			&s.OrderCode,
			&s.ClientName,
			&s.InspectorName,
			&s.InspectionDate,
			&s.PassedQuantity,
			&s.RepairQuantity,
			&s.RejectQuantity,
			&s.TotalChecked,
			&s.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan inspection summary: %w", err)
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate inspection summaries: %w", err)
	}

	return summaries, total, nil
}

// FindInspectionByID retrieves full inspection detail with defect line items.
func (r *PostgresRepository) FindInspectionByID(ctx context.Context, id string) (*QCInspectionDetail, error) {
	query := `
		SELECT 
			qi.id,
			qi.inspection_number,
			qi.assembly_id,
			qi.order_id,
			o.order_number,
			c.name AS client_name,
			art.title AS article_name,
			qi.inspector_id,
			u.full_name AS inspector_name,
			qi.inspection_date::TEXT,
			qi.passed_quantity,
			qi.repair_quantity,
			qi.reject_quantity,
			qi.total_checked,
			qi.notes,
			qi.created_at
		FROM qc_inspections qi
		JOIN orders o ON qi.order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN assemblies a ON qi.assembly_id = a.id
		JOIN articles art ON a.article_id = art.id
		JOIN users u ON qi.inspector_id = u.id
		WHERE qi.id = $1
	`
	var detail QCInspectionDetail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&detail.InspectionNumber,
		&detail.AssemblyID,
		&detail.OrderID,
		&detail.OrderCode,
		&detail.ClientName,
		&detail.ArticleName,
		&detail.InspectorID,
		&detail.InspectorName,
		&detail.InspectionDate,
		&detail.PassedQuantity,
		&detail.RepairQuantity,
		&detail.RejectQuantity,
		&detail.TotalChecked,
		&detail.Notes,
		&detail.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInspectionNotFound
		}
		return nil, fmt.Errorf("find inspection by id: %w", err)
	}

	// Fetch defects
	defectRows, err := r.db.QueryContext(ctx, `
		SELECT 
			qid.id,
			qdc.code,
			qdc.name,
			qdc.severity,
			qid.component_id,
			comp.name AS component_name,
			qid.quantity,
			qid.disposition,
			qid.notes
		FROM qc_inspection_defects qid
		JOIN qc_defect_categories qdc ON qid.defect_category_id = qdc.id
		LEFT JOIN components comp ON qid.component_id = comp.id
		WHERE qid.inspection_id = $1
		ORDER BY qid.created_at ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query inspection defects: %w", err)
	}
	defer defectRows.Close()

	detail.Defects = []QCInspectionDefect{}
	for defectRows.Next() {
		var d QCInspectionDefect
		if err := defectRows.Scan(
			&d.ID,
			&d.DefectCategoryCode,
			&d.DefectCategoryName,
			&d.Severity,
			&d.ComponentID,
			&d.ComponentName,
			&d.Quantity,
			&d.Disposition,
			&d.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan inspection defect: %w", err)
		}
		detail.Defects = append(detail.Defects, d)
	}

	if err := defectRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate defect rows: %w", err)
	}

	return &detail, nil
}

// GetOrderQualitySummary computes quality analytics for an order.
func (r *PostgresRepository) GetOrderQualitySummary(ctx context.Context, orderID string) (*QCOrderSummary, error) {
	var summary QCOrderSummary
	summary.OrderID = orderID

	// 1. Order details
	err := r.db.QueryRowContext(ctx, `
		SELECT order_number, target_quantity
		FROM orders
		WHERE id = $1
	`, orderID).Scan(&summary.OrderCode, &summary.TargetQuantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order for summary: %w", err)
	}

	// 2. Inspection aggregates
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(total_checked), 0),
			COALESCE(SUM(passed_quantity), 0),
			COALESCE(SUM(repair_quantity), 0),
			COALESCE(SUM(reject_quantity), 0)
		FROM qc_inspections
		WHERE order_id = $1
	`, orderID).Scan(
		&summary.TotalChecked,
		&summary.TotalPassed,
		&summary.TotalRepair,
		&summary.TotalReject,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate qc inspections: %w", err)
	}

	if summary.TotalChecked > 0 {
		summary.PassRatePercentage = (float64(summary.TotalPassed) / float64(summary.TotalChecked)) * 100.0
	}
	summary.IsOrderCompleted = summary.TotalPassed >= summary.TargetQuantity

	// 3. Top defects
	rows, err := r.db.QueryContext(ctx, `
		SELECT qdc.name, COALESCE(SUM(qid.quantity), 0) AS total_count
		FROM qc_inspection_defects qid
		JOIN qc_inspections qi ON qid.inspection_id = qi.id
		JOIN qc_defect_categories qdc ON qid.defect_category_id = qdc.id
		WHERE qi.order_id = $1
		GROUP BY qdc.name
		ORDER BY total_count DESC
		LIMIT 5
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("query top defects: %w", err)
	}
	defer rows.Close()

	summary.TopDefects = []TopDefectItem{}
	for rows.Next() {
		var item TopDefectItem
		if err := rows.Scan(&item.CategoryName, &item.Count); err != nil {
			return nil, fmt.Errorf("scan top defect: %w", err)
		}
		summary.TopDefects = append(summary.TopDefects, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate top defect rows: %w", err)
	}

	return &summary, nil
}

