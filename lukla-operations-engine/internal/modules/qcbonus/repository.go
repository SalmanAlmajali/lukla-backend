package qcbonus

import "context"

// Repository defines storage operations for QC pool bonus (BPMN 08).
type Repository interface {
	// Configuration (Owner only)
	GetBonusConfig(ctx context.Context) (*QCBonusConfig, error)
	UpdateBonusRate(ctx context.Context, rate float64, ownerID string, notes *string) (*QCBonusConfig, error)

	// Recipients (Owner only)
	ListRecipients(ctx context.Context) ([]QCBonusRecipient, error)
	UpdateRecipients(ctx context.Context, userIDs []string) ([]QCBonusRecipient, error)

	// Periods & Calculation (Skenario Uji 3)
	ListPeriods(ctx context.Context, status string) ([]QCBonusPeriodSummary, error)
	CreatePeriod(ctx context.Context, req CreateQCBonusPeriodRequest) (*QCBonusPeriodDetail, error)
	GetPeriodByID(ctx context.Context, id string) (*QCBonusPeriodDetail, error)
	CalculatePeriod(ctx context.Context, id string) (*QCBonusPeriodDetail, error)
	LockPeriod(ctx context.Context, id string, userID string, notes *string) (*QCBonusPeriodDetail, error)
}

