package inventory

import (
	"context"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) isWarehouseOrOwner(role string) bool {
	r := strings.ToLower(role)
	return r == "warehouse" || r == "owner_executive"
}

func (s *Service) isInternalStaff(role string) bool {
	r := strings.ToLower(role)
	return r == "warehouse" || r == "owner_executive" || r == "production_head" || r == "sample_adis" || r == "finance_payroll"
}

// CreateMaterial handles adding a new material specification.
func (s *Service) CreateMaterial(ctx context.Context, callerRole string, req CreateMaterialRequest) (*Material, error) {
	if !s.isWarehouseOrOwner(callerRole) {
		return nil, ErrForbiddenInventory
	}

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	cat := strings.ToUpper(strings.TrimSpace(req.Category))
	unit := strings.ToUpper(strings.TrimSpace(req.DefaultUnit))

	if code == "" || name == "" || cat == "" || unit == "" {
		return nil, ErrMissingRequiredField
	}

	m := &Material{
		Code:        code,
		Name:        name,
		Category:    cat,
		DefaultUnit: unit,
		Description: strings.TrimSpace(req.Description),
	}

	if err := s.repo.CreateMaterial(ctx, m); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *Service) GetMaterial(ctx context.Context, id string) (*Material, error) {
	return s.repo.FindMaterialByID(ctx, id)
}

func (s *Service) ListMaterials(ctx context.Context, filter ListMaterialsFilter) ([]Material, int, error) {
	return s.repo.ListMaterials(ctx, filter)
}

// CreateReceipt handles inbound material arrival with mandatory Client ID and Order ID tagging (FR-6.1).
func (s *Service) CreateReceipt(ctx context.Context, callerRole, callerUserID string, req CreateReceiptRequest) (*ReceiptDetail, error) {
	if !s.isWarehouseOrOwner(callerRole) {
		return nil, ErrForbiddenInventory
	}

	receiptNumber := strings.TrimSpace(req.ReceiptNumber)
	clientID := strings.TrimSpace(req.ClientID)
	orderID := strings.TrimSpace(req.OrderID)
	receivedDate := strings.TrimSpace(req.ReceivedDate)

	if receiptNumber == "" || clientID == "" || orderID == "" || receivedDate == "" {
		return nil, ErrMissingRequiredField
	}

	if len(req.Items) == 0 {
		return nil, ErrReceiptItemsEmpty
	}

	// Validate items
	for _, it := range req.Items {
		if strings.TrimSpace(it.MaterialID) == "" || strings.TrimSpace(it.LotNumber) == "" || strings.TrimSpace(it.WarehouseLocation) == "" {
			return nil, ErrMissingRequiredField
		}
		if it.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
	}

	var supplier *string
	if s := strings.TrimSpace(req.SupplierName); s != "" {
		supplier = &s
	}
	var deliveryNote *string
	if d := strings.TrimSpace(req.DeliveryNoteNumber); d != "" {
		deliveryNote = &d
	}
	var notes *string
	if n := strings.TrimSpace(req.Notes); n != "" {
		notes = &n
	}

	receipt := &InventoryReceipt{
		ReceiptNumber:      receiptNumber,
		ClientID:           clientID,
		OrderID:            orderID,
		ReceivedDate:       receivedDate,
		SupplierName:       supplier,
		DeliveryNoteNumber: deliveryNote,
		ReceivedByUserID:   callerUserID,
		Notes:              notes,
	}

	return s.repo.CreateReceipt(ctx, receipt, req.Items)
}

func (s *Service) GetReceipt(ctx context.Context, id string) (*ReceiptDetail, error) {
	return s.repo.FindReceiptByID(ctx, id)
}

func (s *Service) ListReceipts(ctx context.Context, filter ListReceiptsFilter) ([]ReceiptSummary, int, error) {
	return s.repo.ListReceipts(ctx, filter)
}

func (s *Service) GetLot(ctx context.Context, id string) (*InventoryLot, error) {
	return s.repo.FindLotByID(ctx, id)
}

func (s *Service) ListStocks(ctx context.Context, callerRole string, callerClientID *string, filter ListStocksFilter) ([]StockBalance, error) {
	r := strings.ToLower(callerRole)
	if r == "client" {
		if callerClientID == nil || *callerClientID == "" {
			return nil, ErrForbiddenInventory
		}
		// Strict multi-tenant isolation
		filter.ClientID = *callerClientID
	} else if !s.isInternalStaff(callerRole) {
		return nil, ErrForbiddenInventory
	}

	return s.repo.ListStockBalances(ctx, filter)
}

