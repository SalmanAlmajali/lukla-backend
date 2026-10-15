package notification

import (
	"errors"
	"time"
)

var (
	ErrNotificationNotFound  = errors.New("notifikasi tidak ditemukan atau bukan milik klien Anda")
	ErrClientTenantRequired  = errors.New("identitas klien diperlukan untuk mengakses notifikasi portal")
	ErrForbiddenClientPortal = errors.New("hanya akun klien yang memiliki izin untuk mengakses notifikasi portal")
	ErrInvalidEventType      = errors.New("tipe event tidak valid (harus ORDER_COMPLETED, MATERIAL_RECEIVED, PRODUCTION_MILESTONE, atau ANNOUNCEMENT)")
	ErrMissingRequiredField  = errors.New("field wajib (clientId, title, message, eventType) tidak boleh kosong")
)

const (
	EventTypeOrderCompleted            = "ORDER_COMPLETED"
	EventTypeMaterialReceived          = "MATERIAL_RECEIVED"
	EventTypeProductionMilestone       = "PRODUCTION_MILESTONE"
	EventTypeAnnouncement              = "ANNOUNCEMENT"
	EventTypeSampleReadyForReview      = "SAMPLE_READY_FOR_REVIEW"
	EventTypeSampleApproved            = "SAMPLE_APPROVED"
	EventTypeSampleRevisionRequested   = "SAMPLE_REVISION_REQUESTED"
)

// Notification merepresentasikan rekaman notifikasi in-app untuk klien portal.
type Notification struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"clientId"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	EventType   string     `json:"eventType"`
	ReferenceID *string    `json:"referenceId,omitempty"`
	IsRead      bool       `json:"isRead"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// SendNotificationRequest merepresentasikan payload kirim notifikasi manual oleh staf internal pabrik.
type SendNotificationRequest struct {
	ClientID    string  `json:"clientId" binding:"required"`
	Title       string  `json:"title" binding:"required"`
	Message     string  `json:"message" binding:"required"`
	EventType   string  `json:"eventType" binding:"required"`
	ReferenceID *string `json:"referenceId,omitempty"`
}

// ListNotificationsResponse merepresentasikan respon standar daftar notifikasi portal.
type ListNotificationsResponse struct {
	UnreadCount int            `json:"unreadCount"`
	Data        []Notification `json:"data"`
}

