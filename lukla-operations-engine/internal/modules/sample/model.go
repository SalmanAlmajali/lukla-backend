package sample

import (
	"errors"
	"time"
)

var (
	ErrSampleNotFound          = errors.New("permintaan sample tidak ditemukan")
	ErrVersionNotFound         = errors.New("versi sample tidak ditemukan")
	ErrComponentNotFound       = errors.New("komponen artikel tidak ditemukan")
	ErrInvalidSampleStatus     = errors.New("status sample tidak mengizinkan operasi ini")
	ErrVersionAlreadyLocked    = errors.New("versi sample telah dikunci dan tidak dapat diubah")
	ErrSampleNotApproved       = errors.New("sample belum berstatus APPROVED untuk dapat dikonversi")
	ErrSampleAlreadyConverted  = errors.New("sample telah dikonversi menjadi order massal sebelumnya")
	ErrArticleNotFound         = errors.New("artikel tidak ditemukan")
	ErrClientNotFound          = errors.New("klien tidak ditemukan")
	ErrSampleCannotBeModified  = errors.New("sample yang telah disetujui atau dikonversi tidak dapat diubah")
	ErrSampleCannotBeDeleted   = errors.New("sample yang telah disetujui atau dikonversi tidak dapat dihapus")
)

// Status konstanta siklus hidup sample dan versinya
const (
	StatusDraft             = "DRAFT"
	StatusInDevelopment     = "IN_DEVELOPMENT"
	StatusInReview          = "IN_REVIEW"
	StatusRevised           = "REVISED"
	StatusApproved          = "APPROVED"
	StatusRejected          = "REJECTED"
	StatusConvertedToOrder  = "CONVERTED_TO_ORDER"
)

// SampleRequestSummary merepresentasikan ikhtisar permintaan sample dalam daftar.
type SampleRequestSummary struct {
	ID             string     `json:"id"`
	SampleCode     string     `json:"sampleCode"`
	ClientID       string     `json:"clientId"`
	ClientName     string     `json:"clientName"`
	ArticleID      string     `json:"articleId"`
	ArticleCode    string     `json:"articleCode"`
	ArticleTitle   string     `json:"articleTitle"`
	CurrentVersion int        `json:"currentVersion"`
	Status         string     `json:"status"`
	TargetDate     *string    `json:"targetDate,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// SampleRequestDetail merepresentasikan rincian lengkap permintaan sample beserta seluruh riwayat revisi versi.
type SampleRequestDetail struct {
	ID                   string                `json:"id"`
	SampleCode           string                `json:"sampleCode"`
	ClientID             string                `json:"clientId"`
	ClientName           string                `json:"clientName"`
	ArticleID            string                `json:"articleId"`
	ArticleCode          string                `json:"articleCode"`
	ArticleTitle         string                `json:"articleTitle"`
	Status               string                `json:"status"`
	TargetDate           *string               `json:"targetDate,omitempty"`
	InitialNotes         *string               `json:"initialNotes,omitempty"`
	ConvertedOrderID     *string               `json:"convertedOrderId,omitempty"`
	ConvertedOrderNumber *string               `json:"convertedOrderNumber,omitempty"`
	CreatedAt            time.Time             `json:"createdAt"`
	UpdatedAt            time.Time             `json:"updatedAt"`
	Versions             []SampleVersionDetail `json:"versions"`
}

// SampleVersionDetail merepresentasikan detail satu versi fisik sample prototipe.
type SampleVersionDetail struct {
	ID                string                `json:"id"`
	SampleRequestID   string                `json:"sampleRequestId,omitempty"`
	VersionNumber     int                   `json:"versionNumber"`
	VersionLabel      string                `json:"versionLabel"` // e.g. "v1", "v2"
	Status            string                `json:"status"`       // DRAFT, IN_DEVELOPMENT, IN_REVIEW, REVISED, APPROVED, REJECTED
	RevisionNotes     *string               `json:"revisionNotes,omitempty"`
	ClientFeedback    *string               `json:"clientFeedback,omitempty"`
	TotalSAM          float64               `json:"totalSAM"`
	ProposedUnitPrice *float64              `json:"proposedUnitPrice,omitempty"`
	Currency          string                `json:"currency"`
	PriceNotes        *string               `json:"priceNotes,omitempty"`
	ApprovedAt        *time.Time            `json:"approvedAt,omitempty"`
	CreatedAt         time.Time             `json:"createdAt"`
	UpdatedAt         time.Time             `json:"updatedAt"`
	Components        []SampleTimeStudyItem `json:"components"`
	Attachments       []SampleAttachment    `json:"attachments"`
}

// SampleAttachment merepresentasikan metadata foto lampiran sampel pada Amazon S3.
type SampleAttachment struct {
	ID              string    `json:"id"`
	SampleRequestID string    `json:"sampleRequestId"`
	SampleVersionID *string   `json:"sampleVersionId,omitempty"`
	FileName        string    `json:"fileName"`
	S3Bucket        string    `json:"s3Bucket"`
	S3Key           string    `json:"s3Key"`
	FileURL         string    `json:"fileUrl"`
	FileSize        int64     `json:"fileSize"`
	MimeType        string    `json:"mimeType"`
	Caption         *string   `json:"caption,omitempty"`
	AttachmentType  string    `json:"attachmentType"` // PROTOTYPE_PHOTO, DEFECT_PHOTO, REVISION_REFERENCE, TECHPACK_SKETCH
	UploadedBy      *string   `json:"uploadedBy,omitempty"`
	UploaderName    string    `json:"uploaderName"`
	UploaderRole    string    `json:"uploaderRole"`
	CreatedAt       time.Time `json:"createdAt"`
}

// SampleTimeStudyItem merepresentasikan hasil observasi stopwatch time study per komponen pada versi sample.
type SampleTimeStudyItem struct {
	ID                  string    `json:"id,omitempty"`
	SampleVersionID     string    `json:"sampleVersionId,omitempty"`
	ComponentID         string    `json:"componentId"`
	ComponentName       string    `json:"componentName"`
	SequenceNumber      int       `json:"sequenceNumber"`
	ObservedSeconds     float64   `json:"observedSeconds"`
	AllowancePercentage float64   `json:"allowancePercentage"`
	CalculatedSAM       float64   `json:"calculatedSAM"`
	TargetPerHour       int       `json:"targetPerHour"`
	OperatorID          *string   `json:"operatorId,omitempty"`
	OperatorName        *string   `json:"operatorName,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

// CreateSampleRequestPayload payload saat membuat permintaan sample baru (otomatis inisialisasi v1).
type CreateSampleRequestPayload struct {
	ClientID     string  `json:"clientId" binding:"required"`
	ArticleID    string  `json:"articleId" binding:"required"`
	InitialNotes *string `json:"initialNotes"`
	TargetDate   *string `json:"targetDate"`
}

// UpdateSampleRequestPayload payload saat memperbarui informasi umum permintaan sample (artikel, klien, skema/catatan, target tanggal).
type UpdateSampleRequestPayload struct {
	ClientID     *string `json:"clientId"`
	ArticleID    *string `json:"articleId"`
	ArticleTitle *string `json:"articleTitle"`
	InitialNotes *string `json:"initialNotes"`
	TargetDate   *string `json:"targetDate"`
}

// CreateSampleVersionPayload payload saat membuat revisi versi baru (misal v1 -> v2).
type CreateSampleVersionPayload struct {
	RevisionNotes               string   `json:"revisionNotes" binding:"required"`
	CopyTimeStudiesFromPrevious *bool    `json:"copyTimeStudiesFromPrevious"`
	ProposedUnitPrice           *float64 `json:"proposedUnitPrice"`
	Currency                    *string  `json:"currency"`
	PriceNotes                  *string  `json:"priceNotes"`
}

// RecordTimeStudyPayload payload saat operator/IE menginput observasi stopwatch per komponen.
type RecordTimeStudyPayload struct {
	ComponentID         string   `json:"componentId" binding:"required"`
	ObservedSeconds     float64  `json:"observedSeconds" binding:"required,gt=0"`
	AllowancePercentage *float64 `json:"allowancePercentage"` // default 15.0%
	OperatorID          *string  `json:"operatorId"`
	Notes               *string  `json:"notes"`
}

// SetCommercialRatePayload payload saat menetapkan atau memperbarui harga maklon ke klien.
type SetCommercialRatePayload struct {
	ProposedUnitPrice float64 `json:"proposedUnitPrice" binding:"required,gte=0"`
	Currency          *string `json:"currency"`
	PriceNotes        *string `json:"priceNotes"`
}

// SubmitReviewPayload payload saat mengajukan sample ke klien untuk direview di portal.
type SubmitReviewPayload struct {
	Notes *string `json:"notes"`
}

// ConvertSampleToOrderPayload payload untuk mengonversi sample approved menjadi order massal (BPMN 03 / Skenario Uji 4).
type ConvertSampleToOrderPayload struct {
	OrderNumber    string   `json:"orderNumber" binding:"required"`
	TargetQuantity int      `json:"targetQuantity" binding:"required,gt=0"`
	TargetDate     string   `json:"targetDate" binding:"required"`
	UnitPrice      *float64 `json:"unitPrice"`
	Notes          *string  `json:"notes"`
}

// ConvertedMassOrderResponse respons kembalian setelah sample berhasil dikonversi menjadi order produksi massal.
type ConvertedMassOrderResponse struct {
	OrderID         string  `json:"orderId"`
	OrderNumber     string  `json:"orderNumber"`
	SampleCode      string  `json:"sampleCode"`
	ApprovedVersion int     `json:"approvedVersion"`
	TargetQuantity  int     `json:"targetQuantity"`
	UnitPrice       float64 `json:"unitPrice"`
	TotalAmount     float64 `json:"totalAmount"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	SpecSnapshotSAM float64 `json:"specSnapshotSAM"`
}

