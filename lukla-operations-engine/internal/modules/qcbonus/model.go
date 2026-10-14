package qcbonus

import (
	"errors"
	"time"
)

var (
	ErrForbiddenOwnerOnly    = errors.New("forbidden: hanya owner yang berwenang mengatur tarif dan penerima bonus QC")
	ErrForbiddenQualityBonus = errors.New("forbidden: akses ditolak untuk operasi bonus QC ini")
	ErrConfigNotFound        = errors.New("konfigurasi tarif bonus QC tidak ditemukan")
	ErrPeriodNotFound        = errors.New("periode bonus QC tidak ditemukan")
	ErrPeriodAlreadyLocked   = errors.New("periode bonus QC sudah dikunci (LOCKED) dan tidak dapat dihitung ulang")
	ErrPeriodDatesInvalid    = errors.New("rentang tanggal periode tidak valid: endDate harus sama atau setelah startDate")
	ErrNoActiveRecipients    = errors.New("tidak ada penerima bonus QC yang aktif")
	ErrInvalidBonusRate      = errors.New("tarif bonus per pcs harus lebih dari atau sama dengan 0")
)

// QCBonusConfig represents configuration and history of QC bonus rate.
type QCBonusConfig struct {
	CurrentRatePerPcs float64              `json:"currentRatePerPcs"`
	Currency          string               `json:"currency"`
	UpdatedAt         time.Time            `json:"updatedAt"`
	UpdatedBy         string               `json:"updatedBy"`
	RateHistory       []QCBonusRateHistory `json:"rateHistory"`
}

// QCBonusRateHistory represents an audit log entry for rate modifications.
type QCBonusRateHistory struct {
	RatePerPcs    float64   `json:"ratePerPcs"`
	EffectiveFrom time.Time `json:"effectiveFrom"`
	ChangedBy     string    `json:"changedBy"`
	Notes         *string   `json:"notes,omitempty"`
}

// UpdateQCBonusRateRequest defines payload for owner updating bonus rate.
type UpdateQCBonusRateRequest struct {
	RatePerPcs float64 `json:"ratePerPcs" binding:"required,gte=0"`
	Notes      *string `json:"notes"`
}

// QCBonusRecipient represents a recipient user with share ratio.
type QCBonusRecipient struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	FullName   string  `json:"fullName"`
	Role       string  `json:"role"`
	Title      string  `json:"title"`
	IsActive   bool    `json:"isActive"`
	ShareRatio float64 `json:"shareRatio"`
}

// UpdateQCBonusRecipientsRequest defines payload for owner updating recipient list.
type UpdateQCBonusRecipientsRequest struct {
	RecipientUserIDs []string `json:"recipientUserIds" binding:"required,min=1"`
}

// CreateQCBonusPeriodRequest defines payload for opening a new bonus period.
type CreateQCBonusPeriodRequest struct {
	Name      string  `json:"name" binding:"required"`
	StartDate string  `json:"startDate" binding:"required"`
	EndDate   string  `json:"endDate" binding:"required"`
	Notes     *string `json:"notes"`
}

// QCBonusPeriodSummary represents a high-level summary of a bonus period.
type QCBonusPeriodSummary struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	StartDate          string    `json:"startDate"`
	EndDate            string    `json:"endDate"`
	Status             string    `json:"status"` // DRAFT, CALCULATED, LOCKED
	TotalPassedPcs     int       `json:"totalPassedPcs"`
	RatePerPcs         float64   `json:"ratePerPcs"`
	TotalPoolAmount    float64   `json:"totalPoolAmount"`
	RecipientCount     int       `json:"recipientCount"`
	AmountPerRecipient float64   `json:"amountPerRecipient"`
	CreatedAt          time.Time `json:"createdAt"`
}

// QCBonusPeriodDetail represents detailed period calculation with recipient shares.
type QCBonusPeriodDetail struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	StartDate          string                  `json:"startDate"`
	EndDate            string                  `json:"endDate"`
	Status             string                  `json:"status"` // DRAFT, CALCULATED, LOCKED
	RatePerPcsApplied  float64                 `json:"ratePerPcsApplied"`
	TotalPassedPcs     int                     `json:"totalPassedPcs"`
	TotalPoolAmount    float64                 `json:"totalPoolAmount"`
	RecipientCount     int                     `json:"recipientCount"`
	AmountPerRecipient float64                 `json:"amountPerRecipient"`
	Notes              *string                 `json:"notes,omitempty"`
	Distributions      []QCBonusRecipientShare `json:"distributions"`
	CreatedAt          time.Time               `json:"createdAt"`
}

// QCBonusRecipientShare represents individual share distribution for a recipient.
type QCBonusRecipientShare struct {
	UserID       string  `json:"userId"`
	FullName     string  `json:"fullName"`
	Role         string  `json:"role"`
	ShareAmount  float64 `json:"shareAmount"`
	PayoutStatus string  `json:"payoutStatus"` // PENDING_DISBURSEMENT, PAID, CANCELLED
}

// LockQCBonusPeriodRequest defines payload when finalizing/locking a period.
type LockQCBonusPeriodRequest struct {
	Notes *string `json:"notes"`
}

