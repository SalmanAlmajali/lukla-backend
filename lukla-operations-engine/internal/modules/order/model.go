package order

import (
	"errors"
	"time"
)

var (
	ErrOrderNotFound         = errors.New("pesanan tidak ditemukan")
	ErrOrderAlreadyCompleted = errors.New("pesanan sudah berstatus COMPLETED sebelumnya")
	ErrInvalidOrderStatus    = errors.New("status pesanan tidak valid untuk diselesaikan")
)

// InternalOrderSummary merepresentasikan ikhtisar pesanan untuk pengawasan lini produksi pabrik.
type InternalOrderSummary struct {
	ID             string     `json:"id"`
	OrderNumber    string     `json:"orderNumber"`
	ClientID       string     `json:"clientId"`
	ClientName     string     `json:"clientName"`
	ArticleID      string     `json:"articleId"`
	ArticleCode    string     `json:"articleCode"`
	ArticleTitle   string     `json:"articleTitle"`
	TargetQuantity int        `json:"targetQuantity"`
	PassedQuantity int        `json:"passedQuantity"`
	Status         string     `json:"status"`
	StartDate      *string    `json:"startDate,omitempty"`
	TargetDate     *string    `json:"targetDate,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// InternalOrderDetail merepresentasikan detail teknis pesanan internal pabrik.
type InternalOrderDetail struct {
	InternalOrderSummary
	Notes            *string               `json:"notes,omitempty"`
	CuttingQuantity  int                   `json:"cuttingQuantity"`
	SewingQuantity   int                   `json:"sewingQuantity"`
	RepairQuantity   int                   `json:"repairQuantity"`
	RejectQuantity   int                   `json:"rejectQuantity"`
	PassRate         float64               `json:"passRate"`
	RecentMilestones []OrderMilestoneEvent `json:"recentMilestones,omitempty"`
}

// OrderMilestoneEvent merepresentasikan rekam jejak status order.
type OrderMilestoneEvent struct {
	Status    string    `json:"status"`
	Notes     string    `json:"notes,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// CompleteOrderRequest payload opsional saat staf menandai order selesai.
type CompleteOrderRequest struct {
	Notes *string `json:"notes"`
}

