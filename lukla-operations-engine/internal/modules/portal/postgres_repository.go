package portal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/lukla/operations-engine/internal/modules/sample"
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

	// 6. Unread notifications count from notifications table
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications 
		WHERE client_id = $1 AND is_read = FALSE
	`, clientID).Scan(&dash.UnreadNotificationsCount)

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

// ListClientSamples retrieves sample requests for the authenticated client.
func (r *PostgresRepository) ListClientSamples(ctx context.Context, clientID, status string) ([]ClientPortalSampleSummary, error) {
	query := `
		SELECT 
			sr.id,
			sr.sample_code,
			a.title AS article_title,
			COALESCE(v.version_number, 1) AS current_version,
			v.proposed_unit_price,
			COALESCE(v.currency, 'IDR') AS currency,
			sr.status,
			sr.target_date::TEXT,
			sr.created_at
		FROM sample_requests sr
		JOIN articles a ON sr.article_id = a.id
		LEFT JOIN LATERAL (
			SELECT version_number, proposed_unit_price, currency
			FROM sample_versions
			WHERE sample_request_id = sr.id
			ORDER BY version_number DESC
			LIMIT 1
		) v ON true
		WHERE sr.client_id = $1::uuid
		  AND ($2 = '' OR $2 = 'ALL' OR sr.status = $2)
		ORDER BY sr.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, clientID, status)
	if err != nil {
		return nil, fmt.Errorf("list client samples: %w", err)
	}
	defer rows.Close()

	var list []ClientPortalSampleSummary
	for rows.Next() {
		var s ClientPortalSampleSummary
		var proposedPrice sql.NullFloat64
		var targetDate sql.NullString

		if err := rows.Scan(
			&s.ID,
			&s.SampleCode,
			&s.ArticleTitle,
			&s.CurrentVersion,
			&proposedPrice,
			&s.Currency,
			&s.Status,
			&targetDate,
			&s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan client sample: %w", err)
		}

		if proposedPrice.Valid {
			price := proposedPrice.Float64
			s.ProposedUnitPrice = &price
		}
		if targetDate.Valid {
			s.TargetDate = &targetDate.String
		}

		list = append(list, s)
	}

	return list, nil
}

// GetClientSampleDetail retrieves comprehensive drill-down of a sample for client portal.
func (r *PostgresRepository) GetClientSampleDetail(ctx context.Context, clientID, sampleID string) (*ClientPortalSampleDetail, error) {
	querySample := `
		SELECT 
			sr.id,
			sr.sample_code,
			a.title AS article_title,
			a.code AS article_code,
			sr.status
		FROM sample_requests sr
		JOIN articles a ON sr.article_id = a.id
		WHERE sr.id = $1::uuid AND sr.client_id = $2::uuid
	`
	var detail ClientPortalSampleDetail
	err := r.db.QueryRowContext(ctx, querySample, sampleID, clientID).Scan(
		&detail.ID,
		&detail.SampleCode,
		&detail.ArticleTitle,
		&detail.ArticleCode,
		&detail.Status,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotFound
		}
		return nil, fmt.Errorf("query sample detail: %w", err)
	}

	// Ambil semua versi sample
	queryVersions := `
		SELECT 
			id, sample_request_id, version_number, version_label, status,
			revision_notes, client_feedback, total_sam, proposed_unit_price,
			currency, price_notes, approved_at, created_at, updated_at
		FROM sample_versions
		WHERE sample_request_id = $1
		ORDER BY version_number ASC
	`
	vRows, err := r.db.QueryContext(ctx, queryVersions, sampleID)
	if err != nil {
		return nil, fmt.Errorf("query versions: %w", err)
	}
	defer vRows.Close()

	var versions []sample.SampleVersionDetail
	var latestVerNum int
	var latestPrice *float64
	var latestCurrency = "IDR"

	for vRows.Next() {
		var v sample.SampleVersionDetail
		var revNotes, clientFeedback, priceNotes sql.NullString
		var proposedPrice sql.NullFloat64
		var approvedAt sql.NullTime

		if err := vRows.Scan(
			&v.ID,
			&v.SampleRequestID,
			&v.VersionNumber,
			&v.VersionLabel,
			&v.Status,
			&revNotes,
			&clientFeedback,
			&v.TotalSAM,
			&proposedPrice,
			&v.Currency,
			&priceNotes,
			&approvedAt,
			&v.CreatedAt,
			&v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}

		if revNotes.Valid {
			v.RevisionNotes = &revNotes.String
		}
		if clientFeedback.Valid {
			v.ClientFeedback = &clientFeedback.String
		}
		if proposedPrice.Valid {
			price := proposedPrice.Float64
			v.ProposedUnitPrice = &price
		}
		if priceNotes.Valid {
			v.PriceNotes = &priceNotes.String
		}
		if approvedAt.Valid {
			v.ApprovedAt = &approvedAt.Time
		}

		if v.VersionNumber >= latestVerNum {
			latestVerNum = v.VersionNumber
			latestPrice = v.ProposedUnitPrice
			latestCurrency = v.Currency
		}

		versions = append(versions, v)
	}
	vRows.Close()

	// Ambil data time study untuk setiap versi
	compQuery := `
		SELECT 
			sts.id, sts.sample_version_id, sts.component_id,
			c.name AS component_name, c.sequence_number,
			sts.observed_seconds, sts.allowance_percentage, sts.calculated_sam, sts.target_per_hour,
			u.full_name AS operator_name, sts.notes, sts.created_at
		FROM sample_time_studies sts
		JOIN components c ON sts.component_id = c.id
		LEFT JOIN users u ON sts.operator_id = u.id
		WHERE sts.sample_version_id = $1
		ORDER BY c.sequence_number ASC
	`
	for i := range versions {
		cRows, err := r.db.QueryContext(ctx, compQuery, versions[i].ID)
		if err != nil {
			return nil, fmt.Errorf("query components time study: %w", err)
		}

		versions[i].Components = make([]sample.SampleTimeStudyItem, 0)
		for cRows.Next() {
			var item sample.SampleTimeStudyItem
			var opName, notes sql.NullString
			if err := cRows.Scan(
				&item.ID,
				&item.SampleVersionID,
				&item.ComponentID,
				&item.ComponentName,
				&item.SequenceNumber,
				&item.ObservedSeconds,
				&item.AllowancePercentage,
				&item.CalculatedSAM,
				&item.TargetPerHour,
				&opName,
				&notes,
				&item.CreatedAt,
			); err != nil {
				cRows.Close()
				return nil, fmt.Errorf("scan time study component: %w", err)
			}
			if opName.Valid {
				item.OperatorName = &opName.String
			}
			if notes.Valid {
				item.Notes = &notes.String
			}
			versions[i].Components = append(versions[i].Components, item)
		}
		cRows.Close()
	}

	detail.CurrentVersionNumber = latestVerNum
	detail.LatestProposedUnitPrice = latestPrice
	detail.Currency = latestCurrency
	detail.Versions = versions

	return &detail, nil
}

// ApproveSample sets the active version of a sample to APPROVED, syncs components standard SAM, and updates sample request status.
func (r *PostgresRepository) ApproveSample(ctx context.Context, clientID, sampleID string, feedbackNotes *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Verifikasi kepemilikan sample oleh client
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM sample_requests WHERE id = $1 AND client_id = $2 FOR UPDATE", sampleID, clientID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSampleNotFound
		}
		return fmt.Errorf("verify sample client: %w", err)
	}

	// 2. Ambil versi aktif (tertinggi)
	var versionID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM sample_versions 
		WHERE sample_request_id = $1 
		ORDER BY version_number DESC 
		LIMIT 1 FOR UPDATE
	`, sampleID).Scan(&versionID)
	if err != nil {
		return fmt.Errorf("find active version: %w", err)
	}

	// 3. Update status versi menjadi APPROVED
	_, err = tx.ExecContext(ctx, `
		UPDATE sample_versions
		SET status = 'APPROVED',
		    approved_at = NOW(),
		    client_feedback = COALESCE($1, client_feedback),
		    updated_at = NOW()
		WHERE id = $2
	`, feedbackNotes, versionID)
	if err != nil {
		return fmt.Errorf("update version to approved: %w", err)
	}

	// 4. Update status sample_requests menjadi APPROVED
	_, err = tx.ExecContext(ctx, `
		UPDATE sample_requests
		SET status = 'APPROVED',
		    updated_at = NOW()
		WHERE id = $1
	`, sampleID)
	if err != nil {
		return fmt.Errorf("update sample request to approved: %w", err)
	}

	// 5. Sinkronisasikan Standard SAM ke masterdata components untuk artikel ini
	_, err = tx.ExecContext(ctx, `
		UPDATE components c
		SET standard_sam = sts.calculated_sam,
		    updated_at = NOW()
		FROM sample_time_studies sts
		WHERE sts.sample_version_id = $1 AND sts.component_id = c.id
	`, versionID)
	if err != nil {
		return fmt.Errorf("sync component standard SAM: %w", err)
	}

	return tx.Commit()
}

// RequestSampleRevision sets the active version of a sample to REVISED and records client feedback.
func (r *PostgresRepository) RequestSampleRevision(ctx context.Context, clientID, sampleID string, revisionNotes string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Verifikasi kepemilikan sample oleh client
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM sample_requests WHERE id = $1 AND client_id = $2 FOR UPDATE", sampleID, clientID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSampleNotFound
		}
		return fmt.Errorf("verify sample client: %w", err)
	}

	// 2. Ambil versi aktif (tertinggi)
	var versionID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM sample_versions 
		WHERE sample_request_id = $1 
		ORDER BY version_number DESC 
		LIMIT 1 FOR UPDATE
	`, sampleID).Scan(&versionID)
	if err != nil {
		return fmt.Errorf("find active version: %w", err)
	}

	// 3. Update status versi menjadi REVISED dan catat masukan revisi klien
	_, err = tx.ExecContext(ctx, `
		UPDATE sample_versions
		SET status = 'REVISED',
		    client_feedback = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, revisionNotes, versionID)
	if err != nil {
		return fmt.Errorf("update version to revised: %w", err)
	}

	// 4. Update status sample_requests menjadi REVISED
	_, err = tx.ExecContext(ctx, `
		UPDATE sample_requests
		SET status = 'REVISED',
		    updated_at = NOW()
		WHERE id = $1
	`, sampleID)
	if err != nil {
		return fmt.Errorf("update sample request to revised: %w", err)
	}

	return tx.Commit()
}
