package sample

import "context"

// Repository mendefinisikan kontrak persistensi data modul sample & time study.
type Repository interface {
	// Sample Requests
	CreateSampleRequest(ctx context.Context, sample *SampleRequestDetail, initialVersion *SampleVersionDetail) (*SampleRequestDetail, error)
	ListSampleRequests(ctx context.Context, clientID, status, search string, page, limit int) ([]SampleRequestSummary, int, error)
	GetSampleRequestByID(ctx context.Context, id string) (*SampleRequestDetail, error)
	UpdateSampleStatus(ctx context.Context, sampleID string, status string, notes *string) error

	// Versioning Engine
	CreateSampleVersion(ctx context.Context, sampleID string, newVersion *SampleVersionDetail, copyFromPrev bool) (*SampleVersionDetail, error)
	GetSampleVersionByNumber(ctx context.Context, sampleID string, versionNumber int) (*SampleVersionDetail, error)
	SetCommercialRate(ctx context.Context, sampleID string, versionNumber int, unitPrice float64, currency, priceNotes string) (*SampleVersionDetail, error)

	// Time Study Tracker
	RecordTimeStudy(ctx context.Context, sampleID string, versionNumber int, study *SampleTimeStudyItem) (*SampleTimeStudyItem, error)
	ListTimeStudies(ctx context.Context, sampleID string, versionNumber int) ([]SampleTimeStudyItem, error)

	// Mass Order Conversion (BPMN 03 / Skenario Uji 4)
	ConvertToOrder(ctx context.Context, sampleID string, payload ConvertSampleToOrderPayload) (*ConvertedMassOrderResponse, error)

	// Attachments (Amazon S3)
	SaveAttachment(ctx context.Context, att *SampleAttachment) (*SampleAttachment, error)
	ListAttachmentsByVersion(ctx context.Context, sampleRequestID string, versionNumber int) ([]SampleAttachment, error)
	GetAttachmentByID(ctx context.Context, id string) (*SampleAttachment, error)
	DeleteAttachment(ctx context.Context, id string) error
}

