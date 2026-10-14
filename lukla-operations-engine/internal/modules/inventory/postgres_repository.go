package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// VerifyClientAndOrder checks if the client and order exist and match.
func (r *PostgresRepository) VerifyClientAndOrder(ctx context.Context, clientID, orderID string) error {
	var cID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM clients WHERE id = $1 AND deleted_at IS NULL`, clientID).Scan(&cID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrClientNotFound
	} else if err != nil {
		return err
	}

	var oID, oClientID string
	err = r.db.QueryRowContext(ctx, `SELECT id, client_id FROM orders WHERE id = $1`, orderID).Scan(&oID, &oClientID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOrderNotFound
	} else if err != nil {
		return err
	}

	if oClientID != clientID {
		return ErrClientOrderMismatch
	}

	return nil
}

// CreateMaterial inserts a new material into materials table.
func (r *PostgresRepository) CreateMaterial(ctx context.Context, m *Material) error {
	var existingID string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM materials WHERE code = $1`, m.Code).Scan(&existingID)
	if err == nil {
		return ErrMaterialCodeExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	query := `
		INSERT INTO materials (code, name, category, default_unit, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx,
		query,
		m.Code,
		m.Name,
		m.Category,
		m.DefaultUnit,
		m.Description,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *PostgresRepository) FindMaterialByID(ctx context.Context, id string) (*Material, error) {
	query := `SELECT id, code, name, category, default_unit, COALESCE(description, ''), created_at, updated_at FROM materials WHERE id = $1`
	var m Material
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID,
		&m.Code,
		&m.Name,
		&m.Category,
		&m.DefaultUnit,
		&m.Description,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMaterialNotFound
	}
	return &m, err
}

func (r *PostgresRepository) FindMaterialByCode(ctx context.Context, code string) (*Material, error) {
	query := `SELECT id, code, name, category, default_unit, COALESCE(description, ''), created_at, updated_at FROM materials WHERE code = $1`
	var m Material
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&m.ID,
		&m.Code,
		&m.Name,
		&m.Category,
		&m.DefaultUnit,
		&m.Description,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMaterialNotFound
	}
	return &m, err
}

func (r *PostgresRepository) ListMaterials(ctx context.Context, filter ListMaterialsFilter) ([]Material, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", idx))
		args = append(args, filter.Category)
		idx++
	}

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d)", idx, idx))
		args = append(args, pattern)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM materials %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, category, default_unit, COALESCE(description, ''), created_at, updated_at
		FROM materials
		%s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	materials := []Material{}
	for rows.Next() {
		var m Material
		if err := rows.Scan(
			&m.ID,
			&m.Code,
			&m.Name,
			&m.Category,
			&m.DefaultUnit,
			&m.Description,
			&m.CreatedAt,
			&m.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		materials = append(materials, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return materials, total, nil
}

// CreateReceipt records inbound receipt and creates corresponding inventory lots atomically.
func (r *PostgresRepository) CreateReceipt(ctx context.Context, receipt *InventoryReceipt, items []CreateReceiptItem) (*ReceiptDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Verify Client and Order (FR-6.1 Mandatory Tagging)
	var oClientID string
	err = tx.QueryRowContext(ctx, `SELECT client_id FROM orders WHERE id = $1`, receipt.OrderID).Scan(&oClientID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	} else if err != nil {
		return nil, err
	}
	if oClientID != receipt.ClientID {
		return nil, ErrClientOrderMismatch
	}

	// 2. Insert receipt
	insertReceiptQuery := `
		INSERT INTO inventory_receipts (
			receipt_number, client_id, order_id, received_date, supplier_name, delivery_note_number, received_by_user_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRowContext(
		ctx,
		insertReceiptQuery,
		receipt.ReceiptNumber,
		receipt.ClientID,
		receipt.OrderID,
		receipt.ReceivedDate,
		receipt.SupplierName,
		receipt.DeliveryNoteNumber,
		receipt.ReceivedByUserID,
		receipt.Notes,
	).Scan(&receipt.ID, &receipt.CreatedAt, &receipt.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// 3. Insert each lot and receipt item
	for _, it := range items {
		// Verify material exists
		var matID string
		err = tx.QueryRowContext(ctx, `SELECT id FROM materials WHERE id = $1`, it.MaterialID).Scan(&matID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMaterialNotFound
		} else if err != nil {
			return nil, err
		}

		rollCount := it.RollCount
		if rollCount <= 0 {
			rollCount = 1
		}

		var notes *string
		if strings.TrimSpace(it.Notes) != "" {
			n := strings.TrimSpace(it.Notes)
			notes = &n
		}

		// Insert lot (FR-6.1 Tag Client ID & Order ID)
		insertLotQuery := `
			INSERT INTO inventory_lots (
				receipt_id, lot_number, material_id, client_id, order_id, initial_quantity, current_quantity, unit, roll_count, status, warehouse_location, notes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'AVAILABLE', $10, $11)
			RETURNING id
		`
		var lotID string
		err = tx.QueryRowContext(
			ctx,
			insertLotQuery,
			receipt.ID,
			it.LotNumber,
			it.MaterialID,
			receipt.ClientID,
			receipt.OrderID,
			it.Quantity,
			it.Quantity,
			it.Unit,
			rollCount,
			it.WarehouseLocation,
			notes,
		).Scan(&lotID)
		if err != nil {
			return nil, err
		}

		// Insert receipt item
		insertItemQuery := `
			INSERT INTO inventory_receipt_items (
				receipt_id, lot_id, material_id, quantity, unit, roll_count, warehouse_location, notes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.ExecContext(
			ctx,
			insertItemQuery,
			receipt.ID,
			lotID,
			it.MaterialID,
			it.Quantity,
			it.Unit,
			rollCount,
			it.WarehouseLocation,
			notes,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.FindReceiptByID(ctx, receipt.ID)
}

func (r *PostgresRepository) FindReceiptByID(ctx context.Context, id string) (*ReceiptDetail, error) {
	query := `
		SELECT 
			r.id, r.receipt_number, r.client_id, c.name AS client_name,
			r.order_id, o.order_number AS order_code, r.received_date,
			COALESCE(r.supplier_name, ''), COALESCE(r.delivery_note_number, ''),
			u.full_name AS received_by_name, COALESCE(r.notes, ''), r.created_at
		FROM inventory_receipts r
		JOIN clients c ON r.client_id = c.id
		JOIN orders o ON r.order_id = o.id
		JOIN users u ON r.received_by_user_id = u.id
		WHERE r.id = $1
	`
	var detail ReceiptDetail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&detail.ReceiptNumber,
		&detail.ClientID,
		&detail.ClientName,
		&detail.OrderID,
		&detail.OrderCode,
		&detail.ReceivedDate,
		&detail.SupplierName,
		&detail.DeliveryNoteNumber,
		&detail.ReceivedByName,
		&detail.Notes,
		&detail.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrReceiptNotFound
	} else if err != nil {
		return nil, err
	}

	// Fetch items
	itemsQuery := `
		SELECT 
			ri.id, ri.material_id, m.code, m.name, ri.lot_id, l.lot_number,
			ri.quantity, ri.unit, ri.roll_count, ri.warehouse_location, COALESCE(ri.notes, '')
		FROM inventory_receipt_items ri
		JOIN materials m ON ri.material_id = m.id
		JOIN inventory_lots l ON ri.lot_id = l.id
		WHERE ri.receipt_id = $1
		ORDER BY m.name ASC
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	detail.Items = []ReceiptItemPayload{}
	for rows.Next() {
		var it ReceiptItemPayload
		if err := rows.Scan(
			&it.ID,
			&it.MaterialID,
			&it.MaterialCode,
			&it.MaterialName,
			&it.LotID,
			&it.LotNumber,
			&it.Quantity,
			&it.Unit,
			&it.RollCount,
			&it.WarehouseLocation,
			&it.Notes,
		); err != nil {
			return nil, err
		}
		detail.Items = append(detail.Items, it)
	}

	detail.TotalItemsCount = len(detail.Items)
	return &detail, nil
}

func (r *PostgresRepository) ListReceipts(ctx context.Context, filter ListReceiptsFilter) ([]ReceiptSummary, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.ClientID != "" {
		conditions = append(conditions, fmt.Sprintf("r.client_id = $%d", idx))
		args = append(args, filter.ClientID)
		idx++
	}

	if filter.OrderID != "" {
		conditions = append(conditions, fmt.Sprintf("r.order_id = $%d", idx))
		args = append(args, filter.OrderID)
		idx++
	}

	if filter.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("r.received_date >= $%d", idx))
		args = append(args, filter.StartDate)
		idx++
	}

	if filter.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("r.received_date <= $%d", idx))
		args = append(args, filter.EndDate)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM inventory_receipts r %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			r.id, r.receipt_number, r.client_id, c.name AS client_name,
			r.order_id, o.order_number AS order_code, r.received_date,
			COALESCE(r.supplier_name, ''),
			(SELECT COUNT(*) FROM inventory_receipt_items ri WHERE ri.receipt_id = r.id) AS total_items,
			r.created_at
		FROM inventory_receipts r
		JOIN clients c ON r.client_id = c.id
		JOIN orders o ON r.order_id = o.id
		%s
		ORDER BY r.received_date DESC, r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	receipts := []ReceiptSummary{}
	for rows.Next() {
		var s ReceiptSummary
		if err := rows.Scan(
			&s.ID,
			&s.ReceiptNumber,
			&s.ClientID,
			&s.ClientName,
			&s.OrderID,
			&s.OrderCode,
			&s.ReceivedDate,
			&s.SupplierName,
			&s.TotalItemsCount,
			&s.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		receipts = append(receipts, s)
	}

	return receipts, total, nil
}

func (r *PostgresRepository) FindLotByID(ctx context.Context, id string) (*InventoryLot, error) {
	query := `
		SELECT 
			l.id, l.receipt_id, l.lot_number, l.material_id, m.code, m.name,
			l.client_id, c.name AS client_name, l.order_id, o.order_number AS order_code,
			l.initial_quantity, l.current_quantity, l.unit, l.roll_count, l.status,
			l.warehouse_location, l.notes, l.created_at, l.updated_at
		FROM inventory_lots l
		JOIN materials m ON l.material_id = m.id
		JOIN clients c ON l.client_id = c.id
		JOIN orders o ON l.order_id = o.id
		WHERE l.id = $1
	`
	var lot InventoryLot
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&lot.ID,
		&lot.ReceiptID,
		&lot.LotNumber,
		&lot.MaterialID,
		&lot.MaterialCode,
		&lot.MaterialName,
		&lot.ClientID,
		&lot.ClientName,
		&lot.OrderID,
		&lot.OrderCode,
		&lot.InitialQuantity,
		&lot.CurrentQuantity,
		&lot.Unit,
		&lot.RollCount,
		&lot.Status,
		&lot.WarehouseLocation,
		&lot.Notes,
		&lot.CreatedAt,
		&lot.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLotNotFound
	}
	return &lot, err
}

func (r *PostgresRepository) ListStockBalances(ctx context.Context, filter ListStocksFilter) ([]StockBalance, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.ClientID != "" {
		conditions = append(conditions, fmt.Sprintf("l.client_id = $%d", idx))
		args = append(args, filter.ClientID)
		idx++
	}

	if filter.MaterialID != "" {
		conditions = append(conditions, fmt.Sprintf("l.material_id = $%d", idx))
		args = append(args, filter.MaterialID)
		idx++
	}

	if filter.OrderID != "" {
		conditions = append(conditions, fmt.Sprintf("l.order_id = $%d", idx))
		args = append(args, filter.OrderID)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			l.material_id,
			m.code AS material_code,
			m.name AS material_name,
			l.client_id,
			c.name AS client_name,
			l.order_id,
			o.order_number AS order_code,
			SUM(l.current_quantity) AS total_quantity,
			l.unit,
			COUNT(CASE WHEN l.status = 'AVAILABLE' AND l.current_quantity > 0 THEN 1 END) AS available_lots_count
		FROM inventory_lots l
		JOIN materials m ON l.material_id = m.id
		JOIN clients c ON l.client_id = c.id
		JOIN orders o ON l.order_id = o.id
		%s
		GROUP BY l.material_id, m.code, m.name, l.client_id, c.name, l.order_id, o.order_number, l.unit
		HAVING SUM(l.current_quantity) > 0
		ORDER BY m.name ASC, c.name ASC
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := []StockBalance{}
	for rows.Next() {
		var b StockBalance
		if err := rows.Scan(
			&b.MaterialID,
			&b.MaterialCode,
			&b.MaterialName,
			&b.ClientID,
			&b.ClientName,
			&b.OrderID,
			&b.OrderCode,
			&b.TotalQuantity,
			&b.Unit,
			&b.AvailableLotsCount,
		); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}

	return balances, nil
}

// ==========================================
// TAHAP 3.2: OUTBOUND, ISSUES & BLOCKED ATTEMPTS
// ==========================================

func (r *PostgresRepository) CreateIssue(ctx context.Context, userID string, req CreateIssueRequest) (*IssueDetail, error) {
	if len(req.Items) == 0 {
		return nil, ErrIssueItemsEmpty
	}

	// 1. Fetch Target Order & Client
	var targetOrderID, targetClientID, targetOrderNumber, targetClientName string
	err := r.db.QueryRowContext(ctx, `
		SELECT o.id, o.client_id, o.order_number, c.name 
		FROM orders o 
		JOIN clients c ON o.client_id = c.id 
		WHERE o.id = $1
	`, req.TargetOrderID).Scan(&targetOrderID, &targetClientID, &targetOrderNumber, &targetClientName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	} else if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 2. Validate each Lot with row-locking
	type lotValidation struct {
		lotID             string
		lotNumber         string
		materialID        string
		materialCode      string
		materialName      string
		clientID          string
		clientName        string
		currentQuantity   float64
		unit              string
		status            string
		requestedQuantity float64
		notes             string
	}

	var validatedLots []lotValidation

	for _, it := range req.Items {
		var lv lotValidation
		lv.lotID = it.LotID
		lv.requestedQuantity = it.Quantity
		lv.notes = it.Notes

		lotQuery := `
			SELECT 
				l.lot_number, l.material_id, m.code, m.name,
				l.client_id, c.name, l.current_quantity, l.unit, l.status
			FROM inventory_lots l
			JOIN materials m ON l.material_id = m.id
			JOIN clients c ON l.client_id = c.id
			WHERE l.id = $1
			FOR UPDATE
		`
		err = tx.QueryRowContext(ctx, lotQuery, it.LotID).Scan(
			&lv.lotNumber,
			&lv.materialID,
			&lv.materialCode,
			&lv.materialName,
			&lv.clientID,
			&lv.clientName,
			&lv.currentQuantity,
			&lv.unit,
			&lv.status,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLotNotFound
		} else if err != nil {
			return nil, err
		}

		// SKENARIO UJI 1: BLOKADE KONTAMINASI SILANG (FR-6.2)
		if lv.clientID != targetClientID {
			// Rollback the transaction immediately to release FOR UPDATE row lock on inventory_lots
			_ = tx.Rollback()

			// Record the blocked attempt to audit log outside the aborted tx
			reason := fmt.Sprintf("BLOKIR KEAMANAN: Bahan pada lot '%s' milik klien '%s' tidak dapat dikeluarkan untuk order '%s' milik klien '%s'.", lv.lotNumber, lv.clientName, targetOrderNumber, targetClientName)
			_ = r.RecordBlockedAttempt(context.Background(), &BlockedAttemptRecord{
				AttemptedAt:       time.Now(),
				AttemptedByUserID: userID,
				LotID:             it.LotID,
				LotNumber:         lv.lotNumber,
				LotClientID:       lv.clientID,
				TargetOrderID:     targetOrderID,
				TargetOrderCode:   targetOrderNumber,
				TargetClientID:    targetClientID,
				RequestedQuantity: it.Quantity,
				Reason:            reason,
			})
			return nil, fmt.Errorf("%w: %s", ErrCrossContaminationBlocked, reason)
		}

		if lv.status != "AVAILABLE" {
			return nil, ErrLotNotAvailable
		}

		if lv.currentQuantity < it.Quantity {
			return nil, fmt.Errorf("%w: sisa kuantitas pada lot '%s' hanya tersisa %.2f %s (diminta %.2f %s)", ErrInsufficientStock, lv.lotNumber, lv.currentQuantity, lv.unit, it.Quantity, lv.unit)
		}

		validatedLots = append(validatedLots, lv)
	}

	// 3. Generate Issue Number if not provided
	issueNumber := strings.TrimSpace(req.IssueNumber)
	if issueNumber == "" {
		todayPrefix := time.Now().Format("20060102")
		var count int
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM inventory_issues WHERE issue_number LIKE $1`, "ISS-"+todayPrefix+"-%").Scan(&count)
		issueNumber = fmt.Sprintf("ISS-%s-%04d", todayPrefix, count+1)
	}

	// 4. Insert into inventory_issues
	var issueID string
	var createdAt, updatedAt time.Time
	insertIssueQuery := `
		INSERT INTO inventory_issues (
			issue_number, target_order_id, cutting_table_number, issue_date, issued_by_user_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRowContext(
		ctx,
		insertIssueQuery,
		issueNumber,
		targetOrderID,
		req.CuttingTableNumber,
		req.IssuedDate,
		userID,
		req.Notes,
	).Scan(&issueID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	// 5. Insert issue items and deduct lot quantity
	var responseItems []IssueItemPayload
	for _, lv := range validatedLots {
		var itemID string
		insertItemQuery := `
			INSERT INTO inventory_issue_items (
				issue_id, lot_id, material_id, quantity, unit, notes
			) VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`
		err = tx.QueryRowContext(ctx, insertItemQuery, issueID, lv.lotID, lv.materialID, lv.requestedQuantity, lv.unit, lv.notes).Scan(&itemID)
		if err != nil {
			return nil, err
		}

		newQuantity := lv.currentQuantity - lv.requestedQuantity
		newStatus := "AVAILABLE"
		if newQuantity <= 0 {
			newQuantity = 0
			newStatus = "DEPLETED"
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE inventory_lots 
			SET current_quantity = $1, status = $2, updated_at = NOW() 
			WHERE id = $3
		`, newQuantity, newStatus, lv.lotID)
		if err != nil {
			return nil, err
		}

		responseItems = append(responseItems, IssueItemPayload{
			ID:                   itemID,
			LotID:                lv.lotID,
			LotNumber:            lv.lotNumber,
			MaterialCode:         lv.materialCode,
			MaterialName:         lv.materialName,
			ClientName:           lv.clientName,
			Quantity:             lv.requestedQuantity,
			Unit:                 lv.unit,
			RemainingLotQuantity: newQuantity,
			Notes:                lv.notes,
		})
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	var userName string
	_ = r.db.QueryRowContext(ctx, `SELECT full_name FROM users WHERE id = $1`, userID).Scan(&userName)

	return &IssueDetail{
		IssueSummary: IssueSummary{
			ID:                 issueID,
			IssueNumber:        issueNumber,
			TargetOrderID:      targetOrderID,
			TargetOrderCode:    targetOrderNumber,
			TargetClientName:   targetClientName,
			CuttingTableNumber: req.CuttingTableNumber,
			IssuedDate:         req.IssuedDate,
			TotalItemsCount:    len(responseItems),
			CreatedAt:          createdAt,
		},
		IssuedByName: userName,
		Notes:        req.Notes,
		Items:        responseItems,
	}, nil
}

func (r *PostgresRepository) FindIssueByID(ctx context.Context, id string) (*IssueDetail, error) {
	query := `
		SELECT 
			i.id, i.issue_number, i.target_order_id, o.order_number, c.name,
			COALESCE(i.cutting_table_number, ''), i.issue_date,
			COALESCE(u.full_name, ''), COALESCE(i.notes, ''), i.created_at
		FROM inventory_issues i
		JOIN orders o ON i.target_order_id = o.id
		JOIN clients c ON o.client_id = c.id
		JOIN users u ON i.issued_by_user_id = u.id
		WHERE i.id = $1
	`
	var d IssueDetail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.IssueNumber,
		&d.TargetOrderID,
		&d.TargetOrderCode,
		&d.TargetClientName,
		&d.CuttingTableNumber,
		&d.IssuedDate,
		&d.IssuedByName,
		&d.Notes,
		&d.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrIssueNotFound
	} else if err != nil {
		return nil, err
	}

	itemsQuery := `
		SELECT 
			ii.id, ii.lot_id, l.lot_number, m.code, m.name, c.name,
			ii.quantity, ii.unit, l.current_quantity, COALESCE(ii.notes, '')
		FROM inventory_issue_items ii
		JOIN inventory_lots l ON ii.lot_id = l.id
		JOIN materials m ON ii.material_id = m.id
		JOIN clients c ON l.client_id = c.id
		WHERE ii.issue_id = $1
		ORDER BY m.name ASC
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	d.Items = []IssueItemPayload{}
	for rows.Next() {
		var it IssueItemPayload
		if err := rows.Scan(
			&it.ID,
			&it.LotID,
			&it.LotNumber,
			&it.MaterialCode,
			&it.MaterialName,
			&it.ClientName,
			&it.Quantity,
			&it.Unit,
			&it.RemainingLotQuantity,
			&it.Notes,
		); err != nil {
			return nil, err
		}
		d.Items = append(d.Items, it)
	}

	d.TotalItemsCount = len(d.Items)
	return &d, nil
}

func (r *PostgresRepository) ListIssues(ctx context.Context, filter ListIssuesFilter) ([]IssueSummary, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.OrderID != "" {
		conditions = append(conditions, fmt.Sprintf("i.target_order_id = $%d", idx))
		args = append(args, filter.OrderID)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM inventory_issues i %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			i.id, i.issue_number, i.target_order_id, o.order_number, c.name,
			COALESCE(i.cutting_table_number, ''), i.issue_date,
			(SELECT COUNT(*) FROM inventory_issue_items ii WHERE ii.issue_id = i.id) AS total_items,
			i.created_at
		FROM inventory_issues i
		JOIN orders o ON i.target_order_id = o.id
		JOIN clients c ON o.client_id = c.id
		%s
		ORDER BY i.issue_date DESC, i.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	issues := []IssueSummary{}
	for rows.Next() {
		var is IssueSummary
		if err := rows.Scan(
			&is.ID,
			&is.IssueNumber,
			&is.TargetOrderID,
			&is.TargetOrderCode,
			&is.TargetClientName,
			&is.CuttingTableNumber,
			&is.IssuedDate,
			&is.TotalItemsCount,
			&is.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		issues = append(issues, is)
	}

	return issues, total, nil
}

func (r *PostgresRepository) RecordBlockedAttempt(ctx context.Context, b *BlockedAttemptRecord) error {
	query := `
		INSERT INTO inventory_blocked_attempts (
			attempted_at, attempted_by_user_id, lot_id, lot_number, lot_client_id,
			target_order_id, target_order_number, target_client_id, requested_quantity, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	return r.db.QueryRowContext(
		ctx,
		query,
		b.AttemptedAt,
		b.AttemptedByUserID,
		b.LotID,
		b.LotNumber,
		b.LotClientID,
		b.TargetOrderID,
		b.TargetOrderCode,
		b.TargetClientID,
		b.RequestedQuantity,
		b.Reason,
	).Scan(&b.ID)
}

func (r *PostgresRepository) ListBlockedAttempts(ctx context.Context, filter ListBlockedAttemptsFilter) ([]BlockedAttemptRecord, int, error) {
	countQuery := `SELECT COUNT(*) FROM inventory_blocked_attempts`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT 
			ba.id, ba.attempted_at, ba.attempted_by_user_id, u.full_name,
			ba.lot_id, ba.lot_number, ba.lot_client_id, lc.name AS lot_client_name,
			ba.target_order_id, ba.target_order_number, ba.target_client_id, tc.name AS target_client_name,
			ba.requested_quantity, ba.reason
		FROM inventory_blocked_attempts ba
		JOIN users u ON ba.attempted_by_user_id = u.id
		JOIN clients lc ON ba.lot_client_id = lc.id
		JOIN clients tc ON ba.target_client_id = tc.id
		ORDER BY ba.attempted_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	attempts := []BlockedAttemptRecord{}
	for rows.Next() {
		var a BlockedAttemptRecord
		if err := rows.Scan(
			&a.ID,
			&a.AttemptedAt,
			&a.AttemptedByUserID,
			&a.AttemptedByName,
			&a.LotID,
			&a.LotNumber,
			&a.LotClientID,
			&a.LotClientName,
			&a.TargetOrderID,
			&a.TargetOrderCode,
			&a.TargetClientID,
			&a.TargetClientName,
			&a.RequestedQuantity,
			&a.Reason,
		); err != nil {
			return nil, 0, err
		}
		attempts = append(attempts, a)
	}

	return attempts, total, nil
}

// ==========================================
// TAHAP 3.2: CUTTING SCRAPS & WASTE
// ==========================================

func (r *PostgresRepository) CreateScrap(ctx context.Context, userID string, req CreateScrapRequest) (*CuttingScrapRecord, error) {
	// Validate order exists
	var orderCode string
	err := r.db.QueryRowContext(ctx, `SELECT order_number FROM orders WHERE id = $1`, req.OrderID).Scan(&orderCode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	} else if err != nil {
		return nil, err
	}

	// Validate material exists
	var materialName string
	err = r.db.QueryRowContext(ctx, `SELECT name FROM materials WHERE id = $1`, req.MaterialID).Scan(&materialName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMaterialNotFound
	} else if err != nil {
		return nil, err
	}

	// Validate category and disposition
	switch req.ScrapCategory {
	case string(ScrapCategoryPercaKecil),
		string(ScrapCategoryCacatKainRoll),
		string(ScrapCategoryUjungGulungan),
		string(ScrapCategoryKesalahanPotong):
		// valid
	default:
		return nil, ErrScrapCategoryInvalid
	}

	switch req.Disposition {
	case string(ScrapDispositionSimpanGudangSisa),
		string(ScrapDispositionKembalikanKlien),
		string(ScrapDispositionJualLimbah),
		string(ScrapDispositionBuang):
		// valid
	default:
		return nil, ErrScrapDispositionInvalid
	}

	var scrapID string
	var createdAt time.Time
	query := `
		INSERT INTO cutting_scraps (
			order_id, material_id, scrap_category, quantity, unit, disposition, reported_by_user_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	err = r.db.QueryRowContext(
		ctx,
		query,
		req.OrderID,
		req.MaterialID,
		req.ScrapCategory,
		req.Quantity,
		req.Unit,
		req.Disposition,
		userID,
		req.Notes,
	).Scan(&scrapID, &createdAt)
	if err != nil {
		return nil, err
	}

	var userName string
	_ = r.db.QueryRowContext(ctx, `SELECT full_name FROM users WHERE id = $1`, userID).Scan(&userName)

	return &CuttingScrapRecord{
		ID:             scrapID,
		OrderID:        req.OrderID,
		OrderCode:      orderCode,
		MaterialID:     req.MaterialID,
		MaterialName:   materialName,
		ScrapCategory:  req.ScrapCategory,
		Quantity:       req.Quantity,
		Unit:           req.Unit,
		Disposition:    req.Disposition,
		Notes:          req.Notes,
		RecordedByName: userName,
		CreatedAt:      createdAt,
	}, nil
}

func (r *PostgresRepository) ListScraps(ctx context.Context, filter ListScrapsFilter) ([]CuttingScrapRecord, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.OrderID != "" {
		conditions = append(conditions, fmt.Sprintf("cs.order_id = $%d", idx))
		args = append(args, filter.OrderID)
		idx++
	}

	if filter.ScrapCategory != "" {
		conditions = append(conditions, fmt.Sprintf("cs.scrap_category = $%d", idx))
		args = append(args, filter.ScrapCategory)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM cutting_scraps cs %s`, whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT 
			cs.id, cs.order_id, o.order_number, cs.material_id, m.name,
			cs.scrap_category, cs.quantity, cs.unit, cs.disposition,
			COALESCE(cs.notes, ''), u.full_name, cs.created_at
		FROM cutting_scraps cs
		JOIN orders o ON cs.order_id = o.id
		JOIN materials m ON cs.material_id = m.id
		JOIN users u ON cs.reported_by_user_id = u.id
		%s
		ORDER BY cs.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	scraps := []CuttingScrapRecord{}
	for rows.Next() {
		var s CuttingScrapRecord
		if err := rows.Scan(
			&s.ID,
			&s.OrderID,
			&s.OrderCode,
			&s.MaterialID,
			&s.MaterialName,
			&s.ScrapCategory,
			&s.Quantity,
			&s.Unit,
			&s.Disposition,
			&s.Notes,
			&s.RecordedByName,
			&s.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		scraps = append(scraps, s)
	}

	return scraps, total, nil
}
