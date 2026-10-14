package production

import (
	"context"
)

// Repository specifies the contract for production persistence operations.
type Repository interface {
	// Line operations
	ListLines(ctx context.Context) ([]ProductionLine, error)
	FindLineByID(ctx context.Context, id string) (*ProductionLineDetail, error)
	FindLineByCode(ctx context.Context, code string) (*ProductionLine, error)
	CreateLine(ctx context.Context, line *ProductionLine) error
	UpdateLine(ctx context.Context, line *ProductionLine) error
	GetTotalFactoryCapacity(ctx context.Context, excludeLineID string) (int, error)

	// Allocation operations
	ListAllocations(ctx context.Context, filter ListAllocationsFilter) ([]LineAllocation, error)
	FindAllocationByID(ctx context.Context, id string) (*LineAllocation, error)
	FindActiveAllocationByOperator(ctx context.Context, operatorID string) (*LineAllocation, error)
	CreateAllocation(ctx context.Context, alloc *LineAllocation) error
	TransferOperator(ctx context.Context, oldAllocID string, newAlloc *LineAllocation, transferLog *AllocationTransferLog) error
	ListTransferHistoriesByAllocation(ctx context.Context, allocationID string) ([]AllocationTransferLog, error)

	// Validation helpers
	ValidateOperator(ctx context.Context, operatorID string) (name string, code string, skillTier string, err error)
	ValidateOrder(ctx context.Context, orderID string) (orderCode string, clientID string, articleID string, err error)
	ValidateComponent(ctx context.Context, componentID string) (componentName string, err error)

	// Output operations (BPMN 05)
	ListOutputs(ctx context.Context, filter ListOutputsFilter) ([]ProductionOutput, error)
	FindOutputByID(ctx context.Context, id string) (*ProductionOutput, error)
	CreateOutput(ctx context.Context, output *ProductionOutput) error
	CreateOutputsBatch(ctx context.Context, outputs []*ProductionOutput) error
	GetActiveSewingRate(ctx context.Context, articleID string, componentID string) (float64, error)

	// Assembly & QC Queue operations (BPMN 05)
	ListAssemblies(ctx context.Context, filter ListAssembliesFilter) ([]AssemblyRecord, error)
	FindAssemblyByID(ctx context.Context, id string) (*AssemblyRecord, error)
	CreateAssembly(ctx context.Context, assembly *AssemblyRecord, recordedByUserID *string) error
	GetTotalComponentOutputsForOrder(ctx context.Context, orderID string) (int, error)
	GetTotalAssembledForOrder(ctx context.Context, orderID string) (int, error)
	ListQCQueue(ctx context.Context, filter ListQCQueueFilter) ([]QCQueueItem, error)

	// Dashboard operations (BPMN 05)
	GetRealtimeMetrics(ctx context.Context) (*RealtimeProductionMetrics, error)
	GetLineDailyProgress(ctx context.Context, lineID string, date string) (*LineDailyProgress, error)
	GetOrderProductionProgress(ctx context.Context, orderID string) (*OrderProductionProgress, error)
}

