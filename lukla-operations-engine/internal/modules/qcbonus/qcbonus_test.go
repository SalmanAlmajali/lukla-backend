package qcbonus

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/lukla/operations-engine/internal/platform/authn"
)

type mockQCBonusRepo struct {
	config       *QCBonusConfig
	recipients   []QCBonusRecipient
	periods      map[string]*QCBonusPeriodDetail
	passedPcsMap map[string]int
}

func newMockQCBonusRepo() *mockQCBonusRepo {
	return &mockQCBonusRepo{
		config: &QCBonusConfig{
			CurrentRatePerPcs: 1500.0,
			Currency:          "IDR",
			UpdatedAt:         time.Now(),
			UpdatedBy:         "Owner",
			RateHistory:       []QCBonusRateHistory{},
		},
		recipients: []QCBonusRecipient{
			{ID: "r1", UserID: "u1", FullName: "Bambang Supriyadi", Role: "Kepala Produksi", IsActive: true, ShareRatio: 1.0 / 3.0},
			{ID: "r2", UserID: "u2", FullName: "Cecep Hidayat", Role: "Mandor Line 1", IsActive: true, ShareRatio: 1.0 / 3.0},
			{ID: "r3", UserID: "u3", FullName: "Dadang Kurnia", Role: "Mandor Line 2", IsActive: true, ShareRatio: 1.0 / 3.0},
		},
		periods:      make(map[string]*QCBonusPeriodDetail),
		passedPcsMap: make(map[string]int),
	}
}

func (m *mockQCBonusRepo) GetBonusConfig(ctx context.Context) (*QCBonusConfig, error) {
	return m.config, nil
}

func (m *mockQCBonusRepo) UpdateBonusRate(ctx context.Context, rate float64, ownerID string, notes *string) (*QCBonusConfig, error) {
	m.config.CurrentRatePerPcs = rate
	m.config.UpdatedAt = time.Now()
	m.config.RateHistory = append(m.config.RateHistory, QCBonusRateHistory{
		RatePerPcs:    rate,
		EffectiveFrom: time.Now(),
		ChangedBy:     "Owner",
		Notes:         notes,
	})
	return m.config, nil
}

func (m *mockQCBonusRepo) ListRecipients(ctx context.Context) ([]QCBonusRecipient, error) {
	return m.recipients, nil
}

func (m *mockQCBonusRepo) UpdateRecipients(ctx context.Context, userIDs []string) ([]QCBonusRecipient, error) {
	var newRecs []QCBonusRecipient
	n := len(userIDs)
	for i, uid := range userIDs {
		newRecs = append(newRecs, QCBonusRecipient{
			ID:         string(rune('r' + i)),
			UserID:     uid,
			FullName:   "User " + uid,
			Role:       "Mandor",
			IsActive:   true,
			ShareRatio: 1.0 / float64(n),
		})
	}
	m.recipients = newRecs
	return m.recipients, nil
}

func (m *mockQCBonusRepo) ListPeriods(ctx context.Context, status string) ([]QCBonusPeriodSummary, error) {
	var list []QCBonusPeriodSummary
	for _, p := range m.periods {
		if status == "" || p.Status == status {
			list = append(list, QCBonusPeriodSummary{
				ID:                 p.ID,
				Name:               p.Name,
				StartDate:          p.StartDate,
				EndDate:            p.EndDate,
				Status:             p.Status,
				TotalPassedPcs:     p.TotalPassedPcs,
				RatePerPcs:         p.RatePerPcsApplied,
				TotalPoolAmount:    p.TotalPoolAmount,
				RecipientCount:     p.RecipientCount,
				AmountPerRecipient: p.AmountPerRecipient,
				CreatedAt:          p.CreatedAt,
			})
		}
	}
	return list, nil
}

func (m *mockQCBonusRepo) CreatePeriod(ctx context.Context, req CreateQCBonusPeriodRequest) (*QCBonusPeriodDetail, error) {
	p := &QCBonusPeriodDetail{
		ID:            "period-mock-1",
		Name:          req.Name,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		Status:        "DRAFT",
		Notes:         req.Notes,
		Distributions: []QCBonusRecipientShare{},
		CreatedAt:     time.Now(),
	}
	m.periods[p.ID] = p
	return p, nil
}

func (m *mockQCBonusRepo) GetPeriodByID(ctx context.Context, id string) (*QCBonusPeriodDetail, error) {
	p, ok := m.periods[id]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	return p, nil
}

func (m *mockQCBonusRepo) CalculatePeriod(ctx context.Context, id string) (*QCBonusPeriodDetail, error) {
	p, ok := m.periods[id]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if p.Status == "LOCKED" {
		return nil, ErrPeriodAlreadyLocked
	}

	totalPassedPcs := m.passedPcsMap[id]
	rate := m.config.CurrentRatePerPcs
	totalPool := float64(totalPassedPcs) * rate

	activeCount := len(m.recipients)
	if activeCount == 0 {
		return nil, ErrNoActiveRecipients
	}

	totalCents := int64(math.Round(totalPool * 100))
	baseCents := totalCents / int64(activeCount)
	remainderCents := totalCents % int64(activeCount)

	p.Distributions = []QCBonusRecipientShare{}
	for i, r := range m.recipients {
		cents := baseCents
		if int64(i) < remainderCents {
			cents++
		}
		share := float64(cents) / 100.0
		p.Distributions = append(p.Distributions, QCBonusRecipientShare{
			UserID:       r.UserID,
			FullName:     r.FullName,
			Role:         r.Role,
			ShareAmount:  share,
			PayoutStatus: "PENDING_DISBURSEMENT",
		})
	}

	p.Status = "CALCULATED"
	p.RatePerPcsApplied = rate
	p.TotalPassedPcs = totalPassedPcs
	p.TotalPoolAmount = totalPool
	p.RecipientCount = activeCount
	p.AmountPerRecipient = totalPool / float64(activeCount)
	return p, nil
}

func (m *mockQCBonusRepo) LockPeriod(ctx context.Context, id string, userID string, notes *string) (*QCBonusPeriodDetail, error) {
	p, ok := m.periods[id]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if p.Status == "LOCKED" {
		return nil, ErrPeriodAlreadyLocked
	}
	p.Status = "LOCKED"
	if notes != nil {
		p.Notes = notes
	}
	return p, nil
}

func TestQCBonusService_UpdateRate_OwnerOnly(t *testing.T) {
	repo := newMockQCBonusRepo()
	svc := NewService(repo)
	ctx := context.Background()

	owner := &authn.UserIdentity{ID: "owner-1", RoleCode: "owner_executive", IsActive: true}
	supervisor := &authn.UserIdentity{ID: "sup-1", RoleCode: "production_head", IsActive: true}

	// 1. Non-owner gets ErrForbiddenOwnerOnly
	_, err := svc.UpdateBonusRate(ctx, supervisor, UpdateQCBonusRateRequest{RatePerPcs: 2000.0})
	if err != ErrForbiddenOwnerOnly {
		t.Fatalf("expected ErrForbiddenOwnerOnly for non-owner, got %v", err)
	}

	// 2. Owner can update rate
	cfg, err := svc.UpdateBonusRate(ctx, owner, UpdateQCBonusRateRequest{RatePerPcs: 1750.0})
	if err != nil {
		t.Fatalf("unexpected error updating rate: %v", err)
	}
	if cfg.CurrentRatePerPcs != 1750.0 {
		t.Fatalf("expected rate 1750.0, got %.2f", cfg.CurrentRatePerPcs)
	}

	// 3. Negative rate rejected
	_, err = svc.UpdateBonusRate(ctx, owner, UpdateQCBonusRateRequest{RatePerPcs: -100.0})
	if err != ErrInvalidBonusRate {
		t.Fatalf("expected ErrInvalidBonusRate, got %v", err)
	}
}

func TestQCBonusService_SkenarioUji3_3000PcsDiv3(t *testing.T) {
	repo := newMockQCBonusRepo()
	svc := NewService(repo)
	ctx := context.Background()

	owner := &authn.UserIdentity{ID: "owner-1", RoleCode: "owner_executive", IsActive: true}

	// 1. Create period
	period, err := svc.CreatePeriod(ctx, owner, CreateQCBonusPeriodRequest{
		Name:      "Periode Uji Skenario 3",
		StartDate: "2026-10-09",
		EndDate:   "2026-10-15",
	})
	if err != nil {
		t.Fatalf("unexpected error creating period: %v", err)
	}

	// 2. Setup mock 3.000 pcs lolos QC
	repo.passedPcsMap[period.ID] = 3000

	// 3. Calculate pool bonus
	calcResult, err := svc.CalculatePeriod(ctx, owner, period.ID)
	if err != nil {
		t.Fatalf("unexpected error calculating period: %v", err)
	}

	// Verifikasi Skenario Uji 3:
	// - 3.000 pcs x Rp 1.500 = Rp 4.500.000
	// - 3 penerima aktif (2 Mandor + 1 Kepala Produksi)
	// - Tiap penerima memperoleh tepat Rp 1.500.000 tanpa selisih sen
	if calcResult.TotalPassedPcs != 3000 {
		t.Fatalf("expected 3000 passed pcs, got %d", calcResult.TotalPassedPcs)
	}
	if calcResult.TotalPoolAmount != 4500000.0 {
		t.Fatalf("expected total pool 4,500,000, got %.2f", calcResult.TotalPoolAmount)
	}
	if calcResult.RecipientCount != 3 {
		t.Fatalf("expected 3 recipients, got %d", calcResult.RecipientCount)
	}
	if calcResult.AmountPerRecipient != 1500000.0 {
		t.Fatalf("expected amount per recipient 1,500,000, got %.2f", calcResult.AmountPerRecipient)
	}

	var totalDistributed float64
	for _, dist := range calcResult.Distributions {
		if dist.ShareAmount != 1500000.0 {
			t.Fatalf("expected each recipient to receive exactly 1,500,000.00, but %s got %.2f", dist.FullName, dist.ShareAmount)
		}
		totalDistributed += dist.ShareAmount
	}
	if totalDistributed != calcResult.TotalPoolAmount {
		t.Fatalf("penny allocation mismatch: totalDistributed=%.2f, totalPool=%.2f", totalDistributed, calcResult.TotalPoolAmount)
	}

	// 4. Lock period
	locked, err := svc.LockPeriod(ctx, owner, period.ID, LockQCBonusPeriodRequest{})
	if err != nil {
		t.Fatalf("unexpected error locking period: %v", err)
	}
	if locked.Status != "LOCKED" {
		t.Fatalf("expected status LOCKED, got %s", locked.Status)
	}

	// 5. Attempt recalculate locked period -> ErrPeriodAlreadyLocked
	_, err = svc.CalculatePeriod(ctx, owner, period.ID)
	if err != ErrPeriodAlreadyLocked {
		t.Fatalf("expected ErrPeriodAlreadyLocked on locked period, got %v", err)
	}
}

