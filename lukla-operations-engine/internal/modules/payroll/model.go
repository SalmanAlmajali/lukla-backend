package payroll

import (
	"errors"
	"time"
)

var (
	ErrPeriodNotFound               = errors.New("periode payroll tidak ditemukan")
	ErrPeriodAlreadyLocked          = errors.New("periode payroll sudah dikunci (locked) dan tidak dapat dimodifikasi")
	ErrPeriodCodeExists             = errors.New("kode periode payroll sudah digunakan")
	ErrInvalidPeriodDates           = errors.New("tanggal mulai periode harus sebelum atau sama dengan tanggal akhir")
	ErrInvalidMealAllowance         = errors.New("persentase uang makan harus berada di antara 10.00% dan 15.00%")
	ErrSlipNotFound                 = errors.New("slip gaji operator tidak ditemukan")
	ErrForbiddenPayroll             = errors.New("forbidden: hanya owner dan finance yang berwenang")
	ErrPeriodDraftCannotBeLocked    = errors.New("periode berstatus DRAFT belum dapat dikunci sebelum dikalkulasi")
	ErrInvalidAdjustmentType        = errors.New("tipe penyesuaian harus BONUS, DEDUCTION, CORRECTION, atau REVERSAL")
	ErrAdjustmentReasonRequired     = errors.New("alasan penyesuaian (reason) wajib diisi untuk audit trail")
	ErrAdjustmentZeroAmount         = errors.New("nominal penyesuaian (amount) tidak boleh bernilai 0")
	ErrPeriodMustBeCalculatedToLock = errors.New("periode harus berstatus CALCULATED sebelum dapat dikunci")
)

type PeriodStatus string

const (
	PeriodStatusDraft      PeriodStatus = "DRAFT"
	PeriodStatusCalculated PeriodStatus = "CALCULATED"
	PeriodStatusLocked     PeriodStatus = "LOCKED"
)

type SlipStatus string

const (
	SlipStatusCalculated SlipStatus = "CALCULATED"
	SlipStatusLocked     SlipStatus = "LOCKED"
	SlipStatusPaid       SlipStatus = "PAID"
)

// PayrollPeriod represents a weekly cycle of tailor compensation and meal allowance.
type PayrollPeriod struct {
	ID                      string     `json:"id"`
	PeriodCode              string     `json:"periodCode"`
	StartDate               string     `json:"startDate"`
	EndDate                 string     `json:"endDate"`
	MealAllowancePercentage float64    `json:"mealAllowancePercentage"`
	Status                  string     `json:"status"` // DRAFT, CALCULATED, LOCKED
	TotalOperatorsCount     int        `json:"totalOperatorsCount"`
	TotalGrossPieceRate     float64    `json:"totalGrossPieceRate"`
	TotalMealAllowance      float64    `json:"totalMealAllowance"`
	TotalAdjustments        float64    `json:"totalAdjustments"`
	TotalNetPayable         float64    `json:"totalNetPayable"`
	LockedAt                *time.Time `json:"lockedAt,omitempty"`
	LockedByUserID          *string    `json:"lockedByUserId,omitempty"`
	Notes                   *string    `json:"notes,omitempty"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

type CreatePayrollPeriodRequest struct {
	PeriodCode              string  `json:"periodCode"`
	StartDate               string  `json:"startDate" binding:"required"`
	EndDate                 string  `json:"endDate" binding:"required"`
	MealAllowancePercentage float64 `json:"mealAllowancePercentage" binding:"required"`
	Notes                   string  `json:"notes"`
}

type CalculatePayrollRequest struct {
	OperatorIDs []string `json:"operatorIds"`
	Recalculate bool     `json:"recalculate"`
}

type PayrollSlip struct {
	ID                  string    `json:"id"`
	PayrollPeriodID     string    `json:"payrollPeriodId"`
	PeriodCode          string    `json:"periodCode,omitempty"`
	OperatorID          string    `json:"operatorId"`
	OperatorCode        string    `json:"operatorCode"`
	OperatorName        string    `json:"operatorName"`
	OperatorJoinDate    string    `json:"operatorJoinDate"`
	TotalPieces         int       `json:"totalPieces"`
	GrossPieceRateTotal float64   `json:"grossPieceRateTotal"`
	MealAllowanceAmount float64   `json:"mealAllowanceAmount"`
	AdjustmentsTotal    float64   `json:"adjustmentsTotal"`
	NetPayableTotal     float64   `json:"netPayableTotal"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type PayrollSlipDetail struct {
	PayrollSlip
	MealAllowancePercentage float64           `json:"mealAllowancePercentage"`
	Items                   []PayrollSlipItem `json:"items"`
}

type PayrollSlipItem struct {
	ID                 string    `json:"id"`
	PayrollSlipID      string    `json:"payrollSlipId"`
	OrderID            string    `json:"orderId"`
	OrderCode          string    `json:"orderCode"`
	ClientName         string    `json:"clientName"`
	ComponentID        string    `json:"componentId"`
	ComponentName      string    `json:"componentName"`
	WorkDate           string    `json:"workDate"`
	Quantity           int       `json:"quantity"`
	SnapshotSewingRate float64   `json:"snapshotSewingRate"`
	Subtotal           float64   `json:"subtotal"`
	CreatedAt          time.Time `json:"createdAt"`
}

type ListPeriodsFilter struct {
	Status string
	Limit  int
	Offset int
}

type ListSlipsFilter struct {
	OperatorID string
	Search     string
	Limit      int
	Offset     int
}

// OperatorOutputAggregation represents aggregated work done by an operator on a component.
type OperatorOutputAggregation struct {
	OperatorID         string
	OperatorCode       string
	OperatorName       string
	OperatorJoinDate   string
	OrderID            string
	OrderCode          string
	ClientName         string
	ComponentID        string
	ComponentName      string
	WorkDate           string
	Quantity           int
	SnapshotSewingRate float64
	Subtotal           float64
}

type AdjustmentType string

const (
	AdjustmentTypeBonus      AdjustmentType = "BONUS"
	AdjustmentTypeDeduction  AdjustmentType = "DEDUCTION"
	AdjustmentTypeCorrection AdjustmentType = "CORRECTION"
	AdjustmentTypeReversal   AdjustmentType = "REVERSAL"
)

type PayrollAdjustment struct {
	ID              string    `json:"id"`
	PayrollPeriodID string    `json:"payrollPeriodId"`
	OperatorID      string    `json:"operatorId"`
	OperatorName    string    `json:"operatorName"`
	AdjustmentType  string    `json:"adjustmentType"`
	Amount          float64   `json:"amount"`
	Reason          string    `json:"reason"`
	CreatedByUserID string    `json:"createdByUserId"`
	CreatedByName   string    `json:"createdByName"`
	CreatedAt       time.Time `json:"createdAt"`
}

type CreateAdjustmentRequest struct {
	OperatorID     string  `json:"operatorId" binding:"required"`
	AdjustmentType string  `json:"adjustmentType" binding:"required"`
	Amount         float64 `json:"amount" binding:"required"`
	Reason         string  `json:"reason" binding:"required"`
}

type LockPayrollPeriodRequest struct {
	Notes string `json:"notes"`
}

type LinePayrollSummary struct {
	LineID         string  `json:"lineId"`
	LineCode       string  `json:"lineCode"`
	LineName       string  `json:"lineName"`
	OperatorsCount int     `json:"operatorsCount"`
	PiecesProduced int     `json:"piecesProduced"`
	GrossAmount    float64 `json:"grossAmount"`
}

type PayrollSummaryReport struct {
	PeriodID                string               `json:"periodId"`
	PeriodCode              string               `json:"periodCode"`
	StartDate               string               `json:"startDate"`
	EndDate                 string               `json:"endDate"`
	MealAllowancePercentage float64              `json:"mealAllowancePercentage"`
	Status                  string               `json:"status"`
	TotalOperators          int                  `json:"totalOperators"`
	TotalPiecesProduced     int                  `json:"totalPiecesProduced"`
	TotalGrossPieceRate     float64              `json:"totalGrossPieceRate"`
	TotalMealAllowance      float64              `json:"totalMealAllowance"`
	TotalAdjustments        float64              `json:"totalAdjustments"`
	GrandTotalPayable       float64              `json:"grandTotalPayable"`
	LinesSummary            []LinePayrollSummary `json:"linesSummary"`
	OperatorsSummary        []PayrollSlip        `json:"operatorsSummary,omitempty"`
}


