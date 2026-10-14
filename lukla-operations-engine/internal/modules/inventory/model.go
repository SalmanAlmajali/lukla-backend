package inventory

import (
	"errors"
	"time"
)

var (
	ErrMaterialNotFound     = errors.New("material tidak ditemukan")
	ErrMaterialCodeExists   = errors.New("kode material sudah digunakan")
	ErrReceiptNotFound      = errors.New("dokumen penerimaan bahan tidak ditemukan")
	ErrLotNotFound          = errors.New("lot material tidak ditemukan")
	ErrForbiddenInventory   = errors.New("forbidden: peran Anda tidak memiliki wewenang untuk tindakan inventaris ini")
	ErrClientNotFound       = errors.New("klien pemilik bahan tidak ditemukan")
	ErrOrderNotFound        = errors.New("order alokasi tidak ditemukan")
	ErrClientOrderMismatch  = errors.New("alokasi order yang dituju bukan milik klien yang bersangkutan")
	ErrReceiptItemsEmpty    = errors.New("penerimaan bahan harus menyertakan minimal 1 item lot bahan")
	ErrInvalidQuantity      = errors.New("kuantitas material harus bernilai lebih besar dari 0")
	ErrMissingRequiredField = errors.New("seluruh field wajib (kode, nama, klien, order, satuan) harus diisi")
)

type MaterialCategory string

const (
	CategoryFabric    MaterialCategory = "FABRIC"
	CategoryAccessory MaterialCategory = "ACCESSORY"
	CategoryThread    MaterialCategory = "THREAD"
	CategoryPackaging MaterialCategory = "PACKAGING"
	CategoryOther     MaterialCategory = "OTHER"
)

type LotStatus string

const (
	LotStatusAvailable LotStatus = "AVAILABLE"
	LotStatusDepleted  LotStatus = "DEPLETED"
	LotStatusReserved  LotStatus = "RESERVED"
)

type Material struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	DefaultUnit string    `json:"defaultUnit"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateMaterialRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Category    string `json:"category" binding:"required"`
	DefaultUnit string `json:"defaultUnit" binding:"required"`
	Description string `json:"description"`
}

type ListMaterialsFilter struct {
	Category string
	Search   string
	Limit    int
	Offset   int
}

type InventoryReceipt struct {
	ID                 string    `json:"id"`
	ReceiptNumber      string    `json:"receiptNumber"`
	ClientID           string    `json:"clientId"`
	ClientName         string    `json:"clientName,omitempty"`
	OrderID            string    `json:"orderId"`
	OrderCode          string    `json:"orderCode,omitempty"`
	ReceivedDate       string    `json:"receivedDate"`
	SupplierName       *string   `json:"supplierName,omitempty"`
	DeliveryNoteNumber *string   `json:"deliveryNoteNumber,omitempty"`
	ReceivedByUserID   string    `json:"receivedByUserId"`
	ReceivedByName     string    `json:"receivedByName,omitempty"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type ReceiptSummary struct {
	ID              string    `json:"id"`
	ReceiptNumber   string    `json:"receiptNumber"`
	ClientID        string    `json:"clientId"`
	ClientName      string    `json:"clientName"`
	OrderID         string    `json:"orderId"`
	OrderCode       string    `json:"orderCode"`
	ReceivedDate    string    `json:"receivedDate"`
	SupplierName    string    `json:"supplierName,omitempty"`
	TotalItemsCount int       `json:"totalItemsCount"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ReceiptDetail struct {
	ReceiptSummary
	DeliveryNoteNumber string               `json:"deliveryNoteNumber,omitempty"`
	ReceivedByName     string               `json:"receivedByName"`
	Notes              string               `json:"notes,omitempty"`
	Items              []ReceiptItemPayload `json:"items"`
}

type ReceiptItemPayload struct {
	ID                string  `json:"id"`
	MaterialID        string  `json:"materialId"`
	MaterialCode      string  `json:"materialCode"`
	MaterialName      string  `json:"materialName"`
	LotID             string  `json:"lotId"`
	LotNumber         string  `json:"lotNumber"`
	Quantity          float64 `json:"quantity"`
	Unit              string  `json:"unit"`
	RollCount         int     `json:"rollCount"`
	WarehouseLocation string  `json:"warehouseLocation"`
	Notes             string  `json:"notes,omitempty"`
}

type InventoryReceiptItem struct {
	ID                string    `json:"id"`
	ReceiptID         string    `json:"receiptId"`
	LotID             string    `json:"lotId"`
	MaterialID        string    `json:"materialId"`
	Quantity          float64   `json:"quantity"`
	Unit              string    `json:"unit"`
	RollCount         int       `json:"rollCount"`
	WarehouseLocation string    `json:"warehouseLocation"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

type CreateReceiptRequest struct {
	ReceiptNumber      string              `json:"receiptNumber" binding:"required"`
	ClientID           string              `json:"clientId" binding:"required"`
	OrderID            string              `json:"orderId" binding:"required"`
	ReceivedDate       string              `json:"receivedDate" binding:"required"`
	SupplierName       string              `json:"supplierName"`
	DeliveryNoteNumber string              `json:"deliveryNoteNumber"`
	Notes              string              `json:"notes"`
	Items              []CreateReceiptItem `json:"items" binding:"required,dive"`
}

type CreateReceiptItem struct {
	MaterialID        string  `json:"materialId" binding:"required"`
	LotNumber         string  `json:"lotNumber" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required,gt=0"`
	Unit              string  `json:"unit" binding:"required"`
	RollCount         int     `json:"rollCount"`
	WarehouseLocation string  `json:"warehouseLocation" binding:"required"`
	Notes             string  `json:"notes"`
}

type ListReceiptsFilter struct {
	ClientID  string
	OrderID   string
	StartDate string
	EndDate   string
	Limit     int
	Offset    int
}

type InventoryLot struct {
	ID                string    `json:"id"`
	ReceiptID         string    `json:"receiptId"`
	LotNumber         string    `json:"lotNumber"`
	MaterialID        string    `json:"materialId"`
	MaterialCode      string    `json:"materialCode"`
	MaterialName      string    `json:"materialName"`
	ClientID          string    `json:"clientId"`
	ClientName        string    `json:"clientName"`
	OrderID           string    `json:"orderId"`
	OrderCode         string    `json:"orderCode"`
	InitialQuantity   float64   `json:"initialQuantity"`
	CurrentQuantity   float64   `json:"currentQuantity"`
	Unit              string    `json:"unit"`
	RollCount         int       `json:"rollCount"`
	Status            string    `json:"status"`
	WarehouseLocation string    `json:"warehouseLocation"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type StockBalance struct {
	MaterialID         string  `json:"materialId"`
	MaterialCode       string  `json:"materialCode"`
	MaterialName       string  `json:"materialName"`
	ClientID           string  `json:"clientId"`
	ClientName         string  `json:"clientName"`
	OrderID            string  `json:"orderId"`
	OrderCode          string  `json:"orderCode"`
	TotalQuantity      float64 `json:"totalQuantity"`
	Unit               string  `json:"unit"`
	AvailableLotsCount int     `json:"availableLotsCount"`
}

type ListStocksFilter struct {
	ClientID   string
	MaterialID string
	OrderID    string
}

