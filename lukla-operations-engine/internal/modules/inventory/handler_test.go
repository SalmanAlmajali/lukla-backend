package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type mockInventoryRepository struct {
	materials         map[string]*Material
	receipts          map[string]*ReceiptDetail
	lots              map[string]*InventoryLot
	issues            map[string]*IssueDetail
	blockedAttempts   []*BlockedAttemptRecord
	scraps            []*CuttingScrapRecord
	validClientOrders map[string]string // orderID -> clientID
	orderNumbers      map[string]string // orderID -> orderNumber
	clientNames       map[string]string // clientID -> clientName
}

func newMockRepository() *mockInventoryRepository {
	return &mockInventoryRepository{
		materials:         make(map[string]*Material),
		receipts:          make(map[string]*ReceiptDetail),
		lots:              make(map[string]*InventoryLot),
		issues:            make(map[string]*IssueDetail),
		blockedAttempts:   make([]*BlockedAttemptRecord, 0),
		scraps:            make([]*CuttingScrapRecord, 0),
		validClientOrders: make(map[string]string),
		orderNumbers:      make(map[string]string),
		clientNames:       make(map[string]string),
	}
}

func (m *mockInventoryRepository) VerifyClientAndOrder(ctx context.Context, clientID, orderID string) error {
	cID, ok := m.validClientOrders[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	if cID != clientID {
		return ErrClientOrderMismatch
	}
	return nil
}

func (m *mockInventoryRepository) CreateMaterial(ctx context.Context, mat *Material) error {
	for _, existing := range m.materials {
		if existing.Code == mat.Code {
			return ErrMaterialCodeExists
		}
	}
	mat.ID = "mat-" + mat.Code
	mat.CreatedAt = time.Now()
	mat.UpdatedAt = time.Now()
	m.materials[mat.ID] = mat
	return nil
}

func (m *mockInventoryRepository) FindMaterialByID(ctx context.Context, id string) (*Material, error) {
	if mat, ok := m.materials[id]; ok {
		return mat, nil
	}
	return nil, ErrMaterialNotFound
}

func (m *mockInventoryRepository) FindMaterialByCode(ctx context.Context, code string) (*Material, error) {
	for _, mat := range m.materials {
		if mat.Code == code {
			return mat, nil
		}
	}
	return nil, ErrMaterialNotFound
}

func (m *mockInventoryRepository) ListMaterials(ctx context.Context, filter ListMaterialsFilter) ([]Material, int, error) {
	var result []Material
	for _, mat := range m.materials {
		if filter.Category != "" && mat.Category != filter.Category {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(mat.Name), strings.ToLower(filter.Search)) {
			continue
		}
		result = append(result, *mat)
	}
	return result, len(result), nil
}

func (m *mockInventoryRepository) CreateReceipt(ctx context.Context, receipt *InventoryReceipt, items []CreateReceiptItem) (*ReceiptDetail, error) {
	if err := m.VerifyClientAndOrder(ctx, receipt.ClientID, receipt.OrderID); err != nil {
		return nil, err
	}

	receipt.ID = "rcv-001"
	receipt.CreatedAt = time.Now()
	receipt.UpdatedAt = time.Now()

	var payloadItems []ReceiptItemPayload
	for i, it := range items {
		lotID := fmtLotID(receipt.ID, i)
		lot := &InventoryLot{
			ID:                lotID,
			ReceiptID:         receipt.ID,
			LotNumber:         it.LotNumber,
			MaterialID:        it.MaterialID,
			MaterialName:      "Mock Material",
			ClientID:          receipt.ClientID,
			ClientName:        "Mock Client",
			OrderID:           receipt.OrderID,
			OrderCode:         "PO-MOCK",
			InitialQuantity:   it.Quantity,
			CurrentQuantity:   it.Quantity,
			Unit:              it.Unit,
			RollCount:         it.RollCount,
			Status:            string(LotStatusAvailable),
			WarehouseLocation: it.WarehouseLocation,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		m.lots[lotID] = lot

		payloadItems = append(payloadItems, ReceiptItemPayload{
			ID:                fmtItemID(receipt.ID, i),
			MaterialID:        it.MaterialID,
			MaterialCode:      "MAT-MOCK",
			MaterialName:      "Mock Material",
			LotID:             lotID,
			LotNumber:         it.LotNumber,
			Quantity:          it.Quantity,
			Unit:              it.Unit,
			RollCount:         it.RollCount,
			WarehouseLocation: it.WarehouseLocation,
		})
	}

	detail := &ReceiptDetail{
		ReceiptSummary: ReceiptSummary{
			ID:              receipt.ID,
			ReceiptNumber:   receipt.ReceiptNumber,
			ClientID:        receipt.ClientID,
			ClientName:      "Mock Client",
			OrderID:         receipt.OrderID,
			OrderCode:       "PO-MOCK",
			ReceivedDate:    receipt.ReceivedDate,
			TotalItemsCount: len(payloadItems),
			CreatedAt:       receipt.CreatedAt,
		},
		ReceivedByName: "Warehouse Officer",
		Items:          payloadItems,
	}
	m.receipts[receipt.ID] = detail
	return detail, nil
}

func fmtLotID(rcvID string, idx int) string {
	return rcvID + "-lot-" + string(rune('A'+idx))
}
func fmtItemID(rcvID string, idx int) string {
	return rcvID + "-item-" + string(rune('1'+idx))
}

func (m *mockInventoryRepository) FindReceiptByID(ctx context.Context, id string) (*ReceiptDetail, error) {
	if r, ok := m.receipts[id]; ok {
		return r, nil
	}
	return nil, ErrReceiptNotFound
}

func (m *mockInventoryRepository) ListReceipts(ctx context.Context, filter ListReceiptsFilter) ([]ReceiptSummary, int, error) {
	var list []ReceiptSummary
	for _, r := range m.receipts {
		if filter.ClientID != "" && r.ClientID != filter.ClientID {
			continue
		}
		if filter.OrderID != "" && r.OrderID != filter.OrderID {
			continue
		}
		list = append(list, r.ReceiptSummary)
	}
	return list, len(list), nil
}

func (m *mockInventoryRepository) FindLotByID(ctx context.Context, id string) (*InventoryLot, error) {
	if lot, ok := m.lots[id]; ok {
		return lot, nil
	}
	return nil, ErrLotNotFound
}

func (m *mockInventoryRepository) ListStockBalances(ctx context.Context, filter ListStocksFilter) ([]StockBalance, error) {
	balancesMap := make(map[string]*StockBalance)
	for _, lot := range m.lots {
		if filter.ClientID != "" && lot.ClientID != filter.ClientID {
			continue
		}
		if filter.MaterialID != "" && lot.MaterialID != filter.MaterialID {
			continue
		}
		if filter.OrderID != "" && lot.OrderID != filter.OrderID {
			continue
		}
		key := lot.MaterialID + "-" + lot.ClientID + "-" + lot.OrderID
		if b, ok := balancesMap[key]; ok {
			b.TotalQuantity += lot.CurrentQuantity
			b.AvailableLotsCount++
		} else {
			balancesMap[key] = &StockBalance{
				MaterialID:         lot.MaterialID,
				MaterialCode:       lot.MaterialCode,
				MaterialName:       lot.MaterialName,
				ClientID:           lot.ClientID,
				ClientName:         lot.ClientName,
				OrderID:            lot.OrderID,
				OrderCode:          lot.OrderCode,
				TotalQuantity:      lot.CurrentQuantity,
				Unit:               lot.Unit,
				AvailableLotsCount: 1,
			}
		}
	}
	var res []StockBalance
	for _, b := range balancesMap {
		res = append(res, *b)
	}
	return res, nil
}

func (m *mockInventoryRepository) CreateIssue(ctx context.Context, userID string, req CreateIssueRequest) (*IssueDetail, error) {
	targetClientID, ok := m.validClientOrders[req.TargetOrderID]
	if !ok {
		return nil, ErrOrderNotFound
	}

	targetOrderCode := m.orderNumbers[req.TargetOrderID]
	targetClientName := m.clientNames[targetClientID]

	var itemsPayload []IssueItemPayload
	for _, it := range req.Items {
		lot, ok := m.lots[it.LotID]
		if !ok {
			return nil, ErrLotNotFound
		}

		if lot.ClientID != targetClientID {
			reason := "BLOKIR KEAMANAN: Bahan milik klien " + lot.ClientName + " tidak dapat dikeluarkan untuk order " + targetOrderCode + " milik klien " + targetClientName
			_ = m.RecordBlockedAttempt(ctx, &BlockedAttemptRecord{
				AttemptedAt:       time.Now(),
				AttemptedByUserID: userID,
				LotID:             it.LotID,
				LotNumber:         lot.LotNumber,
				LotClientID:       lot.ClientID,
				TargetOrderID:     req.TargetOrderID,
				TargetOrderCode:   targetOrderCode,
				TargetClientID:    targetClientID,
				RequestedQuantity: it.Quantity,
				Reason:            reason,
			})
			return nil, ErrCrossContaminationBlocked
		}

		if lot.Status != "AVAILABLE" {
			return nil, ErrLotNotAvailable
		}

		if lot.CurrentQuantity < it.Quantity {
			return nil, ErrInsufficientStock
		}

		lot.CurrentQuantity -= it.Quantity
		if lot.CurrentQuantity <= 0 {
			lot.Status = "DEPLETED"
		}

		itemsPayload = append(itemsPayload, IssueItemPayload{
			ID:                   "issue-item-" + it.LotID,
			LotID:                lot.ID,
			LotNumber:            lot.LotNumber,
			MaterialCode:         lot.MaterialCode,
			MaterialName:         lot.MaterialName,
			ClientName:           lot.ClientName,
			Quantity:             it.Quantity,
			Unit:                 lot.Unit,
			RemainingLotQuantity: lot.CurrentQuantity,
			Notes:                it.Notes,
		})
	}

	issueID := "iss-test-01"
	detail := &IssueDetail{
		IssueSummary: IssueSummary{
			ID:                 issueID,
			IssueNumber:        req.IssueNumber,
			TargetOrderID:      req.TargetOrderID,
			TargetOrderCode:    targetOrderCode,
			TargetClientName:   targetClientName,
			CuttingTableNumber: req.CuttingTableNumber,
			IssuedDate:         req.IssuedDate,
			TotalItemsCount:    len(itemsPayload),
			CreatedAt:          time.Now(),
		},
		IssuedByName: "Warehouse Staff",
		Notes:        req.Notes,
		Items:        itemsPayload,
	}
	m.issues[issueID] = detail
	return detail, nil
}

func (m *mockInventoryRepository) FindIssueByID(ctx context.Context, id string) (*IssueDetail, error) {
	if iss, ok := m.issues[id]; ok {
		return iss, nil
	}
	return nil, ErrIssueNotFound
}

func (m *mockInventoryRepository) ListIssues(ctx context.Context, filter ListIssuesFilter) ([]IssueSummary, int, error) {
	var list []IssueSummary
	for _, iss := range m.issues {
		if filter.OrderID != "" && iss.TargetOrderID != filter.OrderID {
			continue
		}
		list = append(list, iss.IssueSummary)
	}
	return list, len(list), nil
}

func (m *mockInventoryRepository) RecordBlockedAttempt(ctx context.Context, record *BlockedAttemptRecord) error {
	record.ID = "blk-01"
	m.blockedAttempts = append(m.blockedAttempts, record)
	return nil
}

func (m *mockInventoryRepository) ListBlockedAttempts(ctx context.Context, filter ListBlockedAttemptsFilter) ([]BlockedAttemptRecord, int, error) {
	var list []BlockedAttemptRecord
	for _, r := range m.blockedAttempts {
		list = append(list, *r)
	}
	return list, len(list), nil
}

func (m *mockInventoryRepository) CreateScrap(ctx context.Context, userID string, req CreateScrapRequest) (*CuttingScrapRecord, error) {
	scrap := &CuttingScrapRecord{
		ID:             "scrap-001",
		OrderID:        req.OrderID,
		OrderCode:      m.orderNumbers[req.OrderID],
		MaterialID:     req.MaterialID,
		MaterialName:   "Cotton Combed 30s",
		ScrapCategory:  req.ScrapCategory,
		Quantity:       req.Quantity,
		Unit:           req.Unit,
		Disposition:    req.Disposition,
		Notes:          req.Notes,
		RecordedByName: "Warehouse Operator",
		CreatedAt:      time.Now(),
	}
	m.scraps = append(m.scraps, scrap)
	return scrap, nil
}

func (m *mockInventoryRepository) ListScraps(ctx context.Context, filter ListScrapsFilter) ([]CuttingScrapRecord, int, error) {
	var list []CuttingScrapRecord
	for _, s := range m.scraps {
		if filter.OrderID != "" && s.OrderID != filter.OrderID {
			continue
		}
		if filter.ScrapCategory != "" && s.ScrapCategory != filter.ScrapCategory {
			continue
		}
		list = append(list, *s)
	}
	return list, len(list), nil
}

func setupTestRouter(repo Repository, role string, clientID *string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := NewService(repo)
	handler := NewHandler(svc)

	authnMock := func(c *gin.Context) {
		c.Set("user_id", "user-test-123")
		c.Set("role", role)
		if clientID != nil {
			c.Set("client_id", *clientID)
		}
		c.Next()
	}

	api := r.Group("/api/v1")
	RegisterRoutes(api, handler, authnMock)
	return r
}

func TestCreateMaterial_Success_ByWarehouse(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "warehouse", nil)

	body := `{"code":"MAT-CTN-30S","name":"Cotton Combed 30s","category":"FABRIC","defaultUnit":"KG"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/materials", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateMaterial_Success_ByOwner(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "owner_executive", nil)

	body := `{"code":"MAT-ZIP-05","name":"Zipper YKK 05","category":"ACCESSORY","defaultUnit":"PCS"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/materials", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateMaterial_Forbidden_ByTailor(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "tailor", nil)

	body := `{"code":"MAT-ZIP-05","name":"Zipper YKK 05","category":"ACCESSORY","defaultUnit":"PCS"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/materials", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}

func TestCreateMaterial_DuplicateCode_Conflict(t *testing.T) {
	repo := newMockRepository()
	repo.materials["mat-1"] = &Material{Code: "MAT-CTN-30S", Name: "Kain"}
	r := setupTestRouter(repo, "warehouse", nil)

	body := `{"code":"MAT-CTN-30S","name":"Cotton Combed 30s","category":"FABRIC","defaultUnit":"KG"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/materials", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", w.Code)
	}
}

func TestListMaterials_Success(t *testing.T) {
	repo := newMockRepository()
	repo.materials["m-1"] = &Material{ID: "m-1", Code: "MAT-1", Name: "Cotton Combed", Category: "FABRIC"}
	repo.materials["m-2"] = &Material{ID: "m-2", Code: "MAT-2", Name: "Rib Spandex", Category: "FABRIC"}
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/materials", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool       `json:"success"`
		Total   int        `json:"total"`
		Data    []Material `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected 2 materials, got %d", resp.Total)
	}
}

func TestGetMaterial_Success(t *testing.T) {
	repo := newMockRepository()
	repo.materials["m-1"] = &Material{ID: "m-1", Code: "MAT-1", Name: "Cotton Combed", Category: "FABRIC"}
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/materials/m-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestGetMaterial_NotFound(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/materials/non-existent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestCreateReceipt_Success_WithMandatoryTagging(t *testing.T) {
	repo := newMockRepository()
	repo.validClientOrders["order-sno-1"] = "client-sno"
	repo.materials["mat-1"] = &Material{ID: "mat-1", Code: "MAT-1"}
	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"receiptNumber": "RCV-202610-001",
		"clientId": "client-sno",
		"orderId": "order-sno-1",
		"receivedDate": "2026-10-15",
		"supplierName": "PT Grand Textile",
		"items": [
			{
				"materialId": "mat-1",
				"lotNumber": "LOT-SNO-01",
				"quantity": 150.50,
				"unit": "KG",
				"rollCount": 6,
				"warehouseLocation": "RAK-A1"
			}
		]
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool          `json:"success"`
		Data    ReceiptDetail `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ReceiptNumber != "RCV-202610-001" {
		t.Errorf("expected receipt number RCV-202610-001, got %s", resp.Data.ReceiptNumber)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
	}
	// Verify lot created
	if len(repo.lots) != 1 {
		t.Errorf("expected 1 lot created in inventory, got %d", len(repo.lots))
	}
}

func TestCreateReceipt_ClientOrderMismatch_Rejected(t *testing.T) {
	repo := newMockRepository()
	// Order belongs to client-sno, but receipt body specifies client-urban
	repo.validClientOrders["order-sno-1"] = "client-sno"
	repo.materials["mat-1"] = &Material{ID: "mat-1"}
	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"receiptNumber": "RCV-FAIL-001",
		"clientId": "client-urban",
		"orderId": "order-sno-1",
		"receivedDate": "2026-10-15",
		"items": [
			{
				"materialId": "mat-1",
				"lotNumber": "LOT-01",
				"quantity": 100,
				"unit": "KG",
				"warehouseLocation": "RAK-A1"
			}
		]
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for mismatch tagging, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "INVENTORY_CLIENT_ORDER_MISMATCH" {
		t.Errorf("expected error code INVENTORY_CLIENT_ORDER_MISMATCH, got %s", resp.Error.Code)
	}
}

func TestCreateReceipt_EmptyItems_Rejected(t *testing.T) {
	repo := newMockRepository()
	repo.validClientOrders["order-1"] = "client-1"
	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"receiptNumber": "RCV-EMPTY",
		"clientId": "client-1",
		"orderId": "order-1",
		"receivedDate": "2026-10-15",
		"items": []
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty items, got %d", w.Code)
	}
}

func TestCreateReceipt_Forbidden_ByClient(t *testing.T) {
	repo := newMockRepository()
	cid := "client-sno"
	r := setupTestRouter(repo, "client", &cid)

	body := `{"receiptNumber":"RCV-1","clientId":"client-sno","orderId":"order-1","receivedDate":"2026-10-15","items":[]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/receipts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for client creating receipt, got %d", w.Code)
	}
}

func TestListReceipts_Success(t *testing.T) {
	repo := newMockRepository()
	repo.receipts["r-1"] = &ReceiptDetail{
		ReceiptSummary: ReceiptSummary{
			ID:            "r-1",
			ReceiptNumber: "RCV-001",
			ClientID:      "c-1",
			ClientName:    "Client A",
		},
	}
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/receipts", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestGetReceipt_Success(t *testing.T) {
	repo := newMockRepository()
	repo.receipts["r-1"] = &ReceiptDetail{
		ReceiptSummary: ReceiptSummary{
			ID:            "r-1",
			ReceiptNumber: "RCV-001",
		},
		ReceivedByName: "Agus Pratama",
	}
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/receipts/r-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

func TestListStocks_InternalStaff_Success(t *testing.T) {
	repo := newMockRepository()
	repo.lots["lot-1"] = &InventoryLot{
		ID:              "lot-1",
		MaterialID:      "m-1",
		MaterialName:    "Cotton Combed 30s",
		ClientID:        "c-1",
		ClientName:      "Client A",
		OrderID:         "o-1",
		OrderCode:       "PO-1",
		CurrentQuantity: 100.0,
		Unit:            "KG",
	}
	repo.lots["lot-2"] = &InventoryLot{
		ID:              "lot-2",
		MaterialID:      "m-1",
		MaterialName:    "Cotton Combed 30s",
		ClientID:        "c-2",
		ClientName:      "Client B",
		OrderID:         "o-2",
		OrderCode:       "PO-2",
		CurrentQuantity: 80.0,
		Unit:            "KG",
	}
	r := setupTestRouter(repo, "production_head", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/stocks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Total   int            `json:"total"`
		Data    []StockBalance `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected 2 stock balances for internal staff, got %d", resp.Total)
	}
}

func TestListStocks_Client_MultiTenantIsolation(t *testing.T) {
	repo := newMockRepository()
	repo.lots["lot-1"] = &InventoryLot{
		ID:              "lot-1",
		MaterialID:      "m-1",
		MaterialName:    "Cotton Combed 30s",
		ClientID:        "client-sno",
		ClientName:      "Snøhund",
		OrderID:         "o-1",
		OrderCode:       "PO-1",
		CurrentQuantity: 100.0,
		Unit:            "KG",
	}
	repo.lots["lot-2"] = &InventoryLot{
		ID:              "lot-2",
		MaterialID:      "m-1",
		MaterialName:    "Cotton Combed 30s",
		ClientID:        "client-other",
		ClientName:      "Other Client",
		OrderID:         "o-2",
		OrderCode:       "PO-2",
		CurrentQuantity: 80.0,
		Unit:            "KG",
	}

	// Caller is client-sno
	cid := "client-sno"
	r := setupTestRouter(repo, "client", &cid)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/stocks", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Total   int            `json:"total"`
		Data    []StockBalance `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	// Must ONLY see client-sno stocks (multi-tenant isolation)
	if resp.Total != 1 {
		t.Fatalf("MULTI-TENANT LEAK: expected 1 stock for client-sno, got %d", resp.Total)
	}
	if resp.Data[0].ClientID != "client-sno" {
		t.Errorf("expected client-sno stock, got %s", resp.Data[0].ClientID)
	}
}

func TestGetLot_Success(t *testing.T) {
	repo := newMockRepository()
	repo.lots["lot-1"] = &InventoryLot{
		ID:              "lot-1",
		LotNumber:       "LOT-001",
		MaterialName:    "Cotton Combed 30s",
		CurrentQuantity: 50.0,
		Unit:            "KG",
	}
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/lots/lot-1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    InventoryLot `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.LotNumber != "LOT-001" {
		t.Errorf("expected LOT-001, got %s", resp.Data.LotNumber)
	}
}

func TestGetLot_NotFound(t *testing.T) {
	repo := newMockRepository()
	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/lots/non-existent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

// ==========================================
// TAHAP 3.2: OUTBOUND & BLOKADE UNIT TESTS
// ==========================================

func TestCreateIssue_Success_ByWarehouse(t *testing.T) {
	repo := newMockRepository()
	repo.validClientOrders["order-sno-1"] = "client-sno"
	repo.orderNumbers["order-sno-1"] = "PO-SNO-01"
	repo.clientNames["client-sno"] = "Snøhund Brand"

	repo.lots["lot-sno-1"] = &InventoryLot{
		ID:              "lot-sno-1",
		LotNumber:       "LOT-SNO-001",
		MaterialID:      "mat-ctn-blk",
		MaterialCode:    "MAT-CTN-BLK",
		MaterialName:    "Cotton Combed 30s Hitam",
		ClientID:        "client-sno",
		ClientName:      "Snøhund Brand",
		CurrentQuantity: 100.0,
		Unit:            "KG",
		Status:          "AVAILABLE",
	}

	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"targetOrderId": "order-sno-1",
		"cuttingTableNumber": "MEJA-01",
		"issuedDate": "2026-10-15",
		"notes": "Pengeluaran bahan uji sukses",
		"items": [
			{"lotId": "lot-sno-1", "quantity": 40.0, "notes": "Ambil 40kg"}
		]
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool        `json:"success"`
		Data    IssueDetail `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Data.TargetOrderCode != "PO-SNO-01" {
		t.Errorf("expected order code PO-SNO-01, got %s", resp.Data.TargetOrderCode)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
	}
	if resp.Data.Items[0].RemainingLotQuantity != 60.0 {
		t.Errorf("expected remaining quantity 60.0, got %.2f", resp.Data.Items[0].RemainingLotQuantity)
	}
}

// SKENARIO UJI 1: BLOKADE KONTAMINASI SILANG
func TestCreateIssue_CrossContaminationBlocked(t *testing.T) {
	repo := newMockRepository()
	// Target order is for Snøhund
	repo.validClientOrders["order-sno-1"] = "client-sno"
	repo.orderNumbers["order-sno-1"] = "PO-SNO-01"
	repo.clientNames["client-sno"] = "Snøhund Brand"

	// But Lot belongs to Aventura!
	repo.clientNames["client-ave"] = "Aventura Outdoor"
	repo.lots["lot-ave-1"] = &InventoryLot{
		ID:              "lot-ave-1",
		LotNumber:       "LOT-AVE-001",
		MaterialID:      "mat-ctn-wht",
		MaterialCode:    "MAT-CTN-WHT",
		MaterialName:    "Cotton Combed 30s Putih",
		ClientID:        "client-ave",
		ClientName:      "Aventura Outdoor",
		CurrentQuantity: 100.0,
		Unit:            "KG",
		Status:          "AVAILABLE",
	}

	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"targetOrderId": "order-sno-1",
		"cuttingTableNumber": "MEJA-01",
		"issuedDate": "2026-10-15",
		"notes": "Uji kontaminasi silang",
		"items": [
			{"lotId": "lot-ave-1", "quantity": 30.0}
		]
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Must be rejected with 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)

	if errResp.Error.Code != "INVENTORY_CLIENT_ORDER_MISMATCH" {
		t.Errorf("expected INVENTORY_CLIENT_ORDER_MISMATCH, got %s", errResp.Error.Code)
	}

	// Verify lot current quantity was NOT deducted
	if repo.lots["lot-ave-1"].CurrentQuantity != 100.0 {
		t.Errorf("SECURITY FLAW: Lot quantity was deducted despite cross contamination blockade! Got %.2f", repo.lots["lot-ave-1"].CurrentQuantity)
	}

	// Verify audit log recorded the blocked attempt
	if len(repo.blockedAttempts) != 1 {
		t.Fatalf("expected 1 blocked attempt in audit log, got %d", len(repo.blockedAttempts))
	}
	if repo.blockedAttempts[0].LotNumber != "LOT-AVE-001" {
		t.Errorf("expected lot LOT-AVE-001 in audit, got %s", repo.blockedAttempts[0].LotNumber)
	}
}

func TestCreateIssue_InsufficientStock(t *testing.T) {
	repo := newMockRepository()
	repo.validClientOrders["order-sno-1"] = "client-sno"
	repo.orderNumbers["order-sno-1"] = "PO-SNO-01"
	repo.clientNames["client-sno"] = "Snøhund Brand"

	repo.lots["lot-sno-1"] = &InventoryLot{
		ID:              "lot-sno-1",
		LotNumber:       "LOT-SNO-001",
		ClientID:        "client-sno",
		CurrentQuantity: 25.0,
		Unit:            "KG",
		Status:          "AVAILABLE",
	}

	r := setupTestRouter(repo, "warehouse", nil)

	body := `{
		"targetOrderId": "order-sno-1",
		"issuedDate": "2026-10-15",
		"items": [
			{"lotId": "lot-sno-1", "quantity": 50.0}
		]
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/issues", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for insufficient stock, got %d", w.Code)
	}
}

func TestListIssues_Success(t *testing.T) {
	repo := newMockRepository()
	repo.issues["iss-1"] = &IssueDetail{
		IssueSummary: IssueSummary{
			ID:               "iss-1",
			IssueNumber:      "ISS-20261014-0001",
			TargetOrderID:    "order-1",
			TargetOrderCode:  "PO-001",
			TargetClientName: "Snøhund",
			IssuedDate:       "2026-10-14",
			TotalItemsCount:  1,
		},
	}

	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/issues", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Total   int            `json:"total"`
		Data    []IssueSummary `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("expected 1 issue, got %d", resp.Total)
	}
}

func TestListBlockedAttempts_AuditTrail(t *testing.T) {
	repo := newMockRepository()
	repo.blockedAttempts = append(repo.blockedAttempts, &BlockedAttemptRecord{
		ID:                "blk-1",
		LotNumber:         "LOT-X",
		LotClientName:     "Brand X",
		TargetOrderCode:   "PO-Y",
		TargetClientName:  "Brand Y",
		RequestedQuantity: 20.0,
		Reason:            "Blokir uji coba",
	})

	r := setupTestRouter(repo, "warehouse", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/inventory/blocked-attempts", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp struct {
		Success bool                   `json:"success"`
		Total   int                    `json:"total"`
		Data    []BlockedAttemptRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Total != 1 || len(resp.Data) != 1 {
		t.Fatalf("expected 1 blocked attempt, got %d", resp.Total)
	}
}

func TestCreateScrap_Success(t *testing.T) {
	repo := newMockRepository()
	repo.orderNumbers["order-1"] = "PO-SNO-01"

	r := setupTestRouter(repo, "production_head", nil)

	body := `{
		"orderId": "order-1",
		"materialId": "mat-1",
		"scrapCategory": "PERCA_KECIL",
		"quantity": 3.5,
		"unit": "KG",
		"disposition": "SIMPAN_GUDANG_SISA",
		"notes": "Sisa pola lengan"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/inventory/scraps", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool               `json:"success"`
		Data    CuttingScrapRecord `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.Success {
		t.Errorf("expected success true")
	}
	if resp.Data.Quantity != 3.5 {
		t.Errorf("expected quantity 3.5, got %.2f", resp.Data.Quantity)
	}
}


