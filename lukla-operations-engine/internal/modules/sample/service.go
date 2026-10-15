package sample

import (
	"context"
	"fmt"
	"strings"

	"github.com/lukla/operations-engine/internal/modules/notification"
)

// Service mengelola logika bisnis alur kerja sample prototipe, versioning, dan time study.
type Service struct {
	repo            Repository
	notificationSvc *notification.Service
}

// NewService membuat instance Service baru.
func NewService(repo Repository, notificationSvc *notification.Service) *Service {
	return &Service{
		repo:            repo,
		notificationSvc: notificationSvc,
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

