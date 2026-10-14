package quality

import (
	"errors"
	"time"
)

var (
	ErrDefectCategoryNotFound   = errors.New("kategori defect tidak ditemukan")
	ErrDefectCategoryCodeExists = errors.New("kode kategori defect sudah digunakan")
	ErrInspectionNotFound       = errors.New("data inspeksi QC tidak ditemukan")
	ErrAssemblyNotFound         = errors.New("data assembly tidak ditemukan")
	ErrOrderNotFound            = errors.New("data order tidak ditemukan")
	ErrInvalidQuantity          = errors.New("kuantitas inspeksi tidak valid")
	ErrDefectDetailsRequired    = errors.New("rincian defect wajib diisi jika terdapat repair atau reject")
	ErrDefectQuantityMismatch   = errors.New("total kuantitas pada rincian defect tidak sesuai dengan jumlah repair dan reject")
	ErrAssemblyAlreadyInspected = errors.New("assembly sudah diinspeksi sebelumnya")
	ErrForbiddenQuality         = errors.New("forbidden: pengguna tidak memiliki wewenang untuk aksi QC ini")
)

// DefectCategory represents a master defect catalog item (BPMN 06).
type DefectCategory struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Severity    string    `json:"severity"` // MINOR, MAJOR, CRITICAL
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateDefectCategoryRequest defines payload for creating a new defect category.
type CreateDefectCategoryRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Severity    string  `json:"severity" binding:"required,oneof=MINOR MAJOR CRITICAL"`
	Description *string `json:"description"`
}

// QCQueueItem represents an assembly item waiting for QC inspection (BPMN 06).
type QCQueueItem struct {
	AssemblyID        string    `json:"assemblyId"`
	OrderID           string    `json:"orderId"`
	OrderCode         string    `json:"orderCode"`
	ClientName        string    `json:"clientName"`
	ArticleName       string    `json:"articleName"`
	LineID            *string   `json:"lineId,omitempty"`
	LineCode          *string   `json:"lineCode,omitempty"`
	CompletedQuantity int       `json:"completedQuantity"`
	Status            string    `json:"status"`
	CompletedAt       time.Time `json:"completedAt"`
}

// ListQCQueueFilter defines filter for QC queue.
type ListQCQueueFilter struct {
	OrderID string `form:"orderId"`
	LineID  string `form:"lineId"`
	Limit   int    `form:"limit"`
	Offset  int    `form:"offset"`
}

// CreateQCInspectionRequest defines payload for recording a QC inspection.
type CreateQCInspectionRequest struct {
	AssemblyID     string                     `json:"assemblyId" binding:"required"`
	OrderID        string                     `json:"orderId" binding:"required"`
	InspectionDate string                     `json:"inspectionDate"`
	PassedQuantity int                        `json:"passedQuantity" binding:"gte=0"`
	RepairQuantity int                        `json:"repairQuantity" binding:"gte=0"`
	RejectQuantity int                        `json:"rejectQuantity" binding:"gte=0"`
	Notes          *string                    `json:"notes"`
	Defects        []CreateQCInspectionDefect `json:"defects"`
}

// CreateQCInspectionDefect defines defect item in inspection payload.
type CreateQCInspectionDefect struct {
	DefectCategoryID string  `json:"defectCategoryId" binding:"required"`
	ComponentID      *string `json:"componentId"`
	Quantity         int     `json:"quantity" binding:"required,gt=0"`
	Disposition      string  `json:"disposition" binding:"required,oneof=REPAIR REJECT"`
	Notes            *string `json:"notes"`
}

// QCInspectionSummary represents summary item in inspection list.
type QCInspectionSummary struct {
	ID               string    `json:"id"`
	InspectionNumber string    `json:"inspectionNumber"`
	OrderCode        string    `json:"orderCode"`
	ClientName       string    `json:"clientName"`
	InspectorName    string    `json:"inspectorName"`
	InspectionDate   string    `json:"inspectionDate"`
	PassedQuantity   int       `json:"passedQuantity"`
	RepairQuantity   int       `json:"repairQuantity"`
	RejectQuantity   int       `json:"rejectQuantity"`
	TotalChecked     int       `json:"totalChecked"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ListQCInspectionsFilter defines filter options for listing inspections.
type ListQCInspectionsFilter struct {
	OrderID   string `form:"orderId"`
	Status    string `form:"status"` // LOLOS, REPAIR, REJECT
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}

// QCInspectionDetail represents detailed inspection with defects breakdown.
type QCInspectionDetail struct {
	ID               string               `json:"id"`
	InspectionNumber string               `json:"inspectionNumber"`
	AssemblyID       string               `json:"assemblyId"`
	OrderID          string               `json:"orderId"`
	OrderCode        string               `json:"orderCode"`
	ClientName       string               `json:"clientName"`
	ArticleName      string               `json:"articleName"`
	InspectorID      string               `json:"inspectorId"`
	InspectorName    string               `json:"inspectorName"`
	InspectionDate   string               `json:"inspectionDate"`
	PassedQuantity   int                  `json:"passedQuantity"`
	RepairQuantity   int                  `json:"repairQuantity"`
	RejectQuantity   int                  `json:"rejectQuantity"`
	TotalChecked     int                  `json:"totalChecked"`
	Notes            *string              `json:"notes,omitempty"`
	CreatedAt        time.Time            `json:"createdAt"`
	Defects          []QCInspectionDefect `json:"defects"`
}

// QCInspectionDefect represents an individual defect item detected during inspection.
type QCInspectionDefect struct {
	ID                 string  `json:"id"`
	DefectCategoryCode string  `json:"defectCategoryCode"`
	DefectCategoryName string  `json:"defectCategoryName"`
	Severity           string  `json:"severity"`
	ComponentID        *string `json:"componentId,omitempty"`
	ComponentName      *string `json:"componentName,omitempty"`
	Quantity           int     `json:"quantity"`
	Disposition        string  `json:"disposition"` // REPAIR, REJECT
	Notes              *string `json:"notes,omitempty"`
}

// QCOrderSummary represents quality metrics for an order.
type QCOrderSummary struct {
	OrderID            string          `json:"orderId"`
	OrderCode          string          `json:"orderCode"`
	TargetQuantity     int             `json:"targetQuantity"`
	TotalChecked       int             `json:"totalChecked"`
	TotalPassed        int             `json:"totalPassed"`
	TotalRepair        int             `json:"totalRepair"`
	TotalReject        int             `json:"totalReject"`
	PassRatePercentage float64         `json:"passRatePercentage"`
	IsOrderCompleted   bool            `json:"isOrderCompleted"`
	TopDefects         []TopDefectItem `json:"topDefects"`
}

// TopDefectItem represents ranked defect item.
type TopDefectItem struct {
	CategoryName string `json:"categoryName"`
	Count        int    `json:"count"`
}

