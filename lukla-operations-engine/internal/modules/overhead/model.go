package overhead

import (
	"errors"
	"time"
)

var (
	ErrCategoryNotFound     = errors.New("kategori overhead tidak ditemukan")
	ErrCategoryCodeExists   = errors.New("kode kategori overhead sudah terdaftar")
	ErrPeriodNotFound       = errors.New("periode overhead tidak ditemukan")
	ErrPeriodAlreadyExists   = errors.New("periode overhead untuk bulan dan tahun tersebut sudah terdaftar")
	ErrPeriodAlreadyLocked   = errors.New("periode overhead sudah dikunci (LOCKED) dan tidak dapat diubah lagi")
	ErrExpenseNotFound       = errors.New("catatan pengeluaran overhead tidak ditemukan")
	ErrInvalidMonth         = errors.New("bulan periode harus di antara 1 dan 12")
	ErrInvalidYear          = errors.New("tahun periode tidak valid")
	ErrInvalidAmount        = errors.New("nominal biaya harus lebih dari 0")
	ErrInvalidWeights       = errors.New("bobot rasio produk harus lebih dari 0")
	ErrOrderNotFound        = errors.New("order tidak ditemukan")
)

const (
	StatusOpen       = "OPEN"
	StatusCalculated = "CALCULATED"
	StatusLocked     = "LOCKED"
)

// OverheadCategory merepresentasikan master kategori biaya overhead pabrik.
type OverheadCategory struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateOverheadCategoryRequest payload saat mendaftarkan kategori biaya overhead baru.
type CreateOverheadCategoryRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

// OverheadPeriodSummary ringkasan satu periode overhead bulanan.
type OverheadPeriodSummary struct {
	ID                 string    `json:"id"`
	PeriodCode         string    `json:"periodCode"`
	PeriodMonth        int       `json:"periodMonth"`
	PeriodYear         int       `json:"periodYear"`
	StartDate          string    `json:"startDate"`
	EndDate            string    `json:"endDate"`
	Status             string    `json:"status"`
	TotalExpenseAmount float64   `json:"totalExpenseAmount"`
	TotalProducedUnits int       `json:"totalProducedUnits"`
	FlatRatePerUnit    float64   `json:"flatRatePerUnit"`
	GarmentRatePerUnit float64   `json:"garmentRatePerUnit"`
	BagRatePerUnit     float64   `json:"bagRatePerUnit"`
	Currency           string    `json:"currency"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// OverheadExpenseItem catatan pengeluaran biaya aktual pada suatu periode overhead.
type OverheadExpenseItem struct {
	ID               string    `json:"id"`
	OverheadPeriodID string    `json:"overheadPeriodId"`
	CategoryID       string    `json:"categoryId"`
	CategoryName     string    `json:"categoryName"`
	CategoryCode     string    `json:"categoryCode"`
	ExpenseDate      string    `json:"expenseDate"`
	Amount           float64   `json:"amount"`
	InvoiceReference *string   `json:"invoiceReference,omitempty"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// OverheadPeriodDetail detail lengkap periode overhead beserta seluruh pengeluaran dan status penguncian.
type OverheadPeriodDetail struct {
	ID                 string                `json:"id"`
	PeriodCode         string                `json:"periodCode"`
	PeriodMonth        int                   `json:"periodMonth"`
	PeriodYear         int                   `json:"periodYear"`
	StartDate          string                `json:"startDate"`
	EndDate            string                `json:"endDate"`
	Status             string                `json:"status"`
	TotalExpenseAmount float64               `json:"totalExpenseAmount"`
	TotalProducedUnits int                   `json:"totalProducedUnits"`
	GarmentUnits       int                   `json:"garmentUnits"`
	BagUnits           int                   `json:"bagUnits"`
	WeightGarment      float64               `json:"weightGarment"`
	WeightBag          float64               `json:"weightBag"`
	FlatRatePerUnit    float64               `json:"flatRatePerUnit"`
	GarmentRatePerUnit float64               `json:"garmentRatePerUnit"`
	BagRatePerUnit     float64               `json:"bagRatePerUnit"`
	Currency           string                `json:"currency"`
	Notes              *string               `json:"notes,omitempty"`
	CalculatedAt       *time.Time            `json:"calculatedAt,omitempty"`
	LockedAt           *time.Time            `json:"lockedAt,omitempty"`
	LockedBy           *string               `json:"lockedBy,omitempty"`
	LockedByName       *string               `json:"lockedByName,omitempty"`
	CreatedAt          time.Time             `json:"createdAt"`
	UpdatedAt          time.Time             `json:"updatedAt"`
	Expenses           []OverheadExpenseItem `json:"expenses"`
}

// CreateOverheadPeriodRequest payload untuk membuka periode overhead bulanan baru.
type CreateOverheadPeriodRequest struct {
	PeriodMonth int     `json:"periodMonth" binding:"required,min=1,max=12"`
	PeriodYear  int     `json:"periodYear" binding:"required,min=2024,max=2050"`
	Notes       *string `json:"notes"`
}

// RecordOverheadExpenseRequest payload untuk mencatat pengeluaran biaya aktual.
type RecordOverheadExpenseRequest struct {
	CategoryID       string  `json:"categoryId" binding:"required"`
	ExpenseDate      string  `json:"expenseDate" binding:"required"`
	Amount           float64 `json:"amount" binding:"required,gt=0"`
	InvoiceReference *string `json:"invoiceReference"`
	Notes            *string `json:"notes"`
}

// CalculateOverheadRequest payload opsi kalkulasi pembobotan overhead per unit.
type CalculateOverheadRequest struct {
	WeightGarment *float64 `json:"weightGarment"` // Default 1.00
	WeightBag     *float64 `json:"weightBag"`     // Default 1.25
	Notes         *string  `json:"notes"`
}

// OverheadCalculationResult hasil kalkulasi tarif overhead per unit.
type OverheadCalculationResult struct {
	PeriodID           string    `json:"periodId"`
	PeriodCode         string    `json:"periodCode"`
	Status             string    `json:"status"`
	TotalExpenseAmount float64   `json:"totalExpenseAmount"`
	TotalProducedUnits int       `json:"totalProducedUnits"`
	GarmentUnits       int       `json:"garmentUnits"`
	BagUnits           int       `json:"bagUnits"`
	FlatRatePerUnit    float64   `json:"flatRatePerUnit"`
	GarmentRatePerUnit float64   `json:"garmentRatePerUnit"`
	BagRatePerUnit     float64   `json:"bagRatePerUnit"`
	Currency           string    `json:"currency"`
	CalculatedAt       time.Time `json:"calculatedAt"`
}

// ExpenseBreakdownItem rincian persentase pengeluaran per kategori.
type ExpenseBreakdownItem struct {
	CategoryCode string  `json:"categoryCode"`
	CategoryName string  `json:"categoryName"`
	TotalAmount  float64 `json:"totalAmount"`
	Percentage   float64 `json:"percentage"`
}

// ProductionBreakdown rekapitulasi volume produksi dalam periode.
type ProductionBreakdown struct {
	TotalUnits   int `json:"totalUnits"`
	GarmentUnits int `json:"garmentUnits"`
	BagUnits     int `json:"bagUnits"`
}

// RatesBreakdown rekapitulasi tarif terhitung.
type RatesBreakdown struct {
	FlatRatePerUnit    float64 `json:"flatRatePerUnit"`
	GarmentRatePerUnit float64 `json:"garmentRatePerUnit"`
	BagRatePerUnit     float64 `json:"bagRatePerUnit"`
}

// OverheadPeriodSummaryReport laporan eksekutif alokasi overhead bulanan.
type OverheadPeriodSummaryReport struct {
	PeriodCode          string                 `json:"periodCode"`
	PeriodMonth         int                    `json:"periodMonth"`
	PeriodYear          int                    `json:"periodYear"`
	Status              string                 `json:"status"`
	TotalExpenseAmount  float64                `json:"totalExpenseAmount"`
	ExpenseBreakdown    []ExpenseBreakdownItem `json:"expenseBreakdown"`
	ProductionBreakdown ProductionBreakdown    `json:"productionBreakdown"`
	Rates               RatesBreakdown         `json:"rates"`
}

// OrderCOGSReport laporan HPP / COGS komprehensif per order (BPMN 01).
type OrderCOGSReport struct {
	OrderID              string         `json:"orderId"`
	OrderNumber          string         `json:"orderNumber"`
	ClientName           string         `json:"clientName"`
	ArticleCode          string         `json:"articleCode"`
	ArticleTitle         string         `json:"articleTitle"`
	Category             string         `json:"category"`
	ProducedQuantity     int            `json:"producedQuantity"`
	ContractPricePerUnit float64        `json:"contractPricePerUnit"`
	TotalRevenue         float64        `json:"totalRevenue"`
	CostComponents       COGSComponents `json:"costComponents"`
	TotalCostPerUnit     float64        `json:"totalCostPerUnit"`
	TotalCOGS            float64        `json:"totalCOGS"`
	GrossMarginPerUnit   float64        `json:"grossMarginPerUnit"`
	TotalGrossMargin     float64        `json:"totalGrossMargin"`
	GrossMarginPct       float64        `json:"grossMarginPercentage"`
}

// COGSComponents rincian komponen biaya pembentuk HPP order.
type COGSComponents struct {
	DirectLaborPerUnit float64 `json:"directLaborPerUnit"`
	TotalDirectLabor   float64 `json:"totalDirectLabor"`
	QCBonusPerUnit     float64 `json:"qcBonusPerUnit"`
	TotalQCBonus       float64 `json:"totalQCBonus"`
	OverheadPerUnit    float64 `json:"overheadPerUnit"`
	TotalOverhead      float64 `json:"totalOverhead"`
}

