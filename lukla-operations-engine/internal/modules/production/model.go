package production

import (
	"errors"
	"time"
)

var (
	ErrLineNotFound             = errors.New("line produksi tidak ditemukan")
	ErrLineCodeExists           = errors.New("kode line produksi sudah digunakan")
	ErrLineCapacityExceeded     = errors.New("kapasitas mesin line telah tercapai")
	ErrFactoryCapacityExceeded  = errors.New("total kapasitas pabrik melebihi batas 70 mesin")
	ErrAllocationNotFound       = errors.New("alokasi penjahit tidak ditemukan")
	ErrOperatorAlreadyAllocated = errors.New("operator sedang aktif bertugas pada alokasi line lain")
	ErrOperatorNotFound         = errors.New("operator tidak ditemukan atau tidak aktif")
	ErrOrderNotFound            = errors.New("order tidak ditemukan")
	ErrDestinationLineNotFound  = errors.New("line tujuan perpindahan tidak ditemukan")
	ErrSameLineTransfer         = errors.New("tidak dapat memindahkan operator ke line yang sama")
	ErrAllocationNotActive      = errors.New("hanya alokasi berstatus ACTIVE yang dapat dipindahkan")
	ErrTransferReasonRequired   = errors.New("transferReason wajib diisi untuk audit log")
	ErrForbiddenProduction      = errors.New("forbidden: hanya owner dan kepala produksi yang berwenang")
	ErrOutputNotFound           = errors.New("catatan output tidak ditemukan")
	ErrInvalidQuantity          = errors.New("jumlah output harus lebih dari 0")
	ErrTimeSlotRequired         = errors.New("timeSlot wajib diisi")
	ErrWorkDateRequired         = errors.New("workDate wajib diisi")
	ErrEmptyBatchItems          = errors.New("items batch output tidak boleh kosong")
)

// ProductionLine represents a sewing assembly line on the factory floor.
type ProductionLine struct {
	ID                   string     `json:"id"`
	Code                 string     `json:"code"`
	Name                 string     `json:"name"`
	MaxCapacity          int        `json:"maxCapacity"`
	ActiveOperatorsCount int        `json:"activeOperatorsCount"`
	SupervisorID         *string    `json:"supervisorId,omitempty"`
	SupervisorName       *string    `json:"supervisorName,omitempty"`
	Status               string     `json:"status"` // ACTIVE, INACTIVE, MAINTENANCE
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	DeletedAt            *time.Time `json:"deletedAt,omitempty"`
}

// ProductionLineDetail includes current allocations and active order references.
type ProductionLineDetail struct {
	ProductionLine
	CurrentAllocations []LineAllocation  `json:"currentAllocations"`
	ActiveOrders       []LineActiveOrder `json:"activeOrders"`
}

// LineActiveOrder represents active order metadata running on a line.
type LineActiveOrder struct {
	OrderID      string `json:"orderId"`
	OrderCode    string `json:"orderCode"`
	ClientName   string `json:"clientName"`
	ArticleTitle string `json:"articleTitle"`
}

// CreateLineRequest defines input payload for creating a new production line.
type CreateLineRequest struct {
	Code         string  `json:"code" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	MaxCapacity  int     `json:"maxCapacity" binding:"required"`
	SupervisorID *string `json:"supervisorId"`
}

// UpdateLineRequest defines input payload for modifying a production line.
type UpdateLineRequest struct {
	Name         *string `json:"name"`
	MaxCapacity  *int    `json:"maxCapacity"`
	SupervisorID *string `json:"supervisorId"`
	Status       *string `json:"status"`
}

// LineAllocation represents operator assignment to a line, order, and component.
type LineAllocation struct {
	ID                    string     `json:"id"`
	LineID                string     `json:"lineId"`
	LineCode              string     `json:"lineCode"`
	OrderID               string     `json:"orderId"`
	OrderCode             string     `json:"orderCode"`
	OperatorID            string     `json:"operatorId"`
	OperatorCode          string     `json:"operatorCode"`
	OperatorName          string     `json:"operatorName"`
	SkillTier             string     `json:"skillTier"`
	AssignedComponentID   *string    `json:"assignedComponentId,omitempty"`
	AssignedComponentName *string    `json:"assignedComponentName,omitempty"`
	Status                string     `json:"status"` // ACTIVE, COMPLETED, TRANSFERRED
	AssignedAt            time.Time  `json:"assignedAt"`
	CompletedAt           *time.Time `json:"completedAt,omitempty"`
}

// ListAllocationsFilter specifies query criteria for listing allocations.
type ListAllocationsFilter struct {
	LineID     string `form:"lineId"`
	OrderID    string `form:"orderId"`
	OperatorID string `form:"operatorId"`
	Status     string `form:"status"`
}

// CreateAllocationRequest defines input payload for assigning an operator.
type CreateAllocationRequest struct {
	LineID              string  `json:"lineId" binding:"required"`
	OrderID             string  `json:"orderId" binding:"required"`
	OperatorID          string  `json:"operatorId" binding:"required"`
	AssignedComponentID *string `json:"assignedComponentId"`
	AssignedAt          *string `json:"assignedAt"`
}

// TransferOperatorRequest defines payload for transferring an operator between lines (BPMN 04).
type TransferOperatorRequest struct {
	DestinationLineID   string  `json:"destinationLineId" binding:"required"`
	TransferReason      string  `json:"transferReason" binding:"required"`
	AssignedComponentID *string `json:"assignedComponentId"`
	TransferredAt       *string `json:"transferredAt"`
}

// TransferResult captures the outcome of an operator line transfer.
type TransferResult struct {
	OldAllocationID     string    `json:"oldAllocationId"`
	NewAllocationID     string    `json:"newAllocationId"`
	OperatorID          string    `json:"operatorId"`
	PreviousLineCode    string    `json:"previousLineCode"`
	DestinationLineCode string    `json:"destinationLineCode"`
	TransferredAt       time.Time `json:"transferredAt"`
}

// AllocationTransferLog represents an audit log entry for operator movements.
type AllocationTransferLog struct {
	ID                  string    `json:"id"`
	AllocationID        string    `json:"allocationId"`
	OperatorID          string    `json:"operatorId"`
	FromLineID          string    `json:"fromLineId"`
	FromLineCode        string    `json:"fromLineCode"`
	ToLineID            string    `json:"toLineId"`
	ToLineCode          string    `json:"toLineCode"`
	Reason              string    `json:"reason"`
	TransferredByUserID *string   `json:"transferredByUserId,omitempty"`
	TransferredByName   string    `json:"transferredByName,omitempty"`
	TransferredAt       time.Time `json:"transferredAt"`
}

// ProductionOutput represents a recorded sewing output for an operator within a 30-minute interval.
type ProductionOutput struct {
	ID                 string    `json:"id"`
	LineID             string    `json:"lineId"`
	LineCode           string    `json:"lineCode"`
	OrderID            string    `json:"orderId"`
	OperatorID         string    `json:"operatorId"`
	OperatorName       string    `json:"operatorName"`
	ComponentID        string    `json:"componentId"`
	ComponentName      string    `json:"componentName"`
	Quantity           int       `json:"quantity"`
	TimeSlot           string    `json:"timeSlot"`
	WorkDate           string    `json:"workDate"`
	SnapshotSewingRate float64   `json:"snapshotSewingRate"`
	Notes              *string   `json:"notes,omitempty"`
	RecordedByUserID   *string   `json:"recordedByUserId,omitempty"`
	RecordedByUserName string    `json:"recordedByUserName"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// ListOutputsFilter defines search and filter options for production outputs.
type ListOutputsFilter struct {
	LineID      string `form:"line_id"`
	OrderID     string `form:"order_id"`
	OperatorID  string `form:"operator_id"`
	ComponentID string `form:"component_id"`
	Date        string `form:"date"`
	TimeSlot    string `form:"time_slot"`
}

// CreateOutputRequest defines payload for single 30-minute output submission by Mandor.
type CreateOutputRequest struct {
	LineID      string  `json:"lineId" binding:"required"`
	OrderID     string  `json:"orderId" binding:"required"`
	OperatorID  string  `json:"operatorId" binding:"required"`
	ComponentID string  `json:"componentId" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required"`
	TimeSlot    string  `json:"timeSlot" binding:"required"`
	WorkDate    string  `json:"workDate" binding:"required"`
	Notes       *string `json:"notes"`
}

// BatchCreateOutputRequest defines payload for multi-operator output submission at Mandor terminal.
type BatchCreateOutputRequest struct {
	LineID   string            `json:"lineId" binding:"required"`
	OrderID  string            `json:"orderId" binding:"required"`
	TimeSlot string            `json:"timeSlot" binding:"required"`
	WorkDate string            `json:"workDate" binding:"required"`
	Items    []BatchOutputItem `json:"items" binding:"required"`
}

// BatchOutputItem defines an individual operator's piece output within a batch submission.
type BatchOutputItem struct {
	OperatorID  string  `json:"operatorId" binding:"required"`
	ComponentID string  `json:"componentId" binding:"required"`
	Quantity    int     `json:"quantity" binding:"required"`
	Notes       *string `json:"notes"`
}


