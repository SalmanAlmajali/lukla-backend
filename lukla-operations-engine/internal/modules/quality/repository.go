package quality

import (
	"context"
)

// Repository defines storage operations for quality control.
type Repository interface {
	// Defect Categories
	ListDefectCategories(ctx context.Context, isActive *bool) ([]DefectCategory, error)
	CreateDefectCategory(ctx context.Context, cat *DefectCategory) error
	FindDefectCategoryByID(ctx context.Context, id string) (*DefectCategory, error)

	// QC Queue
	ListQCQueue(ctx context.Context, filter ListQCQueueFilter) ([]QCQueueItem, int, error)

	// Inspections
	CreateInspection(ctx context.Context, req *CreateQCInspectionRequest, inspectorID string) (*QCInspectionDetail, error)
	ListInspections(ctx context.Context, filter ListQCInspectionsFilter) ([]QCInspectionSummary, int, error)
	FindInspectionByID(ctx context.Context, id string) (*QCInspectionDetail, error)

	// Analytics / Summary
	GetOrderQualitySummary(ctx context.Context, orderID string) (*QCOrderSummary, error)
}

