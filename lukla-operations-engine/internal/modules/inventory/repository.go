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
}

