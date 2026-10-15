package overhead

import (
	"context"
)

// Repository mendefinisikan interface persistensi data untuk modul overhead pabrik.
type Repository interface {
	ListCategories(ctx context.Context) ([]OverheadCategory, error)
	CreateCategory(ctx context.Context, cat *OverheadCategory) (*OverheadCategory, error)
	GetCategoryByID(ctx context.Context, id string) (*OverheadCategory, error)

	ListPeriods(ctx context.Context, year *int, status *string) ([]OverheadPeriodSummary, error)
	CreatePeriod(ctx context.Context, period *OverheadPeriodDetail) (*OverheadPeriodDetail, error)
	GetPeriodByID(ctx context.Context, id string) (*OverheadPeriodDetail, error)

	RecordExpense(ctx context.Context, expense *OverheadExpenseItem) (*OverheadExpenseItem, error)
	DeleteExpense(ctx context.Context, periodID string, expenseID string) error

	GetProductionVolumeByDateRange(ctx context.Context, startDate, endDate string) (totalUnits, garmentUnits, bagUnits int, err error)
	UpdateCalculation(ctx context.Context, periodID string, result *OverheadCalculationResult, weightGarment, weightBag float64, notes *string) error
	LockPeriod(ctx context.Context, periodID string, lockedBy string) (*OverheadPeriodDetail, error)

	GetPeriodSummaryReport(ctx context.Context, periodID string) (*OverheadPeriodSummaryReport, error)
	GetOrderCOGSReport(ctx context.Context, orderID string) (*OrderCOGSReport, error)
}

