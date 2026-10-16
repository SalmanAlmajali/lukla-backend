package sample_test

import (
	"context"
	"testing"
	"time"

	"github.com/lukla/operations-engine/internal/modules/sample"
)

// mockRepository adalah mock repository untuk unit testing sample.Service.
type mockRepository struct {
	samples      map[string]*sample.SampleRequestDetail
	versions     map[string][]*sample.SampleVersionDetail
	timeStudies  map[string][]sample.SampleTimeStudyItem
	updatedStats map[string]string
}

func newMockRepo() *mockRepository {
	return &mockRepository{
		samples:      make(map[string]*sample.SampleRequestDetail),
		versions:     make(map[string][]*sample.SampleVersionDetail),
		timeStudies:  make(map[string][]sample.SampleTimeStudyItem),
		updatedStats: make(map[string]string),
	}
}

func (m *mockRepository) CreateSampleRequest(ctx context.Context, s *sample.SampleRequestDetail, initialVer *sample.SampleVersionDetail) (*sample.SampleRequestDetail, error) {
	s.ID = "sample-uuid-1"
	s.SampleCode = "SMP-202610-001"
	s.Status = sample.StatusDraft
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	ver1 := sample.SampleVersionDetail{
		ID:            "ver-uuid-1",
		VersionNumber: 1,
		VersionLabel:  "v1",
		Status:        sample.StatusDraft,
		TotalSAM:      0,
		Currency:      "IDR",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Components:    []sample.SampleTimeStudyItem{},
	}
	s.Versions = []sample.SampleVersionDetail{ver1}

	m.samples[s.ID] = s
	m.versions[s.ID] = []*sample.SampleVersionDetail{&ver1}

	return s, nil
}

func (m *mockRepository) ListSampleRequests(ctx context.Context, clientID, status, search string, page, limit int) ([]sample.SampleRequestSummary, int, error) {
	var list []sample.SampleRequestSummary
	for _, s := range m.samples {
		list = append(list, sample.SampleRequestSummary{
			ID:             s.ID,
			SampleCode:     s.SampleCode,
			ClientID:       s.ClientID,
			ArticleID:      s.ArticleID,
			CurrentVersion: len(s.Versions),
			Status:         s.Status,
			CreatedAt:      s.CreatedAt,
		})
	}
	return list, len(list), nil
}

func (m *mockRepository) GetSampleRequestByID(ctx context.Context, id string) (*sample.SampleRequestDetail, error) {
	s, ok := m.samples[id]
	if !ok {
		return nil, sample.ErrSampleNotFound
	}
	return s, nil
}

func (m *mockRepository) CreateSampleVersion(ctx context.Context, sampleID string, newVer *sample.SampleVersionDetail, copyFromPrev bool) (*sample.SampleVersionDetail, error) {
	s, ok := m.samples[sampleID]
	if !ok {
		return nil, sample.ErrSampleNotFound
	}

	// Kunci versi lama
	for _, v := range m.versions[sampleID] {
		v.Status = sample.StatusRevised
	}

	nextNum := len(m.versions[sampleID]) + 1
	ver := &sample.SampleVersionDetail{
		ID:                "ver-uuid-2",
		VersionNumber:     nextNum,
		VersionLabel:      "v2",
		Status:            sample.StatusDraft,
		RevisionNotes:     newVer.RevisionNotes,
		TotalSAM:          newVer.TotalSAM,
		ProposedUnitPrice: newVer.ProposedUnitPrice,
		Currency:          newVer.Currency,
		PriceNotes:        newVer.PriceNotes,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	m.versions[sampleID] = append(m.versions[sampleID], ver)
	s.Versions = append(s.Versions, *ver)
	s.Status = sample.StatusInDevelopment

	return ver, nil
}

func (m *mockRepository) GetSampleVersionByNumber(ctx context.Context, sampleID string, versionNumber int) (*sample.SampleVersionDetail, error) {
	vers, ok := m.versions[sampleID]
	if !ok {
		return nil, sample.ErrSampleNotFound
	}
	for _, v := range vers {
		if v.VersionNumber == versionNumber {
			return v, nil
		}
	}
	return nil, sample.ErrVersionNotFound
}

func (m *mockRepository) RecordTimeStudy(ctx context.Context, sampleID string, versionNumber int, study *sample.SampleTimeStudyItem) (*sample.SampleTimeStudyItem, error) {
	// Formula: SAM = (Observed_Seconds * (1 + Allowance% / 100)) / 60.0
	allowance := study.AllowancePercentage
	if allowance <= 0 {
		allowance = 15.00
	}
	calculatedSAM := (study.ObservedSeconds * (1.0 + (allowance / 100.0))) / 60.0
	targetPerHour := int(60.0 / calculatedSAM)

	study.ID = "study-uuid-1"
	study.CalculatedSAM = calculatedSAM
	study.TargetPerHour = targetPerHour
	study.ComponentName = "Jahit Kerah & Placket"
	study.SequenceNumber = 1

	return study, nil
}

func (m *mockRepository) ListTimeStudies(ctx context.Context, sampleID string, versionNumber int) ([]sample.SampleTimeStudyItem, error) {
	return m.timeStudies[sampleID], nil
}

func (m *mockRepository) SetCommercialRate(ctx context.Context, sampleID string, versionNumber int, unitPrice float64, currency, priceNotes string) (*sample.SampleVersionDetail, error) {
	v, err := m.GetSampleVersionByNumber(ctx, sampleID, versionNumber)
	if err != nil {
		return nil, err
	}
	v.ProposedUnitPrice = &unitPrice
	v.Currency = currency
	v.PriceNotes = &priceNotes
	return v, nil
}

func (m *mockRepository) UpdateSampleStatus(ctx context.Context, sampleID string, status string, notes *string) error {
	s, ok := m.samples[sampleID]
	if !ok {
		return sample.ErrSampleNotFound
	}
	s.Status = status
	m.updatedStats[sampleID] = status
	return nil
}

func (m *mockRepository) ConvertToOrder(ctx context.Context, sampleID string, payload sample.ConvertSampleToOrderPayload) (*sample.ConvertedMassOrderResponse, error) {
	s, ok := m.samples[sampleID]
	if !ok {
		return nil, sample.ErrSampleNotFound
	}
	if s.ConvertedOrderID != nil {
		return nil, sample.ErrSampleAlreadyConverted
	}
	if s.Status != sample.StatusApproved {
		return nil, sample.ErrSampleNotApproved
	}

	unitPrice := 45000.00
	if payload.UnitPrice != nil && *payload.UnitPrice > 0 {
		unitPrice = *payload.UnitPrice
	}

	orderID := "order-converted-uuid-1"
	s.ConvertedOrderID = &orderID
	s.Status = sample.StatusConvertedToOrder

	return &sample.ConvertedMassOrderResponse{
		OrderID:         orderID,
		OrderNumber:     payload.OrderNumber,
		SampleCode:      s.SampleCode,
		ApprovedVersion: 2,
		TargetQuantity:  payload.TargetQuantity,
		UnitPrice:       unitPrice,
		TotalAmount:     float64(payload.TargetQuantity) * unitPrice,
		Currency:        "IDR",
		Status:          "IN_PRODUCTION",
		SpecSnapshotSAM: 13.80,
	}, nil
}

func (m *mockRepository) SaveAttachment(ctx context.Context, att *sample.SampleAttachment) (*sample.SampleAttachment, error) {
	att.ID = "att-uuid-1"
	att.CreatedAt = time.Now()
	return att, nil
}

func (m *mockRepository) ListAttachmentsByVersion(ctx context.Context, sampleRequestID string, versionNumber int) ([]sample.SampleAttachment, error) {
	return []sample.SampleAttachment{}, nil
}

func (m *mockRepository) GetAttachmentByID(ctx context.Context, id string) (*sample.SampleAttachment, error) {
	return &sample.SampleAttachment{
		ID:              id,
		SampleRequestID: "sample-uuid-1",
		FileName:        "test.jpg",
		S3Key:           "samples/SMP-001/test.jpg",
		MimeType:        "image/jpeg",
	}, nil
}

func (m *mockRepository) DeleteAttachment(ctx context.Context, id string) error {
	return nil
}

func TestSampleService_CreateSampleRequest_Success(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	notes := "Prototipe sample pertama"
	res, err := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:     "client-uuid-1",
		ArticleID:    "article-uuid-1",
		InitialNotes: &notes,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SampleCode != "SMP-202610-001" {
		t.Errorf("expected sample code SMP-202610-001, got %s", res.SampleCode)
	}
	if len(res.Versions) != 1 {
		t.Fatalf("expected 1 initial version, got %d", len(res.Versions))
	}
	if res.Versions[0].VersionLabel != "v1" {
		t.Errorf("expected v1, got %s", res.Versions[0].VersionLabel)
	}
}

func TestSampleService_Versioning_v1_to_v2(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	// Buat sample v1
	created, err := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:  "client-uuid-1",
		ArticleID: "article-uuid-1",
	})
	if err != nil {
		t.Fatalf("create sample failed: %v", err)
	}

	// Terbitkan versi v2
	v2, err := svc.CreateSampleVersion(ctx, created.ID, sample.CreateSampleVersionPayload{
		RevisionNotes: "Revisi pola saku dan lingkar dada",
	})
	if err != nil {
		t.Fatalf("create v2 failed: %v", err)
	}

	if v2.VersionNumber != 2 {
		t.Errorf("expected version number 2, got %d", v2.VersionNumber)
	}
	if v2.VersionLabel != "v2" {
		t.Errorf("expected version label v2, got %s", v2.VersionLabel)
	}

	// Verifikasi versi v1 telah berstatus REVISED
	v1, err := svc.GetSampleVersion(ctx, created.ID, 1)
	if err != nil {
		t.Fatalf("get v1 failed: %v", err)
	}
	if v1.Status != sample.StatusRevised {
		t.Errorf("expected v1 status to be REVISED, got %s", v1.Status)
	}
}

func TestSampleService_RecordTimeStudy_SAM_Calculation(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	created, _ := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:  "client-uuid-1",
		ArticleID: "article-uuid-1",
	})

	allowance := 15.0
	notes := "Jahit mesin overlock jarum dua"
	study, err := svc.RecordTimeStudy(ctx, created.ID, 1, sample.RecordTimeStudyPayload{
		ComponentID:         "comp-uuid-1",
		ObservedSeconds:     480.0,
		AllowancePercentage: &allowance,
		Notes:               &notes,
	})
	if err != nil {
		t.Fatalf("record time study failed: %v", err)
	}

	// Expected SAM: (480 * 1.15) / 60 = 552 / 60 = 9.20
	if study.CalculatedSAM < 9.19 || study.CalculatedSAM > 9.21 {
		t.Errorf("expected calculated SAM ~9.20, got %.2f", study.CalculatedSAM)
	}
	// Expected Target Per Hour: 60 / 9.2 = 6
	if study.TargetPerHour != 6 {
		t.Errorf("expected target per hour 6, got %d", study.TargetPerHour)
	}
}

func TestSampleService_SetCommercialRate(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	created, _ := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:  "client-uuid-1",
		ArticleID: "article-uuid-1",
	})

	priceNotes := "Tarif CMT standar ekspor"
	ver, err := svc.SetCommercialRate(ctx, created.ID, 1, sample.SetCommercialRatePayload{
		ProposedUnitPrice: 45000.00,
		PriceNotes:        &priceNotes,
	})
	if err != nil {
		t.Fatalf("set commercial rate failed: %v", err)
	}

	if ver.ProposedUnitPrice == nil || *ver.ProposedUnitPrice != 45000.00 {
		t.Errorf("expected proposed unit price 45000, got %v", ver.ProposedUnitPrice)
	}
}

func TestSampleService_SubmitReview(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	created, _ := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:  "client-uuid-1",
		ArticleID: "article-uuid-1",
	})

	notes := "Sample fisik v1 telah dikirim via ekspedisi"
	err := svc.SubmitReview(ctx, created.ID, &notes)
	if err != nil {
		t.Fatalf("submit review failed: %v", err)
	}

	if repo.updatedStats[created.ID] != sample.StatusInReview {
		t.Errorf("expected sample status IN_REVIEW, got %s", repo.updatedStats[created.ID])
	}
}

func TestSampleService_ConvertToOrder(t *testing.T) {
	repo := newMockRepo()
	svc := sample.NewService(repo, nil)

	ctx := context.Background()
	created, _ := svc.CreateSampleRequest(ctx, sample.CreateSampleRequestPayload{
		ClientID:  "client-uuid-1",
		ArticleID: "article-uuid-1",
	})

	// Set sample to APPROVED
	_ = repo.UpdateSampleStatus(ctx, created.ID, sample.StatusApproved, nil)

	unitPrice := 45000.00
	notes := "Konversi PO Massal"
	res, err := svc.ConvertToOrder(ctx, created.ID, sample.ConvertSampleToOrderPayload{
		OrderNumber:    "PO-2026-10-003",
		TargetQuantity: 2000,
		TargetDate:     "2026-12-01",
		UnitPrice:      &unitPrice,
		Notes:          &notes,
	})
	if err != nil {
		t.Fatalf("convert to order failed: %v", err)
	}

	if res.OrderNumber != "PO-2026-10-003" {
		t.Errorf("expected order number PO-2026-10-003, got %s", res.OrderNumber)
	}
	if res.TargetQuantity != 2000 {
		t.Errorf("expected quantity 2000, got %d", res.TargetQuantity)
	}
	if res.TotalAmount != 90000000.00 {
		t.Errorf("expected total amount 90000000, got %v", res.TotalAmount)
	}
}

