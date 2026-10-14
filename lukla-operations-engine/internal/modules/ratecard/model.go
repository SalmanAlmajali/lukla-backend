package ratecard

import (
	"errors"
	"time"
)

var (
	ErrRateCardNotFound        = errors.New("rate card tidak ditemukan")
	ErrForbiddenRateCardManage = errors.New("forbidden: only owner executive and production head can manage rate cards")
	ErrForbiddenRateCardAccess = errors.New("forbidden: client cannot access other client's rate cards")
	ErrInvalidRateValue        = errors.New("tarif jahit dan potong tidak boleh negatif")
	ErrChangeReasonRequired    = errors.New("changeReason is required for rate card audit trail")
	ErrDuplicateRateCard       = errors.New("rate card for this article component already exists")
	ErrClientNotFound          = errors.New("client tidak ditemukan")
	ErrArticleNotFound         = errors.New("artikel tidak ditemukan")
	ErrComponentMismatch       = errors.New("komponen tidak ditemukan atau tidak cocok dengan artikel/klien")
)

// RateCard represents the standard sewing and cutting piece rates for an article component.
type RateCard struct {
	ID            string     `json:"id"`
	ClientID      string     `json:"clientId"`
	ClientName    string     `json:"clientName,omitempty"`
	ArticleID     string     `json:"articleId"`
	ArticleTitle  string     `json:"articleTitle,omitempty"`
	ComponentID   string     `json:"componentId"`
	ComponentName string     `json:"componentName,omitempty"`
	SewingRate    float64    `json:"sewingRate"`
	CuttingRate   float64    `json:"cuttingRate"`
	EffectiveDate string     `json:"effectiveDate"` // Format: YYYY-MM-DD
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
}

// RateCardHistory represents an audit trail log for rate card adjustments.
type RateCardHistory struct {
	ID              string    `json:"id"`
	RateCardID      string    `json:"rateCardId"`
	OldSewingRate   float64   `json:"oldSewingRate"`
	NewSewingRate   float64   `json:"newSewingRate"`
	OldCuttingRate  float64   `json:"oldCuttingRate"`
	NewCuttingRate  float64   `json:"newCuttingRate"`
	ChangedByUserID *string   `json:"changedByUserId,omitempty"`
	ChangedByName   string    `json:"changedByName,omitempty"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"createdAt"`
}

// ListRateCardsFilter defines query parameters for filtering rate cards.
type ListRateCardsFilter struct {
	ClientID  string `form:"clientId"`
	ArticleID string `form:"articleId"`
}

// CreateRateCardRequest defines the payload for creating a new rate card.
type CreateRateCardRequest struct {
	ClientID      string   `json:"clientId" binding:"required"`
	ArticleID     string   `json:"articleId" binding:"required"`
	ComponentID   string   `json:"componentId" binding:"required"`
	SewingRate    float64  `json:"sewingRate"`
	CuttingRate   float64  `json:"cuttingRate"`
	EffectiveDate *string  `json:"effectiveDate"`
	Notes         *string  `json:"notes"`
}

// UpdateRateCardRequest defines the payload for updating rate card rates.
type UpdateRateCardRequest struct {
	SewingRate    float64  `json:"sewingRate"`
	CuttingRate   *float64 `json:"cuttingRate"`
	EffectiveDate *string  `json:"effectiveDate"`
	ChangeReason  string   `json:"changeReason" binding:"required"`
}

// ComponentValidationResult contains hierarchy metadata retrieved during rate card creation.
type ComponentValidationResult struct {
	ClientName    string
	ArticleTitle  string
	ComponentName string
}

