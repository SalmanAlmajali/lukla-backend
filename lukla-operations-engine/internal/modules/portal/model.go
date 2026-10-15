package portal

import (
	"errors"
	"time"

	"github.com/lukla/operations-engine/internal/modules/sample"
)

// Sentinel domain errors for client portal and multi-tenant isolation.
var (
	ErrUnauthorizedPortalAccess   = errors.New("pengguna tidak memiliki hak akses ke portal klien")
	ErrClientTenantRequired       = errors.New("pengguna tidak terhubung dengan akun klien manapun")
	ErrOrderNotFound              = errors.New("pesanan tidak ditemukan atau bukan milik klien Anda")
	ErrSampleNotFound             = errors.New("data sample tidak ditemukan atau bukan milik klien Anda")
	ErrForbiddenTenantCrossAccess = errors.New("pelanggaran keamanan: akses data lintas klien dilarang keras")
)

// ClientPortalDashboard summarizes key metrics for the authenticated client.
type ClientPortalDashboard struct {
	Client                   ClientInfo `json:"client"`
	ActiveOrdersCount        int        `json:"activeOrdersCount"`
	CompletedOrdersCount     int        `json:"completedOrdersCount"`
	ReadyFinishedGoodsCount  int        `json:"readyFinishedGoodsCount"`
	ActiveMaterialsCount     int        `json:"activeMaterialsCount"`
	UnreadNotificationsCount int        `json:"unreadNotificationsCount"`
}

// ClientInfo represents basic client profile.
type ClientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// ClientPortalOrderSummary represents high-level order status in the portal.
type ClientPortalOrderSummary struct {
	ID                 string     `json:"id"`
	OrderNumber        string     `json:"orderNumber"`
	ArticleName        string     `json:"articleName"`
	ArticleCode        string     `json:"articleCode"`
	Category           string     `json:"category"`
	TargetQuantity     int        `json:"targetQuantity"`
	PassedQuantity     int        `json:"passedQuantity"`
	ProgressPercentage float64    `json:"progressPercentage"`
	Status             string     `json:"status"`
	TargetDate         *string    `json:"targetDate,omitempty"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// ClientPortalOrderDetail contains deep drill-down and 3-stage progress breakdown.
type ClientPortalOrderDetail struct {
	ID             string                `json:"id"`
	OrderNumber    string                `json:"orderNumber"`
	Article        ClientArticleInfo     `json:"article"`
	TargetQuantity int                   `json:"targetQuantity"`
	Status         string                `json:"status"`
	StartDate      *string               `json:"startDate,omitempty"`
	TargetDate     *string               `json:"targetDate,omitempty"`
	CompletedAt    *time.Time            `json:"completedAt,omitempty"`
	Notes          *string               `json:"notes,omitempty"`
	Progress       ClientOrderProgress   `json:"progress"`
}

// ClientArticleInfo represents article identity for an order.
type ClientArticleInfo struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// ClientOrderProgress represents the 3-stage manufacturing progress.
type ClientOrderProgress struct {
	CuttingQuantity   int     `json:"cuttingQuantity"`
	SewingQuantity    int     `json:"sewingQuantity"`
	PassedQCQuantity  int     `json:"passedQCQuantity"`
	RepairQCQuantity  int     `json:"repairQCQuantity"`
	RejectQCQuantity  int     `json:"rejectQCQuantity"`
	OverallPercentage float64 `json:"overallPercentage"`
}

// ClientPortalOrderQuality provides quality inspection analytics for a client order.
type ClientPortalOrderQuality struct {
	OrderID          string  `json:"orderId"`
	OrderNumber      string  `json:"orderNumber"`
	TargetQuantity   int     `json:"targetQuantity"`
	TotalChecked     int     `json:"totalChecked"`
	PassedQuantity   int     `json:"passedQuantity"`
	RepairQuantity   int     `json:"repairQuantity"`
	RejectQuantity   int     `json:"rejectQuantity"`
	PassRate         float64 `json:"passRate"`
	InspectionsCount int     `json:"inspectionsCount"`
}

// ClientPortalMaterialStock represents consigned raw materials belonging to the client.
type ClientPortalMaterialStock struct {
	MaterialID             string                 `json:"materialId"`
	MaterialCode           string                 `json:"materialCode"`
	MaterialName           string                 `json:"materialName"`
	Category               string                 `json:"category"`
	DefaultUnit            string                 `json:"defaultUnit"`
	TotalAvailableQuantity float64                `json:"totalAvailableQuantity"`
	Lots                   []ClientPortalLotStock `json:"lots"`
}

// ClientPortalLotStock represents individual batch/lot stored in warehouse.
type ClientPortalLotStock struct {
	LotID             string  `json:"lotId"`
	LotNumber         string  `json:"lotNumber"`
	AvailableQuantity float64 `json:"availableQuantity"`
	Unit              string  `json:"unit"`
	Location          string  `json:"location"`
	ReceivedDate      *string `json:"receivedDate,omitempty"`
}

// ClientPortalFinishedGoodsStock represents completed garments ready for dispatch.
type ClientPortalFinishedGoodsStock struct {
	OrderID       string `json:"orderId"`
	OrderNumber   string `json:"orderNumber"`
	ArticleCode   string `json:"articleCode"`
	ArticleName   string `json:"articleName"`
	ReadyQuantity int    `json:"readyQuantity"`
	Status        string `json:"status"`
}

// ListClientOrdersFilter represents query parameters for filtering client orders.
type ListClientOrdersFilter struct {
	Status string
	Search string
	Page   int
	Limit  int
}

// ClientPortalSampleSummary represents a sample request summary in client portal.
type ClientPortalSampleSummary struct {
	ID                string    `json:"id"`
	SampleCode        string    `json:"sampleCode"`
	ArticleTitle      string    `json:"articleTitle"`
	CurrentVersion    int       `json:"currentVersion"`
	ProposedUnitPrice *float64  `json:"proposedUnitPrice,omitempty"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	TargetDate        *string   `json:"targetDate,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
}

// ClientPortalSampleDetail represents deep drill-down of a sample in client portal.
type ClientPortalSampleDetail struct {
	ID                      string                       `json:"id"`
	SampleCode              string                       `json:"sampleCode"`
	ArticleTitle            string                       `json:"articleTitle"`
	ArticleCode             string                       `json:"articleCode"`
	Status                  string                       `json:"status"`
	CurrentVersionNumber    int                          `json:"currentVersionNumber"`
	LatestProposedUnitPrice *float64                     `json:"latestProposedUnitPrice,omitempty"`
	Currency                string                       `json:"currency"`
	Versions                []sample.SampleVersionDetail `json:"versions"`
}

// ClientPortalApproveSamplePayload represents feedback payload when client approves sample.
type ClientPortalApproveSamplePayload struct {
	FeedbackNotes *string `json:"feedbackNotes"`
}

// ClientPortalRequestRevisionPayload represents revision notes when client requests revision.
type ClientPortalRequestRevisionPayload struct {
	RevisionNotes string `json:"revisionNotes" binding:"required"`
}

