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
	materials map[string]*Material
	receipts  map[string]*ReceiptDetail
	lots      map[string]*InventoryLot
	// For client & order validation
	validClientOrders map[string]string // orderID -> clientID
}

func newMockRepository() *mockInventoryRepository {
	return &mockInventoryRepository{
		materials:         make(map[string]*Material),
		receipts:          make(map[string]*ReceiptDetail),
		lots:              make(map[string]*InventoryLot),
		validClientOrders: make(map[string]string),
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

