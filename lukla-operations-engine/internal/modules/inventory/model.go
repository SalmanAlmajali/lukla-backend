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
	ErrReceiptItemsEmpty         = errors.New("penerimaan bahan harus menyertakan minimal 1 item lot bahan")
	ErrInvalidQuantity           = errors.New("kuantitas material harus bernilai lebih besar dari 0")
	ErrMissingRequiredField      = errors.New("seluruh field wajib (kode, nama, klien, order, satuan) harus diisi")
	ErrCrossContaminationBlocked = errors.New("BLOKIR KEAMANAN: Bahan tidak dapat dikeluarkan karena pemilik lot bahan berbeda dengan klien order tujuan")
	ErrInsufficientStock         = errors.New("sisa kuantitas pada lot tidak mencukupi untuk jumlah yang diminta")
	ErrLotNotAvailable           = errors.New("lot bahan tidak berstatus AVAILABLE atau stok sudah habis")
	ErrIssueNotFound             = errors.New("dokumen pengeluaran bahan tidak ditemukan")
	ErrIssueItemsEmpty           = errors.New("pengeluaran bahan harus menyertakan minimal 1 item lot bahan")
	ErrScrapCategoryInvalid      = errors.New("kategori scrap tidak valid (harus PERCA_KECIL, CACAT_KAIN_ROLL, UJUNG_GULUNGAN, atau KESALAHAN_POTONG)")
	ErrScrapDispositionInvalid   = errors.New("disposisi scrap tidak valid (harus SIMPAN_GUDANG_SISA, KEMBALIKAN_KLIEN, JUAL_LIMBAH, atau BUANG)")
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

// ==========================================
// TAHAP 3.2: OUTBOUND, ISSUES & BLOCKED ATTEMPTS
// ==========================================

type CreateIssueRequest struct {
	IssueNumber        string            `json:"issueNumber"`
	TargetOrderID      string            `json:"targetOrderId" binding:"required"`
	CuttingTableNumber string            `json:"cuttingTableNumber"`
	IssuedDate         string            `json:"issuedDate" binding:"required"`
	Notes              string            `json:"notes"`
	Items              []CreateIssueItem `json:"items" binding:"required,dive"`
}

type CreateIssueItem struct {
	LotID    string  `json:"lotId" binding:"required"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	Notes    string  `json:"notes"`
}

type IssueSummary struct {
	ID                 string    `json:"id"`
	IssueNumber        string    `json:"issueNumber"`
	TargetOrderID      string    `json:"targetOrderId"`
	TargetOrderCode    string    `json:"targetOrderCode"`
	TargetClientName   string    `json:"targetClientName"`
	CuttingTableNumber string    `json:"cuttingTableNumber,omitempty"`
	IssuedDate         string    `json:"issuedDate"`
	TotalItemsCount    int       `json:"totalItemsCount"`
	CreatedAt          time.Time `json:"createdAt"`
}

type IssueDetail struct {
	IssueSummary
	IssuedByName string             `json:"issuedByName"`
	Notes        string             `json:"notes,omitempty"`
	Items        []IssueItemPayload `json:"items"`
}

type IssueItemPayload struct {
	ID                   string  `json:"id"`
	LotID                string  `json:"lotId"`
	LotNumber            string  `json:"lotNumber"`
	MaterialCode         string  `json:"materialCode"`
	MaterialName         string  `json:"materialName"`
	ClientName           string  `json:"clientName"`
	Quantity             float64 `json:"quantity"`
	Unit                 string  `json:"unit"`
	RemainingLotQuantity float64 `json:"remainingLotQuantity"`
	Notes                string  `json:"notes,omitempty"`
}

type ListIssuesFilter struct {
	OrderID string
	Limit   int
	Offset  int
}

type BlockedAttemptRecord struct {
	ID                string    `json:"id"`
	AttemptedAt       time.Time `json:"attemptedAt"`
	AttemptedByUserID string    `json:"attemptedByUserId"`
	AttemptedByName   string    `json:"attemptedByName"`
	LotID             string    `json:"lotId"`
	LotNumber         string    `json:"lotNumber"`
	LotClientID       string    `json:"lotClientId"`
	LotClientName     string    `json:"lotClientName"`
	TargetOrderID     string    `json:"targetOrderId"`
	TargetOrderCode   string    `json:"targetOrderCode"`
	TargetClientID    string    `json:"targetClientId"`
	TargetClientName  string    `json:"targetClientName"`
	RequestedQuantity float64   `json:"requestedQuantity"`
	Reason            string    `json:"reason"`
}

type ListBlockedAttemptsFilter struct {
	Limit  int
	Offset int
}

// ==========================================
// TAHAP 3.2: CUTTING SCRAPS & WASTE
// ==========================================

type ScrapCategory string

const (
	ScrapCategoryPercaKecil       ScrapCategory = "PERCA_KECIL"
	ScrapCategoryCacatKainRoll    ScrapCategory = "CACAT_KAIN_ROLL"
	ScrapCategoryUjungGulungan    ScrapCategory = "UJUNG_GULUNGAN"
	ScrapCategoryKesalahanPotong  ScrapCategory = "KESALAHAN_POTONG"
)

type ScrapDisposition string

const (
	ScrapDispositionSimpanGudangSisa ScrapDisposition = "SIMPAN_GUDANG_SISA"
	ScrapDispositionKembalikanKlien  ScrapDisposition = "KEMBALIKAN_KLIEN"
	ScrapDispositionJualLimbah       ScrapDisposition = "JUAL_LIMBAH"
	ScrapDispositionBuang            ScrapDisposition = "BUANG"
)

type CreateScrapRequest struct {
	OrderID       string  `json:"orderId" binding:"required"`
	MaterialID    string  `json:"materialId" binding:"required"`
	ScrapCategory string  `json:"scrapCategory" binding:"required"`
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	Unit          string  `json:"unit" binding:"required"`
	Disposition   string  `json:"disposition" binding:"required"`
	Notes         string  `json:"notes"`
}

type CuttingScrapRecord struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"orderId"`
	OrderCode      string    `json:"orderCode"`
	MaterialID     string    `json:"materialId"`
	MaterialName   string    `json:"materialName"`
	ScrapCategory  string    `json:"scrapCategory"`
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"`
	Disposition    string    `json:"disposition"`
	Notes          string    `json:"notes,omitempty"`
	RecordedByName string    `json:"recordedByName"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ListScrapsFilter struct {
	OrderID       string
	ScrapCategory string
	Limit         int
	Offset        int
}

