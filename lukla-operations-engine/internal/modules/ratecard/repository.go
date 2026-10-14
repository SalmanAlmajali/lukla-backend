package ratecard

import "context"

// Repository defines data access contracts for rate cards and their audit trail histories.
type Repository interface {
	ListRateCards(ctx context.Context, filter ListRateCardsFilter) ([]RateCard, error)
	FindRateCardByID(ctx context.Context, id string) (*RateCard, error)
	FindRateCardByComponent(ctx context.Context, articleID, componentID string) (*RateCard, error)
	CreateRateCard(ctx context.Context, rc *RateCard, initialHistory *RateCardHistory) error
	UpdateRateCard(ctx context.Context, rc *RateCard, history *RateCardHistory) error
	ListRateCardHistories(ctx context.Context, rateCardID string) ([]RateCardHistory, error)
	ValidateComponentHierarchy(ctx context.Context, clientID, articleID, componentID string) (*ComponentValidationResult, error)
}

