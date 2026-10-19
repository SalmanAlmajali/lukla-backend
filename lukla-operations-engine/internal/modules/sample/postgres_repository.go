package sample

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// PostgresRepository mengimplementasikan interface Repository menggunakan PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository membuat instance baru PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateSampleRequest membuat permintaan sample baru dan secara otomatis menginisialisasi versi 1 (v1).
func (r *PostgresRepository) CreateSampleRequest(ctx context.Context, req *SampleRequestDetail, initialVersion *SampleVersionDetail) (*SampleRequestDetail, error) {
	// 1. Validasi keberadaan Client
	var clientName string
	err := r.db.QueryRowContext(ctx, "SELECT name FROM clients WHERE id = $1", req.ClientID).Scan(&clientName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("check client: %w", err)
	}

	// 2. Validasi keberadaan Article
	var articleCode, articleTitle string
	err = r.db.QueryRowContext(ctx, "SELECT code, title FROM articles WHERE id = $1", req.ArticleID).Scan(&articleCode, &articleTitle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, fmt.Errorf("check article: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 3. Generate Sequential Sample Code: SMP-YYYYMM-XXX
	now := time.Now()
	prefix := fmt.Sprintf("SMP-%s-", now.Format("200601"))
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM sample_requests WHERE sample_code LIKE $1", prefix+"%").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("generate sample code: %w", err)
	}
	sampleCode := fmt.Sprintf("%s%03d", prefix, count+1)

	// 4. Insert sample_requests
	var sampleID string
	var createdAt, updatedAt time.Time
	insertSampleQuery := `
		INSERT INTO sample_requests (
			sample_code, client_id, article_id, status, target_date, initial_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, insertSampleQuery,
		sampleCode, req.ClientID, req.ArticleID, StatusDraft, req.TargetDate, req.InitialNotes,
	).Scan(&sampleID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert sample_request: %w", err)
	}

	// 5. Insert initial version (v1)
	insertVersionQuery := `
		INSERT INTO sample_versions (
			sample_request_id, version_number, version_label, status, revision_notes, total_sam, currency, created_at, updated_at
		) VALUES (
			$1, 1, 'v1', $2, 'Versi awal inisiasi spesifikasi sample prototipe', 0.00, 'IDR', NOW(), NOW()
		) RETURNING id
	`
	var versionID string
	err = tx.QueryRowContext(ctx, insertVersionQuery, sampleID, StatusDraft).Scan(&versionID)
	if err != nil {
		return nil, fmt.Errorf("insert initial version v1: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetSampleRequestByID(ctx, sampleID)
}

// ListSampleRequests mengambil daftar sample request dengan filter dan pagination.
func (r *PostgresRepository) ListSampleRequests(ctx context.Context, clientID, status, search string, page, limit int) ([]SampleRequestSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIdx := 1

	if strings.TrimSpace(clientID) != "" {
		conditions = append(conditions, fmt.Sprintf("sr.client_id = $%d::uuid", argIdx))
		args = append(args, strings.TrimSpace(clientID))
		argIdx++
	}

	if strings.TrimSpace(status) != "" && strings.ToUpper(strings.TrimSpace(status)) != "ALL" {
		conditions = append(conditions, fmt.Sprintf("sr.status = $%d", argIdx))
		args = append(args, strings.ToUpper(strings.TrimSpace(status)))
		argIdx++
	}

	if strings.TrimSpace(search) != "" {
		searchPattern := "%" + strings.TrimSpace(search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(sr.sample_code ILIKE $%d OR a.code ILIKE $%d OR a.title ILIKE $%d OR c.name ILIKE $%d)",
			argIdx, argIdx, argIdx, argIdx,
		))
		args = append(args, searchPattern)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Hitung total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM sample_requests sr
		JOIN clients c ON sr.client_id = c.id
		JOIN articles a ON sr.article_id = a.id
		%s
	`, whereClause)

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count sample requests: %w", err)
	}

	// Ambil data
	query := fmt.Sprintf(`
		SELECT 
			sr.id,
			sr.sample_code,
			sr.client_id,
			c.name AS client_name,
			sr.article_id,
			a.code AS article_code,
			a.title AS article_title,
			COALESCE((SELECT MAX(version_number) FROM sample_versions sv WHERE sv.sample_request_id = sr.id), 1) AS current_version,
			sr.status,
			sr.target_date::TEXT,
			sr.created_at
		FROM sample_requests sr
		JOIN clients c ON sr.client_id = c.id
		JOIN articles a ON sr.article_id = a.id
		%s
		ORDER BY sr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sample requests: %w", err)
	}
	defer rows.Close()

	list := make([]SampleRequestSummary, 0)
	for rows.Next() {
		var s SampleRequestSummary
		var targetDate sql.NullString
		err := rows.Scan(
			&s.ID,
			&s.SampleCode,
			&s.ClientID,
			&s.ClientName,
			&s.ArticleID,
			&s.ArticleCode,
			&s.ArticleTitle,
			&s.CurrentVersion,
			&s.Status,
			&targetDate,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan sample request summary: %w", err)
		}
		if targetDate.Valid {
			s.TargetDate = &targetDate.String
		}
		list = append(list, s)
	}

	return list, total, nil
}

// GetSampleRequestByID mengambil detail lengkap sample beserta riwayat seluruh versi dan time study-nya.
func (r *PostgresRepository) GetSampleRequestByID(ctx context.Context, id string) (*SampleRequestDetail, error) {
	query := `
		SELECT 
			sr.id,
			sr.sample_code,
			sr.client_id,
			c.name AS client_name,
			sr.article_id,
			a.code AS article_code,
			a.title AS article_title,
			sr.status,
			sr.target_date::TEXT,
			sr.initial_notes,
			sr.converted_order_id,
			o.order_number AS converted_order_number,
			sr.created_at,
			sr.updated_at
		FROM sample_requests sr
		JOIN clients c ON sr.client_id = c.id
		JOIN articles a ON sr.article_id = a.id
		LEFT JOIN orders o ON sr.converted_order_id = o.id
		WHERE sr.id = $1
	`

	var d SampleRequestDetail
	var targetDate, initialNotes, convertedOrderID, convertedOrderNumber sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.SampleCode,
		&d.ClientID,
		&d.ClientName,
		&d.ArticleID,
		&d.ArticleCode,
		&d.ArticleTitle,
		&d.Status,
		&targetDate,
		&initialNotes,
		&convertedOrderID,
		&convertedOrderNumber,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotFound
		}
		return nil, fmt.Errorf("get sample request by id: %w", err)
	}

	if targetDate.Valid {
		d.TargetDate = &targetDate.String
	}
	if initialNotes.Valid {
		d.InitialNotes = &initialNotes.String
	}
	if convertedOrderID.Valid {
		d.ConvertedOrderID = &convertedOrderID.String
	}
	if convertedOrderNumber.Valid {
		d.ConvertedOrderNumber = &convertedOrderNumber.String
	}

	// Ambil semua versi
	versionsQuery := `
		SELECT 
			sv.id,
			sv.sample_request_id,
			sv.version_number,
			sv.version_label,
			sv.status,
			sv.revision_notes,
			sv.client_feedback,
			sv.total_sam,
			sv.proposed_unit_price,
			sv.currency,
			sv.price_notes,
			sv.approved_at,
			sv.created_at,
			sv.updated_at
		FROM sample_versions sv
		WHERE sv.sample_request_id = $1
		ORDER BY sv.version_number ASC
	`
	vRows, err := r.db.QueryContext(ctx, versionsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("query sample versions: %w", err)
	}
	defer vRows.Close()

	d.Versions = make([]SampleVersionDetail, 0)
	for vRows.Next() {
		var v SampleVersionDetail
		var revNotes, clientFeedback, priceNotes sql.NullString
		var proposedPrice sql.NullFloat64
		var approvedAt sql.NullTime

		err := vRows.Scan(
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
		)
		if err != nil {
			return nil, fmt.Errorf("scan sample version: %w", err)
		}

		if revNotes.Valid {
			v.RevisionNotes = &revNotes.String
		}
		if clientFeedback.Valid {
			v.ClientFeedback = &clientFeedback.String
		}
		if proposedPrice.Valid {
			val := proposedPrice.Float64
			v.ProposedUnitPrice = &val
		}
		if priceNotes.Valid {
			v.PriceNotes = &priceNotes.String
		}
		if approvedAt.Valid {
			v.ApprovedAt = &approvedAt.Time
		}

		// Ambil time studies untuk versi ini
		studies, err := r.listTimeStudiesByVersionID(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		v.Components = studies

		// Ambil attachments untuk versi ini
		atts, err := r.listAttachmentsByVersionID(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		v.Attachments = atts

		d.Versions = append(d.Versions, v)
	}

	return &d, nil
}

// listTimeStudiesByVersionID helper untuk mengambil daftar time study versi tertentu.
func (r *PostgresRepository) listTimeStudiesByVersionID(ctx context.Context, versionID string) ([]SampleTimeStudyItem, error) {
	query := `
		SELECT 
			sts.id,
			sts.sample_version_id,
			sts.component_id,
			c.name AS component_name,
			c.sequence_number,
			sts.observed_seconds,
			sts.allowance_percentage,
			sts.calculated_sam,
			sts.target_per_hour,
			sts.operator_id,
			u.full_name AS operator_name,
			sts.notes,
			sts.created_at
		FROM sample_time_studies sts
		JOIN components c ON sts.component_id = c.id
		LEFT JOIN users u ON sts.operator_id = u.id
		WHERE sts.sample_version_id = $1
		ORDER BY c.sequence_number ASC
	`
	rows, err := r.db.QueryContext(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("list time studies: %w", err)
	}
	defer rows.Close()

	items := make([]SampleTimeStudyItem, 0)
	for rows.Next() {
		var item SampleTimeStudyItem
		var opID, opName, notes sql.NullString
		err := rows.Scan(
			&item.ID,
			&item.SampleVersionID,
			&item.ComponentID,
			&item.ComponentName,
			&item.SequenceNumber,
			&item.ObservedSeconds,
			&item.AllowancePercentage,
			&item.CalculatedSAM,
			&item.TargetPerHour,
			&opID,
			&opName,
			&notes,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan time study item: %w", err)
		}
		if opID.Valid {
			item.OperatorID = &opID.String
		}
		if opName.Valid {
			item.OperatorName = &opName.String
		}
		if notes.Valid {
			item.Notes = &notes.String
		}
		items = append(items, item)
	}

	return items, nil
}

// CreateSampleVersion menerbitkan revisi versi baru (misal v1 -> v2) dan mengunci versi lama menjadi REVISED.
func (r *PostgresRepository) CreateSampleVersion(ctx context.Context, sampleID string, newVer *SampleVersionDetail, copyFromPrev bool) (*SampleVersionDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Kunci sample request
	var currentStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM sample_requests WHERE id = $1 FOR UPDATE", sampleID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotFound
		}
		return nil, fmt.Errorf("lock sample request: %w", err)
	}

	if currentStatus == StatusConvertedToOrder {
		return nil, ErrSampleAlreadyConverted
	}

	// 2. Ambil versi aktif / tertinggi terakhir
	var prevID string
	var prevVerNum int
	var prevStatus, prevCurrency string
	var prevTotalSAM float64
	var prevProposedPrice sql.NullFloat64
	var prevPriceNotes sql.NullString

	prevVerQuery := `
		SELECT id, version_number, status, total_sam, proposed_unit_price, currency, price_notes
		FROM sample_versions
		WHERE sample_request_id = $1
		ORDER BY version_number DESC
		LIMIT 1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, prevVerQuery, sampleID).Scan(
		&prevID, &prevVerNum, &prevStatus, &prevTotalSAM, &prevProposedPrice, &prevCurrency, &prevPriceNotes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("get prev version: %w", err)
	}

	// 3. Kunci versi sebelumnya menjadi REVISED
	_, err = tx.ExecContext(ctx, "UPDATE sample_versions SET status = $1, updated_at = NOW() WHERE id = $2", StatusRevised, prevID)
	if err != nil {
		return nil, fmt.Errorf("lock prev version: %w", err)
	}

	// 4. Hitung nomor dan label versi baru
	nextVerNum := prevVerNum + 1
	nextVerLabel := fmt.Sprintf("v%d", nextVerNum)

	// Persiapkan nilai default harga jika tidak disediakan
	currency := "IDR"
	if newVer.Currency != "" {
		currency = newVer.Currency
	} else if prevCurrency != "" {
		currency = prevCurrency
	}

	var proposedPrice *float64 = newVer.ProposedUnitPrice
	if proposedPrice == nil && prevProposedPrice.Valid {
		val := prevProposedPrice.Float64
		proposedPrice = &val
	}

	var priceNotes *string = newVer.PriceNotes
	if priceNotes == nil && prevPriceNotes.Valid {
		val := prevPriceNotes.String
		priceNotes = &val
	}

	totalSAM := 0.00
	if copyFromPrev {
		totalSAM = prevTotalSAM
	}

	// 5. Insert versi baru
	var newVersionID string
	var createdAt, updatedAt time.Time
	insertVerQuery := `
		INSERT INTO sample_versions (
			sample_request_id, version_number, version_label, status, revision_notes,
			total_sam, proposed_unit_price, currency, price_notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, insertVerQuery,
		sampleID, nextVerNum, nextVerLabel, StatusDraft, newVer.RevisionNotes,
		totalSAM, proposedPrice, currency, priceNotes,
	).Scan(&newVersionID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert new sample version: %w", err)
	}

	// 6. Jika copyFromPrev = true, duplikasi catatan time study versi sebelumnya
	if copyFromPrev {
		copyStudiesQuery := `
			INSERT INTO sample_time_studies (
				id, sample_version_id, component_id, observed_seconds, allowance_percentage,
				calculated_sam, target_per_hour, operator_id, notes, created_at, updated_at
			)
			SELECT 
				gen_random_uuid(), $1, component_id, observed_seconds, allowance_percentage,
				calculated_sam, target_per_hour, operator_id, notes, NOW(), NOW()
			FROM sample_time_studies
			WHERE sample_version_id = $2
		`
		_, err = tx.ExecContext(ctx, copyStudiesQuery, newVersionID, prevID)
		if err != nil {
			return nil, fmt.Errorf("copy time studies from prev: %w", err)
		}
	}

	// 7. Update status sample_requests menjadi IN_DEVELOPMENT
	_, err = tx.ExecContext(ctx, "UPDATE sample_requests SET status = $1, updated_at = NOW() WHERE id = $2", StatusInDevelopment, sampleID)
	if err != nil {
		return nil, fmt.Errorf("update sample_requests status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetSampleVersionByNumber(ctx, sampleID, nextVerNum)
}

// GetSampleVersionByNumber mengambil detail satu versi sample berdasarkan nomor versinya.
func (r *PostgresRepository) GetSampleVersionByNumber(ctx context.Context, sampleID string, versionNumber int) (*SampleVersionDetail, error) {
	query := `
		SELECT 
			sv.id,
			sv.sample_request_id,
			sv.version_number,
			sv.version_label,
			sv.status,
			sv.revision_notes,
			sv.client_feedback,
			sv.total_sam,
			sv.proposed_unit_price,
			sv.currency,
			sv.price_notes,
			sv.approved_at,
			sv.created_at,
			sv.updated_at
		FROM sample_versions sv
		WHERE sv.sample_request_id = $1 AND sv.version_number = $2
	`

	var v SampleVersionDetail
	var revNotes, clientFeedback, priceNotes sql.NullString
	var proposedPrice sql.NullFloat64
	var approvedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, sampleID, versionNumber).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("get sample version: %w", err)
	}

	if revNotes.Valid {
		v.RevisionNotes = &revNotes.String
	}
	if clientFeedback.Valid {
		v.ClientFeedback = &clientFeedback.String
	}
	if proposedPrice.Valid {
		val := proposedPrice.Float64
		v.ProposedUnitPrice = &val
	}
	if priceNotes.Valid {
		v.PriceNotes = &priceNotes.String
	}
	if approvedAt.Valid {
		v.ApprovedAt = &approvedAt.Time
	}

	studies, err := r.listTimeStudiesByVersionID(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Components = studies

	atts, err := r.listAttachmentsByVersionID(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Attachments = atts

	return &v, nil
}

// RecordTimeStudy menginput atau memperbarui observasi stopwatch time study per komponen pada versi sample tertentu.
func (r *PostgresRepository) RecordTimeStudy(ctx context.Context, sampleID string, versionNumber int, study *SampleTimeStudyItem) (*SampleTimeStudyItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Verifikasi sample request dan versi
	var versionID, versionStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT sv.id, sv.status 
		FROM sample_versions sv
		WHERE sv.sample_request_id = $1 AND sv.version_number = $2
		FOR UPDATE
	`, sampleID, versionNumber).Scan(&versionID, &versionStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("check version: %w", err)
	}

	// Cegah perubahan jika versi telah dikunci (REVISED atau APPROVED)
	if versionStatus == StatusRevised || versionStatus == StatusApproved {
		return nil, ErrVersionAlreadyLocked
	}

	// 2. Verifikasi komponen terdaftar pada artikel terkait
	var compName string
	var seqNum int
	err = tx.QueryRowContext(ctx, `
		SELECT c.name, c.sequence_number
		FROM components c
		JOIN sample_requests sr ON c.article_id = sr.article_id
		WHERE sr.id = $1 AND c.id = $2
	`, sampleID, study.ComponentID).Scan(&compName, &seqNum)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrComponentNotFound
		}
		return nil, fmt.Errorf("check component: %w", err)
	}

	// 3. Kalkulasi Standard Allowed Minute (SAM) dan Target Per Jam
	// Formula: SAM = (Observed_Seconds * (1 + Allowance% / 100)) / 60.0
	allowance := study.AllowancePercentage
	if allowance <= 0 {
		allowance = 15.00 // Default garment allowance standar industri
	}
	calculatedSAM := (study.ObservedSeconds * (1.0 + (allowance / 100.0))) / 60.0
	// Bulatkan ke 2 desimal
	calculatedSAM = math.Round(calculatedSAM*100) / 100

	targetPerHour := 0
	if calculatedSAM > 0 {
		targetPerHour = int(math.Floor(60.0 / calculatedSAM))
	}

	// 4. Upsert sample_time_studies
	upsertQuery := `
		INSERT INTO sample_time_studies (
			sample_version_id, component_id, observed_seconds, allowance_percentage,
			calculated_sam, target_per_hour, operator_id, notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
		)
		ON CONFLICT (sample_version_id, component_id) DO UPDATE SET
			observed_seconds = EXCLUDED.observed_seconds,
			allowance_percentage = EXCLUDED.allowance_percentage,
			calculated_sam = EXCLUDED.calculated_sam,
			target_per_hour = EXCLUDED.target_per_hour,
			operator_id = EXCLUDED.operator_id,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id, created_at
	`
	var studyID string
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, upsertQuery,
		versionID, study.ComponentID, study.ObservedSeconds, allowance,
		calculatedSAM, targetPerHour, study.OperatorID, study.Notes,
	).Scan(&studyID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("upsert sample_time_study: %w", err)
	}

	// 5. Rekalkulasi Total SAM pada sample_versions
	updateTotalSAMQuery := `
		UPDATE sample_versions
		SET total_sam = (
			SELECT COALESCE(SUM(calculated_sam), 0.00)
			FROM sample_time_studies
			WHERE sample_version_id = $1
		),
		updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, updateTotalSAMQuery, versionID)
	if err != nil {
		return nil, fmt.Errorf("recalculate total sam: %w", err)
	}

	// Pastikan status sample_requests adalah IN_DEVELOPMENT jika masih DRAFT
	_, _ = tx.ExecContext(ctx, "UPDATE sample_requests SET status = $1 WHERE id = $2 AND status = $3", StatusInDevelopment, sampleID, StatusDraft)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	study.ID = studyID
	study.SampleVersionID = versionID
	study.ComponentName = compName
	study.SequenceNumber = seqNum
	study.AllowancePercentage = allowance
	study.CalculatedSAM = calculatedSAM
	study.TargetPerHour = targetPerHour
	study.CreatedAt = createdAt

	return study, nil
}

// ListTimeStudies mengambil seluruh time study untuk versi sample tertentu.
func (r *PostgresRepository) ListTimeStudies(ctx context.Context, sampleID string, versionNumber int) ([]SampleTimeStudyItem, error) {
	var versionID string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM sample_versions WHERE sample_request_id = $1 AND version_number = $2
	`, sampleID, versionNumber).Scan(&versionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("find version: %w", err)
	}

	return r.listTimeStudiesByVersionID(ctx, versionID)
}

// SetCommercialRate menetapkan tarif maklon per pcs ke klien pada versi sample.
func (r *PostgresRepository) SetCommercialRate(ctx context.Context, sampleID string, versionNumber int, unitPrice float64, currency, priceNotes string) (*SampleVersionDetail, error) {
	if currency == "" {
		currency = "IDR"
	}

	query := `
		UPDATE sample_versions
		SET proposed_unit_price = $1,
		    currency = $2,
		    price_notes = $3,
		    updated_at = NOW()
		WHERE sample_request_id = $4 AND version_number = $5
	`
	res, err := r.db.ExecContext(ctx, query, unitPrice, currency, priceNotes, sampleID, versionNumber)
	if err != nil {
		return nil, fmt.Errorf("set commercial rate: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrVersionNotFound
	}

	return r.GetSampleVersionByNumber(ctx, sampleID, versionNumber)
}

// UpdateSampleStatus memperbarui status sample_requests dan versi aktifnya.
func (r *PostgresRepository) UpdateSampleStatus(ctx context.Context, sampleID string, status string, notes *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Update status di sample_requests
	res, err := tx.ExecContext(ctx, "UPDATE sample_requests SET status = $1, updated_at = NOW() WHERE id = $2", status, sampleID)
	if err != nil {
		return fmt.Errorf("update sample request status: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrSampleNotFound
	}

	// Update versi tertinggi terakhir
	if status == StatusInReview {
		_, err = tx.ExecContext(ctx, `
			UPDATE sample_versions 
			SET status = $1, updated_at = NOW() 
			WHERE sample_request_id = $2 
			  AND version_number = (SELECT MAX(version_number) FROM sample_versions WHERE sample_request_id = $2)
		`, StatusInReview, sampleID)
		if err != nil {
			return fmt.Errorf("update active version status: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateSampleRequest memperbarui informasi dasar permintaan sample (klien, artikel, nama artikel, target tanggal, catatan awal).
func (r *PostgresRepository) UpdateSampleRequest(ctx context.Context, sampleID string, payload UpdateSampleRequestPayload) (*SampleRequestDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Cek keberadaan sample dan ambil article_id saat ini
	var currentArticleID string
	err = tx.QueryRowContext(ctx, "SELECT article_id FROM sample_requests WHERE id = $1 FOR UPDATE", sampleID).Scan(&currentArticleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotFound
		}
		return nil, fmt.Errorf("lock sample request: %w", err)
	}

	// 2. Validasi client jika disediakan
	if payload.ClientID != nil && strings.TrimSpace(*payload.ClientID) != "" {
		var dummy string
		err = tx.QueryRowContext(ctx, "SELECT id FROM clients WHERE id = $1", strings.TrimSpace(*payload.ClientID)).Scan(&dummy)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrClientNotFound
			}
			return nil, fmt.Errorf("check client: %w", err)
		}
	}

	targetArticleID := currentArticleID
	// 3. Validasi article jika disediakan
	if payload.ArticleID != nil && strings.TrimSpace(*payload.ArticleID) != "" {
		var dummy string
		targetArticleID = strings.TrimSpace(*payload.ArticleID)
		err = tx.QueryRowContext(ctx, "SELECT id FROM articles WHERE id = $1", targetArticleID).Scan(&dummy)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrArticleNotFound
			}
			return nil, fmt.Errorf("check article: %w", err)
		}
	}

	// 4. Update nama/judul artikel jika diminta
	if payload.ArticleTitle != nil && strings.TrimSpace(*payload.ArticleTitle) != "" {
		_, err = tx.ExecContext(ctx, "UPDATE articles SET title = $1, updated_at = NOW() WHERE id = $2", strings.TrimSpace(*payload.ArticleTitle), targetArticleID)
		if err != nil {
			return nil, fmt.Errorf("update article title: %w", err)
		}
	}

	// 5. Update sample_requests
	var sets []string
	var args []interface{}
	argIdx := 1

	if payload.ClientID != nil && strings.TrimSpace(*payload.ClientID) != "" {
		sets = append(sets, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, strings.TrimSpace(*payload.ClientID))
		argIdx++
	}

	if payload.ArticleID != nil && strings.TrimSpace(*payload.ArticleID) != "" {
		sets = append(sets, fmt.Sprintf("article_id = $%d", argIdx))
		args = append(args, strings.TrimSpace(*payload.ArticleID))
		argIdx++
	}

	if payload.TargetDate != nil {
		sets = append(sets, fmt.Sprintf("target_date = $%d", argIdx))
		if strings.TrimSpace(*payload.TargetDate) == "" {
			args = append(args, nil)
		} else {
			args = append(args, strings.TrimSpace(*payload.TargetDate))
		}
		argIdx++
	}

	if payload.InitialNotes != nil {
		sets = append(sets, fmt.Sprintf("initial_notes = $%d", argIdx))
		args = append(args, payload.InitialNotes)
		argIdx++
	}

	if len(sets) > 0 {
		sets = append(sets, "updated_at = NOW()")
		query := fmt.Sprintf("UPDATE sample_requests SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
		args = append(args, sampleID)

		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("update sample_requests: %w", err)
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return nil, ErrSampleNotFound
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetSampleRequestByID(ctx, sampleID)
}

// DeleteSampleRequest menghapus permintaan sample beserta relasi cascaded (versi, time study, attachments) dari database.
func (r *PostgresRepository) DeleteSampleRequest(ctx context.Context, sampleID string) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM sample_requests WHERE id = $1", sampleID)
	if err != nil {
		return fmt.Errorf("delete sample request: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSampleNotFound
	}
	return nil
}

// ConvertToOrder mengonversi sample approved menjadi Order Produksi Massal (BPMN 03 / Skenario Uji 4).
func (r *PostgresRepository) ConvertToOrder(ctx context.Context, sampleID string, payload ConvertSampleToOrderPayload) (*ConvertedMassOrderResponse, error) {
	// 1. Ambil data sample request
	var sampleCode, clientID, articleID, status string
	var convertedOrderID sql.NullString
	querySample := `
		SELECT sample_code, client_id, article_id, status, converted_order_id
		FROM sample_requests
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, querySample, sampleID).Scan(
		&sampleCode, &clientID, &articleID, &status, &convertedOrderID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotFound
		}
		return nil, fmt.Errorf("find sample request: %w", err)
	}

	if convertedOrderID.Valid && convertedOrderID.String != "" {
		return nil, ErrSampleAlreadyConverted
	}

	// 2. Ambil versi sample yang disetujui (APPROVED)
	var versionID, currency string
	var versionNumber int
	var totalSAM float64
	var proposedPrice sql.NullFloat64
	queryVersion := `
		SELECT id, version_number, total_sam, proposed_unit_price, currency
		FROM sample_versions
		WHERE sample_request_id = $1 AND status = 'APPROVED'
		ORDER BY version_number DESC
		LIMIT 1
	`
	err = r.db.QueryRowContext(ctx, queryVersion, sampleID).Scan(
		&versionID, &versionNumber, &totalSAM, &proposedPrice, &currency,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSampleNotApproved
		}
		return nil, fmt.Errorf("find approved sample version: %w", err)
	}

	// 3. Tentukan harga satuan (unit price)
	var unitPrice float64
	if payload.UnitPrice != nil && *payload.UnitPrice > 0 {
		unitPrice = *payload.UnitPrice
	} else if proposedPrice.Valid && proposedPrice.Float64 > 0 {
		unitPrice = proposedPrice.Float64
	} else {
		return nil, errors.New("harga satuan (unit price) belum ditentukan baik pada payload maupun versi sample yang disetujui")
	}

	totalAmount := math.Round(float64(payload.TargetQuantity)*unitPrice*100) / 100

	// 4. Eksekusi transaksi pembuatan order massal dan penguncian sample
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 4a. Insert ke orders
	var orderID string
	insertOrderQuery := `
		INSERT INTO orders (
			order_number, client_id, article_id, sample_version_id,
			target_quantity, start_date, target_date, unit_price, total_amount,
			status, notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, CURRENT_DATE, $6, $7, $8,
			'IN_PRODUCTION', $9, NOW(), NOW()
		) RETURNING id
	`
	err = tx.QueryRowContext(ctx, insertOrderQuery,
		payload.OrderNumber, clientID, articleID, versionID,
		payload.TargetQuantity, payload.TargetDate, unitPrice, totalAmount,
		payload.Notes,
	).Scan(&orderID)
	if err != nil {
		return nil, fmt.Errorf("insert mass order: %w", err)
	}

	// 4b. Sinkronisasi Standard SAM komponen masterdata dari hasil time study versi approved
	_, err = tx.ExecContext(ctx, `
		UPDATE components c
		SET standard_sam = sts.calculated_sam,
		    updated_at = NOW()
		FROM sample_time_studies sts
		WHERE sts.sample_version_id = $1 AND sts.component_id = c.id
	`, versionID)
	if err != nil {
		return nil, fmt.Errorf("sync component standard SAM: %w", err)
	}

	// 4c. Update sample_requests menandai telah dikonversi ke order massal
	_, err = tx.ExecContext(ctx, `
		UPDATE sample_requests
		SET converted_order_id = $1,
		    status = 'CONVERTED_TO_ORDER',
		    updated_at = NOW()
		WHERE id = $2
	`, orderID, sampleID)
	if err != nil {
		return nil, fmt.Errorf("update sample_requests converted: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &ConvertedMassOrderResponse{
		OrderID:         orderID,
		OrderNumber:     payload.OrderNumber,
		SampleCode:      sampleCode,
		ApprovedVersion: versionNumber,
		TargetQuantity:  payload.TargetQuantity,
		UnitPrice:       unitPrice,
		TotalAmount:     totalAmount,
		Currency:        currency,
		Status:          "IN_PRODUCTION",
		SpecSnapshotSAM: totalSAM,
	}, nil
}

// SaveAttachment menyimpan metadata lampiran foto sampel ke database lukla_db.
func (r *PostgresRepository) SaveAttachment(ctx context.Context, att *SampleAttachment) (*SampleAttachment, error) {
	query := `
		INSERT INTO sample_attachments (
			id, sample_request_id, sample_version_id, file_name, s3_bucket, s3_key, file_url,
			file_size, mime_type, caption, attachment_type, uploaded_by, uploader_name, uploader_role, created_at
		) VALUES (
			COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, NOW()
		)
		RETURNING id, created_at
	`

	var uploadedByVal *string
	if att.UploadedBy != nil && *att.UploadedBy != "" {
		uploadedByVal = att.UploadedBy
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		att.ID,
		att.SampleRequestID,
		att.SampleVersionID,
		att.FileName,
		att.S3Bucket,
		att.S3Key,
		att.FileURL,
		att.FileSize,
		att.MimeType,
		att.Caption,
		att.AttachmentType,
		uploadedByVal,
		att.UploaderName,
		att.UploaderRole,
	).Scan(&att.ID, &att.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert sample attachment: %w", err)
	}

	return att, nil
}

// listAttachmentsByVersionID helper mengambil semua lampiran foto yang terikat pada sample_version_id.
func (r *PostgresRepository) listAttachmentsByVersionID(ctx context.Context, versionID string) ([]SampleAttachment, error) {
	query := `
		SELECT 
			id, sample_request_id, sample_version_id, file_name, s3_bucket, s3_key, file_url,
			file_size, mime_type, caption, attachment_type, uploaded_by, uploader_name, uploader_role, created_at
		FROM sample_attachments
		WHERE sample_version_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, versionID)
	if err != nil {
		return nil, fmt.Errorf("query attachments by version id: %w", err)
	}
	defer rows.Close()

	attachments := make([]SampleAttachment, 0)
	for rows.Next() {
		var a SampleAttachment
		var versionIDNull, captionNull, uploadedByNull sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.SampleRequestID,
			&versionIDNull,
			&a.FileName,
			&a.S3Bucket,
			&a.S3Key,
			&a.FileURL,
			&a.FileSize,
			&a.MimeType,
			&captionNull,
			&a.AttachmentType,
			&uploadedByNull,
			&a.UploaderName,
			&a.UploaderRole,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sample attachment: %w", err)
		}

		if versionIDNull.Valid {
			a.SampleVersionID = &versionIDNull.String
		}
		if captionNull.Valid {
			a.Caption = &captionNull.String
		}
		if uploadedByNull.Valid {
			a.UploadedBy = &uploadedByNull.String
		}

		attachments = append(attachments, a)
	}

	return attachments, nil
}

// ListAttachmentsByVersion mengambil semua lampiran foto pada versi nomor tertentu dari suatu sample request.
func (r *PostgresRepository) ListAttachmentsByVersion(ctx context.Context, sampleRequestID string, versionNumber int) ([]SampleAttachment, error) {
	query := `
		SELECT 
			sa.id, sa.sample_request_id, sa.sample_version_id, sa.file_name, sa.s3_bucket, sa.s3_key, sa.file_url,
			sa.file_size, sa.mime_type, sa.caption, sa.attachment_type, sa.uploaded_by, sa.uploader_name, sa.uploader_role, sa.created_at
		FROM sample_attachments sa
		JOIN sample_versions sv ON sa.sample_version_id = sv.id
		WHERE sa.sample_request_id = $1 AND sv.version_number = $2
		ORDER BY sa.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sampleRequestID, versionNumber)
	if err != nil {
		return nil, fmt.Errorf("query attachments by version number: %w", err)
	}
	defer rows.Close()

	attachments := make([]SampleAttachment, 0)
	for rows.Next() {
		var a SampleAttachment
		var versionIDNull, captionNull, uploadedByNull sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.SampleRequestID,
			&versionIDNull,
			&a.FileName,
			&a.S3Bucket,
			&a.S3Key,
			&a.FileURL,
			&a.FileSize,
			&a.MimeType,
			&captionNull,
			&a.AttachmentType,
			&uploadedByNull,
			&a.UploaderName,
			&a.UploaderRole,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sample attachment: %w", err)
		}

		if versionIDNull.Valid {
			a.SampleVersionID = &versionIDNull.String
		}
		if captionNull.Valid {
			a.Caption = &captionNull.String
		}
		if uploadedByNull.Valid {
			a.UploadedBy = &uploadedByNull.String
		}

		attachments = append(attachments, a)
	}

	return attachments, nil
}

// ListAttachmentsBySampleID mengambil seluruh berkas lampiran yang terkait dengan suatu sample request.
func (r *PostgresRepository) ListAttachmentsBySampleID(ctx context.Context, sampleRequestID string) ([]SampleAttachment, error) {
	query := `
		SELECT 
			id, sample_request_id, sample_version_id, file_name, s3_bucket, s3_key, file_url,
			file_size, mime_type, caption, attachment_type, uploaded_by, uploader_name, uploader_role, created_at
		FROM sample_attachments
		WHERE sample_request_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sampleRequestID)
	if err != nil {
		return nil, fmt.Errorf("query attachments by sample request id: %w", err)
	}
	defer rows.Close()

	attachments := make([]SampleAttachment, 0)
	for rows.Next() {
		var a SampleAttachment
		var versionIDNull, captionNull, uploadedByNull sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.SampleRequestID,
			&versionIDNull,
			&a.FileName,
			&a.S3Bucket,
			&a.S3Key,
			&a.FileURL,
			&a.FileSize,
			&a.MimeType,
			&captionNull,
			&a.AttachmentType,
			&uploadedByNull,
			&a.UploaderName,
			&a.UploaderRole,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan sample attachment: %w", err)
		}

		if versionIDNull.Valid {
			a.SampleVersionID = &versionIDNull.String
		}
		if captionNull.Valid {
			a.Caption = &captionNull.String
		}
		if uploadedByNull.Valid {
			a.UploadedBy = &uploadedByNull.String
		}

		attachments = append(attachments, a)
	}

	return attachments, nil
}

// GetAttachmentByID mengambil metadata lengkap satu berkas lampiran berdasarkan ID-nya.
func (r *PostgresRepository) GetAttachmentByID(ctx context.Context, id string) (*SampleAttachment, error) {
	query := `
		SELECT 
			id, sample_request_id, sample_version_id, file_name, s3_bucket, s3_key, file_url,
			file_size, mime_type, caption, attachment_type, uploaded_by, uploader_name, uploader_role, created_at
		FROM sample_attachments
		WHERE id = $1
	`

	var a SampleAttachment
	var versionIDNull, captionNull, uploadedByNull sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.SampleRequestID,
		&versionIDNull,
		&a.FileName,
		&a.S3Bucket,
		&a.S3Key,
		&a.FileURL,
		&a.FileSize,
		&a.MimeType,
		&captionNull,
		&a.AttachmentType,
		&uploadedByNull,
		&a.UploaderName,
		&a.UploaderRole,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("lampiran foto sampel tidak ditemukan")
		}
		return nil, fmt.Errorf("get sample attachment by id: %w", err)
	}

	if versionIDNull.Valid {
		a.SampleVersionID = &versionIDNull.String
	}
	if captionNull.Valid {
		a.Caption = &captionNull.String
	}
	if uploadedByNull.Valid {
		a.UploadedBy = &uploadedByNull.String
	}

	return &a, nil
}

// DeleteAttachment menghapus data lampiran dari tabel sample_attachments.
func (r *PostgresRepository) DeleteAttachment(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sample_attachments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sample attachment: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("lampiran foto sampel tidak ditemukan")
	}
	return nil
}


