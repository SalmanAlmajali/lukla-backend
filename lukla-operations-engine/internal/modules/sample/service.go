package sample

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/lukla/operations-engine/internal/modules/notification"
	"github.com/lukla/operations-engine/internal/platform/storage"
)

var (
	ErrAttachmentNotFound = errors.New("lampiran foto tidak ditemukan")
	ErrInvalidFileType    = errors.New("tipe berkas tidak didukung (hanya JPEG, PNG, dan WEBP)")
	ErrFileTooLarge       = errors.New("ukuran berkas melebihi batas maksimum 5MB")
)

// Service mengelola logika bisnis alur kerja sample prototipe, versioning, dan time study.
type Service struct {
	repo            Repository
	notificationSvc *notification.Service
	storage         storage.Storage
}

// NewService membuat instance Service baru.
func NewService(repo Repository, notificationSvc *notification.Service, st ...storage.Storage) *Service {
	var s storage.Storage
	if len(st) > 0 {
		s = st[0]
	}
	return &Service{
		repo:            repo,
		notificationSvc: notificationSvc,
		storage:         s,
	}
}

// CreateSampleRequest membuat permintaan sample baru beserta versi 1 (v1) secara otomatis.
func (s *Service) CreateSampleRequest(ctx context.Context, payload CreateSampleRequestPayload) (*SampleRequestDetail, error) {
	if strings.TrimSpace(payload.ClientID) == "" {
		return nil, ErrClientNotFound
	}
	if strings.TrimSpace(payload.ArticleID) == "" {
		return nil, ErrArticleNotFound
	}

	req := &SampleRequestDetail{
		ClientID:     strings.TrimSpace(payload.ClientID),
		ArticleID:    strings.TrimSpace(payload.ArticleID),
		InitialNotes: payload.InitialNotes,
		TargetDate:   payload.TargetDate,
	}

	return s.repo.CreateSampleRequest(ctx, req, nil)
}

// ListSampleRequests mengambil daftar seluruh permintaan sample dengan filter dan pagination.
func (s *Service) ListSampleRequests(ctx context.Context, clientID, status, search string, page, limit int) ([]SampleRequestSummary, int, error) {
	return s.repo.ListSampleRequests(ctx, strings.TrimSpace(clientID), strings.TrimSpace(status), strings.TrimSpace(search), page, limit)
}

// GetSampleRequestByID mengambil detail lengkap sample beserta riwayat seluruh versi.
func (s *Service) GetSampleRequestByID(ctx context.Context, id string) (*SampleRequestDetail, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrSampleNotFound
	}
	return s.repo.GetSampleRequestByID(ctx, strings.TrimSpace(id))
}

// UpdateSampleRequest memperbarui data permintaan sample (klien, artikel, skema maklon / spesifikasi / catatan awal, target tanggal).
func (s *Service) UpdateSampleRequest(ctx context.Context, sampleID string, payload UpdateSampleRequestPayload) (*SampleRequestDetail, error) {
	sampleID = strings.TrimSpace(sampleID)
	if sampleID == "" {
		return nil, ErrSampleNotFound
	}

	// 1. Ambil detail sample saat ini untuk verifikasi status
	existing, err := s.repo.GetSampleRequestByID(ctx, sampleID)
	if err != nil {
		return nil, err
	}

	// 2. Cegah perubahan jika sample sudah APPROVED atau CONVERTED_TO_ORDER
	if existing.Status == StatusApproved || existing.Status == StatusConvertedToOrder || existing.ConvertedOrderID != nil {
		return nil, ErrSampleCannotBeModified
	}

	return s.repo.UpdateSampleRequest(ctx, sampleID, payload)
}

// DeleteSampleRequest menghapus permintaan sample beserta seluruh versi, time study, dan lampiran foto dari S3 & database.
func (s *Service) DeleteSampleRequest(ctx context.Context, sampleID string) error {
	sampleID = strings.TrimSpace(sampleID)
	if sampleID == "" {
		return ErrSampleNotFound
	}

	// 1. Ambil detail sample saat ini untuk verifikasi status
	existing, err := s.repo.GetSampleRequestByID(ctx, sampleID)
	if err != nil {
		return err
	}

	// 2. Cegah penghapusan jika sample sudah APPROVED atau CONVERTED_TO_ORDER
	if existing.Status == StatusApproved || existing.Status == StatusConvertedToOrder || existing.ConvertedOrderID != nil {
		return ErrSampleCannotBeDeleted
	}

	// 3. Bersihkan seluruh file objek lampiran pada Amazon S3 jika S3 storage aktif
	if s.storage != nil {
		attachments, _ := s.repo.ListAttachmentsBySampleID(ctx, sampleID)
		for _, att := range attachments {
			if att.S3Key != "" {
				_ = s.storage.Delete(ctx, att.S3Key)
			}
		}
	}

	// 4. Hapus data dari database (cascade versi, time study, attachments)
	return s.repo.DeleteSampleRequest(ctx, sampleID)
}

// CreateSampleVersion menerbitkan revisi versi baru (misal v1 -> v2) dan mengunci versi sebelumnya.
func (s *Service) CreateSampleVersion(ctx context.Context, sampleID string, payload CreateSampleVersionPayload) (*SampleVersionDetail, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if strings.TrimSpace(payload.RevisionNotes) == "" {
		return nil, fmt.Errorf("catatan revisi wajib diisi")
	}

	copyFromPrev := true
	if payload.CopyTimeStudiesFromPrevious != nil {
		copyFromPrev = *payload.CopyTimeStudiesFromPrevious
	}

	currency := "IDR"
	if payload.Currency != nil && strings.TrimSpace(*payload.Currency) != "" {
		currency = strings.TrimSpace(*payload.Currency)
	}

	newVer := &SampleVersionDetail{
		RevisionNotes:     &payload.RevisionNotes,
		ProposedUnitPrice: payload.ProposedUnitPrice,
		Currency:          currency,
		PriceNotes:        payload.PriceNotes,
	}

	return s.repo.CreateSampleVersion(ctx, strings.TrimSpace(sampleID), newVer, copyFromPrev)
}

// GetSampleVersion mengambil rincian satu versi sample tertentu.
func (s *Service) GetSampleVersion(ctx context.Context, sampleID string, versionNumber int) (*SampleVersionDetail, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if versionNumber < 1 {
		return nil, ErrVersionNotFound
	}
	return s.repo.GetSampleVersionByNumber(ctx, strings.TrimSpace(sampleID), versionNumber)
}

// RecordTimeStudy mencatat hasil observasi stopwatch time study per komponen pada versi sample.
func (s *Service) RecordTimeStudy(ctx context.Context, sampleID string, versionNumber int, payload RecordTimeStudyPayload) (*SampleTimeStudyItem, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if versionNumber < 1 {
		return nil, ErrVersionNotFound
	}
	if strings.TrimSpace(payload.ComponentID) == "" {
		return nil, ErrComponentNotFound
	}
	if payload.ObservedSeconds <= 0 {
		return nil, fmt.Errorf("durasi observasi detik harus lebih besar dari 0")
	}

	allowance := 15.00
	if payload.AllowancePercentage != nil {
		allowance = *payload.AllowancePercentage
	}

	study := &SampleTimeStudyItem{
		ComponentID:         strings.TrimSpace(payload.ComponentID),
		ObservedSeconds:     payload.ObservedSeconds,
		AllowancePercentage: allowance,
		OperatorID:          payload.OperatorID,
		Notes:               payload.Notes,
	}

	return s.repo.RecordTimeStudy(ctx, strings.TrimSpace(sampleID), versionNumber, study)
}

// UpdateTimeStudy memperbarui hasil observasi stopwatch time study komponen tertentu pada versi sample.
func (s *Service) UpdateTimeStudy(ctx context.Context, sampleID string, versionNumber int, componentID string, payload UpdateTimeStudyPayload) (*SampleTimeStudyItem, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if versionNumber < 1 {
		return nil, ErrVersionNotFound
	}
	if strings.TrimSpace(componentID) == "" {
		return nil, ErrComponentNotFound
	}
	if payload.ObservedSeconds <= 0 {
		return nil, fmt.Errorf("durasi observasi detik harus lebih besar dari 0")
	}

	allowance := 15.00
	if payload.AllowancePercentage != nil {
		allowance = *payload.AllowancePercentage
	}

	study := &SampleTimeStudyItem{
		ComponentID:         strings.TrimSpace(componentID),
		ObservedSeconds:     payload.ObservedSeconds,
		AllowancePercentage: allowance,
		OperatorID:          payload.OperatorID,
		Notes:               payload.Notes,
	}

	return s.repo.UpdateTimeStudy(ctx, strings.TrimSpace(sampleID), versionNumber, strings.TrimSpace(componentID), study)
}

// DeleteTimeStudy menghapus data observasi stopwatch komponen tertentu pada versi sample.
func (s *Service) DeleteTimeStudy(ctx context.Context, sampleID string, versionNumber int, componentID string) error {
	if strings.TrimSpace(sampleID) == "" {
		return ErrSampleNotFound
	}
	if versionNumber < 1 {
		return ErrVersionNotFound
	}
	if strings.TrimSpace(componentID) == "" {
		return ErrComponentNotFound
	}

	return s.repo.DeleteTimeStudy(ctx, strings.TrimSpace(sampleID), versionNumber, strings.TrimSpace(componentID))
}

// ListTimeStudies mengambil daftar seluruh hasil observasi time study pada versi sample tertentu.
func (s *Service) ListTimeStudies(ctx context.Context, sampleID string, versionNumber int) ([]SampleTimeStudyItem, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if versionNumber < 1 {
		return nil, ErrVersionNotFound
	}
	return s.repo.ListTimeStudies(ctx, strings.TrimSpace(sampleID), versionNumber)
}

// SetCommercialRate menetapkan tarif maklon per pcs ke klien pada versi sample.
func (s *Service) SetCommercialRate(ctx context.Context, sampleID string, versionNumber int, payload SetCommercialRatePayload) (*SampleVersionDetail, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if versionNumber < 1 {
		return nil, ErrVersionNotFound
	}
	if payload.ProposedUnitPrice < 0 {
		return nil, fmt.Errorf("harga penawaran maklon tidak boleh negatif")
	}

	currency := "IDR"
	if payload.Currency != nil && strings.TrimSpace(*payload.Currency) != "" {
		currency = strings.TrimSpace(*payload.Currency)
	}

	priceNotes := ""
	if payload.PriceNotes != nil {
		priceNotes = strings.TrimSpace(*payload.PriceNotes)
	}

	return s.repo.SetCommercialRate(ctx, strings.TrimSpace(sampleID), versionNumber, payload.ProposedUnitPrice, currency, priceNotes)
}

// SubmitReview mengajukan sample ke klien untuk ditinjau di portal serta memicu notifikasi otomatis.
func (s *Service) SubmitReview(ctx context.Context, sampleID string, notes *string) error {
	sample, err := s.repo.GetSampleRequestByID(ctx, sampleID)
	if err != nil {
		return err
	}

	err = s.repo.UpdateSampleStatus(ctx, sampleID, StatusInReview, notes)
	if err != nil {
		return err
	}

	// Picu notifikasi in-app ke akun klien jika notificationSvc tersedia
	if s.notificationSvc != nil && sample.ClientID != "" {
		refID := sample.ID
		title := fmt.Sprintf("Sample Siap Ditinjau: %s", sample.SampleCode)
		msg := fmt.Sprintf("Sample prototipe artikel %s (%s) telah selesai diproduksi dan siap ditinjau di portal.", sample.ArticleTitle, sample.SampleCode)
		if notes != nil && strings.TrimSpace(*notes) != "" {
			msg = fmt.Sprintf("%s Catatan: %s", msg, strings.TrimSpace(*notes))
		}

		_, _ = s.notificationSvc.SendNotification(ctx, notification.SendNotificationRequest{
			ClientID:    sample.ClientID,
			Title:       title,
			Message:     msg,
			EventType:   "SAMPLE_READY_FOR_REVIEW",
			ReferenceID: &refID,
		})
	}

	return nil
}

// ConvertToOrder mengonversi sample approved menjadi Order Produksi Massal (BPMN 03 / Skenario Uji 4).
func (s *Service) ConvertToOrder(ctx context.Context, sampleID string, payload ConvertSampleToOrderPayload) (*ConvertedMassOrderResponse, error) {
	if strings.TrimSpace(sampleID) == "" {
		return nil, ErrSampleNotFound
	}
	if strings.TrimSpace(payload.OrderNumber) == "" {
		return nil, fmt.Errorf("nomor order massal wajib diisi")
	}
	if payload.TargetQuantity < 1 {
		return nil, fmt.Errorf("target kuantitas produksi massal minimal 1 pcs")
	}
	if strings.TrimSpace(payload.TargetDate) == "" {
		return nil, fmt.Errorf("target tanggal penyelesaian wajib diisi")
	}
	if payload.UnitPrice != nil && *payload.UnitPrice <= 0 {
		return nil, fmt.Errorf("harga satuan unit price jika diisi harus lebih besar dari 0")
	}

	return s.repo.ConvertToOrder(ctx, strings.TrimSpace(sampleID), payload)
}

// UploadAttachment mengunggah foto lampiran sampel ke Amazon S3 dan mencatat metadata ke database.
func (s *Service) UploadAttachment(
	ctx context.Context,
	sampleID string,
	versionNumber int,
	file io.Reader,
	fileName string,
	fileSize int64,
	mimeType string,
	caption *string,
	attType string,
	uploaderID *string,
	uploaderName string,
	uploaderRole string,
) (*SampleAttachment, error) {
	if s.storage == nil {
		return nil, errors.New("penyimpanan Amazon S3 belum diinisialisasi")
	}

	sampleID = strings.TrimSpace(sampleID)
	if sampleID == "" {
		return nil, ErrSampleNotFound
	}

	// Validasi ukuran berkas (maks. 5MB)
	if fileSize > 5*1024*1024 {
		return nil, ErrFileTooLarge
	}

	// Validasi MIME type
	normalizedMime := strings.ToLower(strings.TrimSpace(mimeType))
	if normalizedMime != "image/jpeg" && normalizedMime != "image/png" && normalizedMime != "image/webp" {
		return nil, ErrInvalidFileType
	}

	// Validasi keberadaan sample dan versi
	sample, err := s.repo.GetSampleRequestByID(ctx, sampleID)
	if err != nil {
		return nil, err
	}

	var targetVersionID *string
	for _, v := range sample.Versions {
		if v.VersionNumber == versionNumber {
			vid := v.ID
			targetVersionID = &vid
			break
		}
	}
	if targetVersionID == nil {
		return nil, ErrVersionNotFound
	}

	// Format nama berkas dan S3 Key aman
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		switch normalizedMime {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		}
	}
	cleanBaseName := strings.ReplaceAll(filepath.Base(fileName), " ", "-")
	uniqueKey := fmt.Sprintf("samples/%s/%s-%s", sample.SampleCode, uuid.New().String()[:8], cleanBaseName)

	// Unggah berkas stream langsung ke Amazon S3
	fileURL, err := s.storage.Upload(ctx, file, uniqueKey, normalizedMime)
	if err != nil {
		return nil, fmt.Errorf("unggah ke Amazon S3 gagal: %w", err)
	}

	if attType == "" {
		attType = "PROTOTYPE_PHOTO"
	}

	if strings.TrimSpace(uploaderName) == "" {
		uploaderName = "System"
	}
	if strings.TrimSpace(uploaderRole) == "" {
		uploaderRole = "SAMPLE_MAKER"
	}

	att := &SampleAttachment{
		SampleRequestID: sampleID,
		SampleVersionID: targetVersionID,
		FileName:        fileName,
		S3Bucket:        s.storage.Bucket(),
		S3Key:           uniqueKey,
		FileURL:         fileURL,
		FileSize:        fileSize,
		MimeType:        normalizedMime,
		Caption:         caption,
		AttachmentType:  attType,
		UploadedBy:      uploaderID,
		UploaderName:    uploaderName,
		UploaderRole:    uploaderRole,
	}

	return s.repo.SaveAttachment(ctx, att)
}

// ListAttachments mengambil seluruh lampiran foto untuk versi tertentu.
func (s *Service) ListAttachments(ctx context.Context, sampleID string, versionNumber int) ([]SampleAttachment, error) {
	return s.repo.ListAttachmentsByVersion(ctx, strings.TrimSpace(sampleID), versionNumber)
}

// GetAttachment mengambil metadata lampiran berdasarkan ID-nya.
func (s *Service) GetAttachment(ctx context.Context, id string) (*SampleAttachment, error) {
	return s.repo.GetAttachmentByID(ctx, strings.TrimSpace(id))
}

// DownloadAttachment mengambil reader stream dari Amazon S3 untuk penyajian ke klien.
func (s *Service) DownloadAttachment(ctx context.Context, id string) (io.ReadCloser, string, int64, error) {
	if s.storage == nil {
		return nil, "", 0, errors.New("penyimpanan Amazon S3 belum diinisialisasi")
	}
	att, err := s.repo.GetAttachmentByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, "", 0, err
	}
	return s.storage.Download(ctx, att.S3Key)
}

// DeleteAttachment menghapus foto dari Amazon S3 dan record di database.
func (s *Service) DeleteAttachment(ctx context.Context, id string) error {
	att, err := s.repo.GetAttachmentByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}

	if s.storage != nil && att.S3Key != "" {
		_ = s.storage.Delete(ctx, att.S3Key)
	}

	return s.repo.DeleteAttachment(ctx, att.ID)
}


