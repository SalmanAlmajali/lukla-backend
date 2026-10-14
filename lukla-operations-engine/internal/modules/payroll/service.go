package payroll

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) isAuthorized(roleCode string) bool {
	return roleCode == "owner_executive" || roleCode == "finance_payroll" || roleCode == "finance"
}

func (s *Service) CreatePeriod(ctx context.Context, callerRole string, req CreatePayrollPeriodRequest) (*PayrollPeriod, error) {
	if !s.isAuthorized(callerRole) {
		return nil, ErrForbiddenPayroll
	}

	if req.MealAllowancePercentage < 10.0 || req.MealAllowancePercentage > 15.0 {
		return nil, ErrInvalidMealAllowance
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate format (YYYY-MM-DD): %w", err)
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate format (YYYY-MM-DD): %w", err)
	}

	if start.After(end) {
		return nil, ErrInvalidPeriodDates
	}

	periodCode := strings.TrimSpace(req.PeriodCode)
	if periodCode == "" {
		year, week := start.ISOWeek()
		periodCode = fmt.Sprintf("PAY-%04d-W%02d", year, week)
	}

	var notes *string
	if n := strings.TrimSpace(req.Notes); n != "" {
		notes = &n
	}

	period := &PayrollPeriod{
		PeriodCode:              periodCode,
		StartDate:               req.StartDate,
		EndDate:                 req.EndDate,
		MealAllowancePercentage: req.MealAllowancePercentage,
		Status:                  string(PeriodStatusDraft),
		Notes:                   notes,
	}

	if err := s.repo.CreatePeriod(ctx, period); err != nil {
		return nil, err
	}

	return period, nil
}

func (s *Service) GetPeriod(ctx context.Context, id string) (*PayrollPeriod, error) {
	return s.repo.FindPeriodByID(ctx, id)
}

func (s *Service) ListPeriods(ctx context.Context, filter ListPeriodsFilter) ([]PayrollPeriod, int, error) {
	return s.repo.ListPeriods(ctx, filter)
}

func (s *Service) CalculatePayroll(ctx context.Context, callerRole, periodID string, req CalculatePayrollRequest) (*PayrollPeriod, error) {
	if !s.isAuthorized(callerRole) {
		return nil, ErrForbiddenPayroll
	}

	period, err := s.repo.FindPeriodByID(ctx, periodID)
	if err != nil {
		return nil, err
	}

	if period.Status == string(PeriodStatusLocked) {
		return nil, ErrPeriodAlreadyLocked
	}

	aggregations, err := s.repo.GetEligibleOperatorOutputs(ctx, period.StartDate, period.EndDate, req.OperatorIDs)
	if err != nil {
		return nil, fmt.Errorf("retrieve eligible outputs: %w", err)
	}

	// Group outputs per operator
	operatorMap := make(map[string]*PayrollSlip)
	itemsMap := make(map[string][]PayrollSlipItem)
	operatorOrder := []string{}

	for _, agg := range aggregations {
		slip, exists := operatorMap[agg.OperatorID]
		if !exists {
			slip = &PayrollSlip{
				PayrollPeriodID:  periodID,
				OperatorID:       agg.OperatorID,
				OperatorCode:     agg.OperatorCode,
				OperatorName:     agg.OperatorName,
				OperatorJoinDate: agg.OperatorJoinDate,
				Status:           string(SlipStatusCalculated),
			}
			operatorMap[agg.OperatorID] = slip
			operatorOrder = append(operatorOrder, agg.OperatorID)
		}

		slip.TotalPieces += agg.Quantity
		slip.GrossPieceRateTotal += agg.Subtotal

		item := PayrollSlipItem{
			OrderID:            agg.OrderID,
			OrderCode:          agg.OrderCode,
			ClientName:         agg.ClientName,
			ComponentID:        agg.ComponentID,
			ComponentName:      agg.ComponentName,
			WorkDate:           agg.WorkDate,
			Quantity:           agg.Quantity,
			SnapshotSewingRate: agg.SnapshotSewingRate,
			Subtotal:           agg.Subtotal,
		}
		itemsMap[agg.OperatorID] = append(itemsMap[agg.OperatorID], item)
	}

	var totalGross, totalMeal, totalNet float64
	slips := make([]PayrollSlip, 0, len(operatorOrder))

	for _, opID := range operatorOrder {
		slip := operatorMap[opID]

		// Round currency to 2 decimal places
		slip.GrossPieceRateTotal = math.Round(slip.GrossPieceRateTotal*100) / 100

		// Uang makan = % x borongan penjahit itu
		mealAllowance := slip.GrossPieceRateTotal * (period.MealAllowancePercentage / 100.0)
		slip.MealAllowanceAmount = math.Round(mealAllowance*100) / 100

		slip.NetPayableTotal = math.Round((slip.GrossPieceRateTotal+slip.MealAllowanceAmount+slip.AdjustmentsTotal)*100) / 100

		totalGross += slip.GrossPieceRateTotal
		totalMeal += slip.MealAllowanceAmount
		totalNet += slip.NetPayableTotal

		slips = append(slips, *slip)
	}

	totalGross = math.Round(totalGross*100) / 100
	totalMeal = math.Round(totalMeal*100) / 100
	totalNet = math.Round(totalNet*100) / 100

	err = s.repo.SaveCalculationResults(
		ctx,
		periodID,
		slips,
		itemsMap,
		len(slips),
		totalGross,
		totalMeal,
		0.00,
		totalNet,
	)
	if err != nil {
		return nil, err
	}

	return s.repo.FindPeriodByID(ctx, periodID)
}

func (s *Service) ListSlips(ctx context.Context, periodID string, filter ListSlipsFilter) ([]PayrollSlip, int, error) {
	// Verify period exists
	if _, err := s.repo.FindPeriodByID(ctx, periodID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListSlipsByPeriod(ctx, periodID, filter)
}

func (s *Service) GetSlipDetail(ctx context.Context, slipID string) (*PayrollSlipDetail, error) {
	return s.repo.FindSlipByID(ctx, slipID)
}
