package inventory

import "context"

type Repository interface {
	// Materials Master
	CreateMaterial(ctx context.Context, m *Material) error
	FindMaterialByID(ctx context.Context, id string) (*Material, error)
	FindMaterialByCode(ctx context.Context, code string) (*Material, error)
	ListMaterials(ctx context.Context, filter ListMaterialsFilter) ([]Material, int, error)

	// Receipts & Inbound
	VerifyClientAndOrder(ctx context.Context, clientID, orderID string) error
	CreateReceipt(ctx context.Context, receipt *InventoryReceipt, items []CreateReceiptItem) (*ReceiptDetail, error)
	FindReceiptByID(ctx context.Context, id string) (*ReceiptDetail, error)
	ListReceipts(ctx context.Context, filter ListReceiptsFilter) ([]ReceiptSummary, int, error)

	// Lots & Stocks
	FindLotByID(ctx context.Context, id string) (*InventoryLot, error)
	ListStockBalances(ctx context.Context, filter ListStocksFilter) ([]StockBalance, error)

	// Issues & Outbound (BPMN 02 / Skenario Uji 1)
	CreateIssue(ctx context.Context, userID string, req CreateIssueRequest) (*IssueDetail, error)
	FindIssueByID(ctx context.Context, id string) (*IssueDetail, error)
	ListIssues(ctx context.Context, filter ListIssuesFilter) ([]IssueSummary, int, error)

	// Blocked Attempts Audit Trail
	RecordBlockedAttempt(ctx context.Context, record *BlockedAttemptRecord) error
	ListBlockedAttempts(ctx context.Context, filter ListBlockedAttemptsFilter) ([]BlockedAttemptRecord, int, error)

	// Cutting Scraps & Waste (FR-6.3)
	CreateScrap(ctx context.Context, userID string, req CreateScrapRequest) (*CuttingScrapRecord, error)
	ListScraps(ctx context.Context, filter ListScrapsFilter) ([]CuttingScrapRecord, int, error)
}

