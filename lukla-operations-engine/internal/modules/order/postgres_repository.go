package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

// PostgresRepository mengimplementasikan antarmuka Repository menggunakan PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository membuat instance baru PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// List mengambil daftar seluruh pesanan internal pabrik.
func (r *PostgresRepository) List(ctx context.Context, clientID, status string) ([]InternalOrderSummary, error) {
	query := `
		SELECT 
			o.id,
			o.order_number,
			o.client_id,
			c.name AS client_name,
			o.article_id,
			a.code AS article_code,
			a.title AS article_title,
			o.target_quantity,
			COALESCE(qc.total_passed, 0) AS passed_quantity,
			o.status,
			o.start_date::TEXT,
			o.target_date::TEXT,
			o.completed_at,
			o.created_at
		FROM orders o
		JOIN clients c ON o.client_id = c.id
		JOIN articles a ON o.article_id = a.id
		LEFT JOIN (
			SELECT order_id, COALESCE(SUM(passed_quantity), 0) AS total_passed
			FROM qc_inspections
			GROUP BY order_id
		) qc ON o.id = qc.order_id
		WHERE ($1 = '' OR o.client_id = $1::uuid)
		  AND ($2 = '' OR $2 = 'ALL' OR o.status = $2)
		ORDER BY o.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, clientID, status)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []InternalOrderSummary
	for rows.Next() {
		var o InternalOrderSummary
		var startDate, targetDate sql.NullString
		var completedAt sql.NullTime

		if err := rows.Scan(
			&o.ID,
			&o.OrderNumber,
			&o.ClientID,
			&o.ClientName,
			&o.ArticleID,
			&o.ArticleCode,
			&o.ArticleTitle,
			&o.TargetQuantity,
			&o.PassedQuantity,
			&o.Status,
			&startDate,
			&targetDate,
			&completedAt,
			&o.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		if startDate.Valid {
			o.StartDate = &startDate.String
		}
		if targetDate.Valid {
			o.TargetDate = &targetDate.String
		}
		if completedAt.Valid {
			o.CompletedAt = &completedAt.Time
		}

		orders = append(orders, o)
	}

	return orders, nil
}

// GetByID mengambil detail komprehensif pesanan internal pabrik.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*InternalOrderDetail, error) {
	query := `
		SELECT 
			o.id,
			o.order_number,
			o.client_id,
			c.name AS client_name,
			o.article_id,
			a.code AS article_code,
			a.title AS article_title,
			o.target_quantity,
			o.status,
			o.start_date::TEXT,
			o.target_date::TEXT,
			o.completed_at,
			o.created_at,
			o.notes
		FROM orders o
		JOIN clients c ON o.client_id = c.id
		JOIN articles a ON o.article_id = a.id
		WHERE o.id = $1
	`

	var detail InternalOrderDetail
	var startDate, targetDate, notes sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&detail.ID,
		&detail.OrderNumber,
		&detail.ClientID,
		&detail.ClientName,
		&detail.ArticleID,
		&detail.ArticleCode,
		&detail.ArticleTitle,
		&detail.TargetQuantity,
		&detail.Status,
		&startDate,
		&targetDate,
		&completedAt,
		&detail.CreatedAt,
		&notes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by id: %w", err)
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
	if completedAt.Valid {
		detail.CompletedAt = &completedAt.Time
	}

	// 1. Sewing quantity
	_ = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quantity_assembled), 0)
		FROM assemblies
		WHERE order_id = $1
	`, id).Scan(&detail.SewingQuantity)

	// 2. QC quantities
	var totalChecked int
	_ = r.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(SUM(total_checked), 0),
			COALESCE(SUM(passed_quantity), 0),
			COALESCE(SUM(repair_quantity), 0),
			COALESCE(SUM(reject_quantity), 0)
		FROM qc_inspections
		WHERE order_id = $1
	`, id).Scan(
		&totalChecked,
		&detail.PassedQuantity,
		&detail.RepairQuantity,
		&detail.RejectQuantity,
	)

	// 3. Cutting quantity
	detail.CuttingQuantity = int(math.Max(float64(detail.SewingQuantity), float64(detail.PassedQuantity)))
	if detail.Status == "COMPLETED" {
		detail.CuttingQuantity = detail.TargetQuantity
	}

	if totalChecked > 0 {
		detail.PassRate = math.Round((float64(detail.PassedQuantity)/float64(totalChecked))*1000) / 10
	}

	return &detail, nil
}

// Complete menutup order dan menetapkan completed_at.
func (r *PostgresRepository) Complete(ctx context.Context, id string, notes *string) (*InternalOrderDetail, error) {
	// Cek status saat ini
	var currentStatus string
	err := r.db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id = $1", id).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("check order status: %w", err)
	}

	if currentStatus == "COMPLETED" {
		return nil, ErrOrderAlreadyCompleted
	}

	updateQuery := `
		UPDATE orders 
		SET status = 'COMPLETED',
		    completed_at = NOW(),
		    notes = COALESCE($2, notes),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err = r.db.ExecContext(ctx, updateQuery, id, notes)
	if err != nil {
		return nil, fmt.Errorf("update order completed: %w", err)
	}

	return r.GetByID(ctx, id)
}

