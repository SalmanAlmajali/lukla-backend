package payroll

import "context"

// Repository defines the data persistence contract for the weekly payroll domain.
type Repository interface {
	CreatePeriod(ctx context.Context, period *PayrollPeriod) error
	FindPeriodByID(ctx context.Context, id string) (*PayrollPeriod, error)
	FindPeriodByCode(ctx context.Context, code string) (*PayrollPeriod, error)
	ListPeriods(ctx context.Context, filter ListPeriodsFilter) ([]PayrollPeriod, int, error)
	UpdatePeriodTotalsAndStatus(ctx context.Context, periodID string, status string, opsCount int, gross, meal, adj, net float64) error

	GetEligibleOperatorOutputs(ctx context.Context, startDate, endDate string, operatorIDs []string) ([]OperatorOutputAggregation, error)
	SaveCalculationResults(ctx context.Context, periodID string, slips []PayrollSlip, itemsMap map[string][]PayrollSlipItem, totalOps int, totalGross, totalMeal, totalAdj, totalNet float64) error

	ListSlipsByPeriod(ctx context.Context, periodID string, filter ListSlipsFilter) ([]PayrollSlip, int, error)
	FindSlipByID(ctx context.Context, slipID string) (*PayrollSlipDetail, error)

	LockPeriod(ctx context.Context, periodID string, lockedByUserID string, notes *string) (*PayrollPeriod, error)
	CreateAdjustment(ctx context.Context, periodID string, adj *PayrollAdjustment) error
	ListAdjustmentsByPeriod(ctx context.Context, periodID string) ([]PayrollAdjustment, error)
	GetPayrollSummaryReport(ctx context.Context, periodID string) (*PayrollSummaryReport, error)
}

