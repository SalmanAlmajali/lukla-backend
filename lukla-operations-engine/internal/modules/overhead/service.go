package overhead

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// Service antarmuka logika bisnis untuk modul overhead pabrik.
type Service struct {
	repo Repository
}

// NewService membuat instance Service baru.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListCategories mengambil seluruh master kategori biaya overhead.
func (s *Service) ListCategories(ctx context.Context) ([]OverheadCategory, error) {
	return s.repo.ListCategories(ctx)
}

// CreateCategory mendaftarkan kategori biaya overhead baru.
func (s *Service) CreateCategory(ctx context.Context, req CreateOverheadCategoryRequest) (*OverheadCategory, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return nil, fmt.Errorf("kode dan nama kategori overhead wajib diisi")
	}

	cat := &OverheadCategory{
		Code:        code,
		Name:        name,
		Description: req.Description,
	}

	return s.repo.CreateCategory(ctx, cat)
}

// ListPeriods mengambil daftar riwayat periode pembukuan overhead bulanan.
func (s *Service) ListPeriods(ctx context.Context, year *int, status *string) ([]OverheadPeriodSummary, error) {
	return s.repo.ListPeriods(ctx, year, status)
}

// CreatePeriod membuka periode pembukuan overhead bulanan baru.
func (s *Service) CreatePeriod(ctx context.Context, req CreateOverheadPeriodRequest) (*OverheadPeriodDetail, error) {
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return nil, ErrInvalidMonth
	}
	if req.PeriodYear < 2024 || req.PeriodYear > 2050 {
		return nil, ErrInvalidYear
	}

	period := &OverheadPeriodDetail{
		PeriodMonth: req.PeriodMonth,
		PeriodYear:  req.PeriodYear,
		Notes:       req.Notes,
	}

	return s.repo.CreatePeriod(ctx, period)
}

// GetPeriodByID mengambil detail lengkap periode overhead.
func (s *Service) GetPeriodByID(ctx context.Context, id string) (*OverheadPeriodDetail, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrPeriodNotFound
	}
	return s.repo.GetPeriodByID(ctx, id)
}

// RecordExpense mencatat pengeluaran biaya aktual pada suatu periode.
func (s *Service) RecordExpense(ctx context.Context, periodID string, req RecordOverheadExpenseRequest) (*OverheadExpenseItem, error) {
	if strings.TrimSpace(periodID) == "" {
		return nil, ErrPeriodNotFound
	}
	if strings.TrimSpace(req.CategoryID) == "" {
		return nil, ErrCategoryNotFound
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	expenseDate := strings.TrimSpace(req.ExpenseDate)
	if expenseDate == "" {
		expenseDate = time.Now().Format("2006-01-02")
	}

	item := &OverheadExpenseItem{
		OverheadPeriodID: periodID,
		CategoryID:       req.CategoryID,
		ExpenseDate:      expenseDate,
		Amount:           req.Amount,
		InvoiceReference: req.InvoiceReference,
		Notes:            req.Notes,
	}

	return s.repo.RecordExpense(ctx, item)
}

// DeleteExpense menghapus catatan pengeluaran biaya overhead.
func (s *Service) DeleteExpense(ctx context.Context, periodID, expenseID string) error {
	if strings.TrimSpace(periodID) == "" || strings.TrimSpace(expenseID) == "" {
		return ErrExpenseNotFound
	}
	return s.repo.DeleteExpense(ctx, periodID, expenseID)
}

// CalculateOverhead menjalankan agregasi unit produksi dan kalkulasi tarif alokasi overhead per unit.
func (s *Service) CalculateOverhead(ctx context.Context, periodID string, req CalculateOverheadRequest) (*OverheadCalculationResult, error) {
	period, err := s.repo.GetPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}
	if period.Status == StatusLocked {
		return nil, ErrPeriodAlreadyLocked
	}

	// 1. Tentukan bobot alokasi produk pakaian (GARMENT) vs tas (BAG)
	weightGarment := 1.00
	if req.WeightGarment != nil && *req.WeightGarment > 0 {
		weightGarment = *req.WeightGarment
	}
	weightBag := 1.25
	if req.WeightBag != nil && *req.WeightBag > 0 {
		weightBag = *req.WeightBag
	}

	// 2. Agregasi kuantitas unit barang jadi yang lolos QC pada rentang tanggal periode
	totalUnits, garmentUnits, bagUnits, err := s.repo.GetProductionVolumeByDateRange(ctx, period.StartDate, period.EndDate)
	if err != nil {
		return nil, fmt.Errorf("aggregate production volume: %w", err)
	}

	totalExpense := period.TotalExpenseAmount

	// 3. Hitung tarif alokasi flat dan terbobot
	var flatRate, garmentRate, bagRate float64

	if totalUnits > 0 && totalExpense > 0 {
		// Tarif Rata Sederhana (Flat Rate)
		flatRate = math.Round((totalExpense/float64(totalUnits))*100) / 100

		// Tarif Terbobot Pakaian vs Tas
		weightedUnits := (float64(garmentUnits) * weightGarment) + (float64(bagUnits) * weightBag)
		if weightedUnits > 0 {
			baseRate := totalExpense / weightedUnits
			garmentRate = math.Round(baseRate*weightGarment*100) / 100
			bagRate = math.Round(baseRate*weightBag*100) / 100
		} else {
			garmentRate = flatRate
			bagRate = flatRate
		}
	} else if totalUnits == 0 && totalExpense > 0 {
		// Jika belum ada unit riil tercatat, gunakan estimasi unit default 5000 pcs (garment 4000, bag 1000)
		estUnits := 5000
		estGarment := 4000
		estBag := 1000
		totalUnits = estUnits
		garmentUnits = estGarment
		bagUnits = estBag

		flatRate = math.Round((totalExpense/float64(estUnits))*100) / 100
		weightedUnits := (float64(estGarment) * weightGarment) + (float64(estBag) * weightBag)
		baseRate := totalExpense / weightedUnits
		garmentRate = math.Round(baseRate*weightGarment*100) / 100
		bagRate = math.Round(baseRate*weightBag*100) / 100
	}

	calcResult := &OverheadCalculationResult{
		PeriodID:           periodID,
		PeriodCode:         period.PeriodCode,
		Status:             StatusCalculated,
		TotalExpenseAmount: totalExpense,
		TotalProducedUnits: totalUnits,
		GarmentUnits:       garmentUnits,
		BagUnits:           bagUnits,
		FlatRatePerUnit:    flatRate,
		GarmentRatePerUnit: garmentRate,
		BagRatePerUnit:     bagRate,
		Currency:           period.Currency,
		CalculatedAt:       time.Now(),
	}

	if err := s.repo.UpdateCalculation(ctx, periodID, calcResult, weightGarment, weightBag, req.Notes); err != nil {
		return nil, fmt.Errorf("save calculation: %w", err)
	}

	return calcResult, nil
}

// LockPeriod mengunci periode overhead bulanan secara permanen.
func (s *Service) LockPeriod(ctx context.Context, periodID string, lockedBy string) (*OverheadPeriodDetail, error) {
	if strings.TrimSpace(periodID) == "" {
		return nil, ErrPeriodNotFound
	}
	return s.repo.LockPeriod(ctx, periodID, lockedBy)
}

// GetPeriodSummaryReport menyajikan laporan ringkasan alokasi overhead bulanan.
func (s *Service) GetPeriodSummaryReport(ctx context.Context, periodID string) (*OverheadPeriodSummaryReport, error) {
	if strings.TrimSpace(periodID) == "" {
		return nil, ErrPeriodNotFound
	}
	return s.repo.GetPeriodSummaryReport(ctx, periodID)
}

// GetOrderCOGSReport menyajikan laporan HPP / COGS komprehensif per order (BPMN 01).
func (s *Service) GetOrderCOGSReport(ctx context.Context, orderID string) (*OrderCOGSReport, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, ErrOrderNotFound
	}
	return s.repo.GetOrderCOGSReport(ctx, orderID)
}

