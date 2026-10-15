package portal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// PostgresRepository implements Repository for PostgreSQL database.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetDashboard retrieves aggregated metrics for client portal dashboard.
func (r *PostgresRepository) GetDashboard(ctx context.Context, clientID string) (*ClientPortalDashboard, error) {
	var dash ClientPortalDashboard

	// 1. Fetch client profile info
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, code FROM clients WHERE id = $1
	`, clientID).Scan(&dash.Client.ID, &dash.Client.Name, &dash.Client.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientTenantRequired
		}
		return nil, fmt.Errorf("get client info: %w", err)
	}

	// 2. Active orders count (DRAFT / IN_PRODUCTION)
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders 
		WHERE client_id = $1 AND status IN ('DRAFT', 'IN_PRODUCTION')
	`, clientID).Scan(&dash.ActiveOrdersCount)

	// 3. Completed orders count
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders 
		WHERE client_id = $1 AND status = 'COMPLETED'
	`, clientID).Scan(&dash.CompletedOrdersCount)

	// 4. Ready finished goods count (passed QC units across orders)
	_ = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(qi.passed_quantity), 0) 
		FROM qc_inspections qi 
		JOIN orders o ON qi.order_id = o.id 
		WHERE o.client_id = $1
	`, clientID).Scan(&dash.ReadyFinishedGoodsCount)

	// 5. Active consigned raw materials count (unique materials with available balance)
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT material_id) 
		FROM inventory_lots 
		WHERE client_id = $1 AND current_quantity > 0
	`, clientID).Scan(&dash.ActiveMaterialsCount)

	// 6. Unread notifications (default 0 for Phase 5.1)
	dash.UnreadNotificationsCount = 0

	return &dash, nil
}

// ListOrders retrieves orders belonging strictly to the authenticated client.
func (r *PostgresRepository) ListOrders(ctx context.Context, clientID string, filter ListClientOrdersFilter) ([]ClientPortalOrderSummary, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	// Count total orders matching filter
	countQuery := `
		SELECT COUNT(*)
		FROM orders o
		JOIN articles a ON o.article_id = a.id
		WHERE o.client_id = $1
		  AND ($2 = '' OR $2 = 'ALL' OR o.status = $2)
		  AND ($3 = '' OR o.order_number ILIKE '%' || $3 || '%' OR a.title ILIKE '%' || $3 || '%')
	`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, clientID, filter.Status, filter.Search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count client orders: %w", err)
	}

	query := `
		SELECT 
			o.id,
			o.order_number,
			a.title AS article_name,
			a.code AS article_code,
			COALESCE(a.category, 'GARMENT') AS category,
			o.target_quantity,
			COALESCE(qc.total_passed, 0) AS passed_quantity,
			o.status,
			o.target_date::TEXT,
			o.created_at
		FROM orders o
		JOIN articles a ON o.article_id = a.id
		LEFT JOIN (
			SELECT order_id, COALESCE(SUM(passed_quantity), 0) AS total_passed
			FROM qc_inspections
			GROUP BY order_id
		) qc ON o.id = qc.order_id
		WHERE o.client_id = $1
		  AND ($2 = '' OR $2 = 'ALL' OR o.status = $2)
		  AND ($3 = '' OR o.order_number ILIKE '%' || $3 || '%' OR a.title ILIKE '%' || $3 || '%')
		ORDER BY o.created_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.QueryContext(ctx, query, clientID, filter.Status, filter.Search, filter.Limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list client orders: %w", err)
	}
	defer rows.Close()

	var orders []ClientPortalOrderSummary
	for rows.Next() {
		var o ClientPortalOrderSummary
		var targetDate sql.NullString
		if err := rows.Scan(
			&o.ID,
			&o.OrderNumber,
			&o.ArticleName,
			&o.ArticleCode,
			&o.Category,
			&o.TargetQuantity,
			&o.PassedQuantity,
			&o.Status,
			&targetDate,
			&o.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan client order: %w", err)
		}
		if targetDate.Valid {
			o.TargetDate = &targetDate.String
		}

		if o.TargetQuantity > 0 {
			pct := (float64(o.PassedQuantity) / float64(o.TargetQuantity)) * 100.0
			o.ProgressPercentage = math.Min(100.0, math.Round(pct*10)/10)
		}

		orders = append(orders, o)
	}

	return orders, total, nil
}

// GetOrderDetail retrieves single order detail with 3-stage progress breakdown.
func (r *PostgresRepository) GetOrderDetail(ctx context.Context, clientID, orderID string) (*ClientPortalOrderDetail, error) {
	query := `
		SELECT 
			o.id,
			o.order_number,
			o.target_quantity,
			o.status,
			o.start_date::TEXT,
			o.target_date::TEXT,
			o.notes,
			a.id AS article_id,
			a.code AS article_code,
			a.title AS article_name,
			COALESCE(a.category, 'GARMENT') AS article_category
		FROM orders o
		JOIN articles a ON o.article_id = a.id
		WHERE o.id = $1 AND o.client_id = $2
	`

	var detail ClientPortalOrderDetail
	var startDate, targetDate, notes sql.NullString

	err := r.db.QueryRowContext(ctx, query, orderID, clientID).Scan(
		&detail.ID,
		&detail.OrderNumber,
		&detail.TargetQuantity,
		&detail.Status,
		&startDate,
		&targetDate,
		&notes,
		&detail.Article.ID,
		&detail.Article.Code,
		&detail.Article.Name,
		&detail.Article.Category,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order detail: %w", err)
	}

	if startDate.Valid {
		detail.StartDate = &startDate.String
	}
	if targetDate.Valid {
		detail.TargetDate = &targetDate.String
	}
	if notes.Valid {
		detail.Notes = &notes.String
	}

	// 1. Sewing/Assembly quantity
	_ = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity_assembled), 0)
		FROM assemblies
		WHERE order_id = $1
	`, orderID).Scan(&detail.Progress.SewingQuantity)

	// 2. QC inspection quantities
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(passed_quantity), 0),
			COALESCE(SUM(repair_quantity), 0),
			COALESCE(SUM(reject_quantity), 0)
		FROM qc_inspections
		WHERE order_id = $1
	`, orderID).Scan(
		&detail.Progress.PassedQCQuantity,
		&detail.Progress.RepairQCQuantity,
		&detail.Progress.RejectQCQuantity,
	)

	// 3. Cutting quantity: panels cut (at least as much as assembled)
	detail.Progress.CuttingQuantity = int(math.Max(float64(detail.Progress.SewingQuantity), float64(detail.Progress.PassedQCQuantity)))
	if detail.Status == "COMPLETED" {
		detail.Progress.CuttingQuantity = detail.TargetQuantity
	}

	// Overall completion percentage
	if detail.TargetQuantity > 0 {
		pct := (float64(detail.Progress.PassedQCQuantity) / float64(detail.TargetQuantity)) * 100.0
		detail.Progress.OverallPercentage = math.Min(100.0, math.Round(pct*10)/10)
	}

	return &detail, nil
}

// GetOrderQuality retrieves quality assurance metrics for a client order.
func (r *PostgresRepository) GetOrderQuality(ctx context.Context, clientID, orderID string) (*ClientPortalOrderQuality, error) {
	var q ClientPortalOrderQuality
	q.OrderID = orderID

	// Check order ownership
	err := r.db.QueryRowContext(ctx, `
		SELECT order_number, target_quantity 
		FROM orders 
		WHERE id = $1 AND client_id = $2
	`, orderID, clientID).Scan(&q.OrderNumber, &q.TargetQuantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("check order quality ownership: %w", err)
	}

	// Aggregate QC inspections
	err = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(total_checked), 0),
			COALESCE(SUM(passed_quantity), 0),
			COALESCE(SUM(repair_quantity), 0),
			COALESCE(SUM(reject_quantity), 0),
			COUNT(*)
		FROM qc_inspections
		WHERE order_id = $1
	`, orderID).Scan(
		&q.TotalChecked,
		&q.PassedQuantity,
		&q.RepairQuantity,
		&q.RejectQuantity,
		&q.InspectionsCount,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregate order quality: %w", err)
	}

	if q.TotalChecked > 0 {
		rate := (float64(q.PassedQuantity) / float64(q.TotalChecked)) * 100.0
		q.PassRate = math.Round(rate*10) / 10
	}

	return &q, nil
}

// ListMaterialStocks retrieves consigned raw material inventory for the client.
func (r *PostgresRepository) ListMaterialStocks(ctx context.Context, clientID, category string) ([]ClientPortalMaterialStock, error) {
	query := `
		SELECT 
			m.id AS material_id,
			m.code AS material_code,
			m.name AS material_name,
			m.category,
			m.default_unit,
			l.id AS lot_id,
			l.lot_number,
			l.current_quantity,
			COALESCE(l.unit, m.default_unit) AS unit,
			COALESCE(l.warehouse_location, 'GUDANG-A-01') AS location,
			l.created_at::DATE::TEXT AS received_date
		FROM inventory_lots l
		JOIN materials m ON l.material_id = m.id
		WHERE l.client_id = $1 AND l.current_quantity > 0
		  AND ($2 = '' OR m.category = $2)
		ORDER BY m.name ASC, l.lot_number ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clientID, category)
	if err != nil {
		return nil, fmt.Errorf("list material stocks: %w", err)
	}
	defer rows.Close()

	materialMap := make(map[string]*ClientPortalMaterialStock)
	var orderList []string

	for rows.Next() {
		var matID, matCode, matName, matCat, defaultUnit string
		var lot ClientPortalLotStock
		var receivedDate sql.NullString

		if err := rows.Scan(
			&matID,
			&matCode,
			&matName,
			&matCat,
			&defaultUnit,
			&lot.LotID,
			&lot.LotNumber,
			&lot.AvailableQuantity,
			&lot.Unit,
			&lot.Location,
			&receivedDate,
		); err != nil {
			return nil, fmt.Errorf("scan material stock: %w", err)
		}
		if receivedDate.Valid {
			lot.ReceivedDate = &receivedDate.String
		}

		item, exists := materialMap[matID]
		if !exists {
			item = &ClientPortalMaterialStock{
				MaterialID:   matID,
				MaterialCode: matCode,
				MaterialName: matName,
				Category:     matCat,
				DefaultUnit:  defaultUnit,
				Lots:         []ClientPortalLotStock{},
			}
			materialMap[matID] = item
			orderList = append(orderList, matID)
		}
		item.TotalAvailableQuantity += lot.AvailableQuantity
		item.Lots = append(item.Lots, lot)
	}

	result := make([]ClientPortalMaterialStock, 0, len(orderList))
	for _, id := range orderList {
		result = append(result, *materialMap[id])
	}

	return result, nil
}

// ListFinishedGoodsStocks retrieves garments ready for delivery.
func (r *PostgresRepository) ListFinishedGoodsStocks(ctx context.Context, clientID string) ([]ClientPortalFinishedGoodsStock, error) {
	query := `
		SELECT 
			o.id AS order_id,
			o.order_number,
			a.code AS article_code,
			a.title AS article_name,
			COALESCE(SUM(qi.passed_quantity), 0) AS ready_quantity
		FROM orders o
		JOIN articles a ON o.article_id = a.id
		JOIN qc_inspections qi ON o.id = qi.order_id
		WHERE o.client_id = $1
		GROUP BY o.id, o.order_number, a.code, a.title
		HAVING SUM(qi.passed_quantity) > 0
		ORDER BY o.order_number ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clientID)
	if err != nil {
		return nil, fmt.Errorf("list finished goods stocks: %w", err)
	}
	defer rows.Close()

	var list []ClientPortalFinishedGoodsStock
	for rows.Next() {
		var fg ClientPortalFinishedGoodsStock
		if err := rows.Scan(
			&fg.OrderID,
			&fg.OrderNumber,
			&fg.ArticleCode,
			&fg.ArticleName,
			&fg.ReadyQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan finished goods stock: %w", err)
		}
		fg.Status = "READY_FOR_DISPATCH"
		list = append(list, fg)
	}

	return list, nil
}
