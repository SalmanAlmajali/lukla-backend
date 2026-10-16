package portal

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler handles HTTP requests for client portal endpoints (BPMN 11).
type Handler struct {
	svc *Service
}

// NewHandler creates a new portal Handler instance.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers client portal endpoints on the Gin router.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	portalGroup := rg.Group("/portal")
	portalGroup.Use(authnMiddleware)
	{
		// Dashboard Summary
		portalGroup.GET("/dashboard", h.GetDashboard)

		// Order Tracking & Drill-down
		portalGroup.GET("/orders", h.ListOrders)
		portalGroup.GET("/orders/:id", h.GetOrderDetail)
		portalGroup.GET("/orders/:id/quality", h.GetOrderQuality)

		// Consigned & Finished Inventory Stocks
		portalGroup.GET("/stocks/materials", h.ListMaterialStocks)
		portalGroup.GET("/stocks/finished-goods", h.ListFinishedGoodsStocks)

		// Sample Review & Approval (BPMN 03 & BPMN 11)
		portalGroup.GET("/samples", h.ListSamples)
		portalGroup.GET("/samples/:id", h.GetSampleDetail)
		portalGroup.POST("/samples/:id/approve", h.ApproveSample)
		portalGroup.POST("/samples/:id/request-revision", h.RequestSampleRevision)
		portalGroup.POST("/samples/:id/attachments", h.UploadSampleAttachment)
	}
}

// GetDashboard handles GET /api/v1/portal/dashboard.
func (h *Handler) GetDashboard(c *gin.Context) {
	actor := getActor(c)
	dashboard, err := h.svc.GetDashboard(c.Request.Context(), actor)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dashboard,
	})
}

// ListOrders handles GET /api/v1/portal/orders.
func (h *Handler) ListOrders(c *gin.Context) {
	actor := getActor(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := ListClientOrdersFilter{
		Status: c.DefaultQuery("status", "ALL"),
		Search: c.Query("search"),
		Page:   page,
		Limit:  limit,
	}

	orders, total, err := h.svc.ListOrders(c.Request.Context(), actor, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
		"pagination": gin.H{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
		},
	})
}

// GetOrderDetail handles GET /api/v1/portal/orders/:id.
func (h *Handler) GetOrderDetail(c *gin.Context) {
	actor := getActor(c)
	orderID := c.Param("id")

	order, err := h.svc.GetOrderDetail(c.Request.Context(), actor, orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// GetOrderQuality handles GET /api/v1/portal/orders/:id/quality.
func (h *Handler) GetOrderQuality(c *gin.Context) {
	actor := getActor(c)
	orderID := c.Param("id")

	quality, err := h.svc.GetOrderQuality(c.Request.Context(), actor, orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    quality,
	})
}

// ListMaterialStocks handles GET /api/v1/portal/stocks/materials.
func (h *Handler) ListMaterialStocks(c *gin.Context) {
	actor := getActor(c)
	category := c.Query("category")

	stocks, err := h.svc.ListMaterialStocks(c.Request.Context(), actor, category)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stocks,
	})
}

// ListFinishedGoodsStocks handles GET /api/v1/portal/stocks/finished-goods.
func (h *Handler) ListFinishedGoodsStocks(c *gin.Context) {
	actor := getActor(c)

	stocks, err := h.svc.ListFinishedGoodsStocks(c.Request.Context(), actor)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stocks,
	})
}

// ListSamples handles GET /api/v1/portal/samples.
func (h *Handler) ListSamples(c *gin.Context) {
	actor := getActor(c)
	status := c.DefaultQuery("status", "ALL")

	samples, err := h.svc.ListSamples(c.Request.Context(), actor, status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    samples,
	})
}

// GetSampleDetail handles GET /api/v1/portal/samples/:id.
func (h *Handler) GetSampleDetail(c *gin.Context) {
	actor := getActor(c)
	sampleID := c.Param("id")

	detail, err := h.svc.GetSampleDetail(c.Request.Context(), actor, sampleID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ApproveSample handles POST /api/v1/portal/samples/:id/approve.
func (h *Handler) ApproveSample(c *gin.Context) {
	actor := getActor(c)
	sampleID := c.Param("id")
	var payload ClientPortalApproveSamplePayload
	_ = c.ShouldBindJSON(&payload)

	err := h.svc.ApproveSample(c.Request.Context(), actor, sampleID, payload.FeedbackNotes)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sample berhasil disetujui dan siap dikonversi ke produksi massal",
	})
}

// RequestSampleRevision handles POST /api/v1/portal/samples/:id/request-revision.
func (h *Handler) RequestSampleRevision(c *gin.Context) {
	actor := getActor(c)
	sampleID := c.Param("id")
	var payload ClientPortalRequestRevisionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Catatan revisi wajib diisi: " + err.Error(),
			},
		})
		return
	}

	err := h.svc.RequestSampleRevision(c.Request.Context(), actor, sampleID, payload.RevisionNotes)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Permintaan revisi berhasil disampaikan ke tim pabrik",
	})
}

// UploadSampleAttachment handles POST /api/v1/portal/samples/:id/attachments.
func (h *Handler) UploadSampleAttachment(c *gin.Context) {
	actor := getActor(c)
	sampleID := c.Param("id")

	verStr := c.DefaultPostForm("versionNumber", "1")
	verNum, err := strconv.Atoi(verStr)
	if err != nil || verNum < 1 {
		verNum = 1
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "File foto wajib diunggah pada field 'file': " + err.Error(),
			},
		})
		return
	}

	captionText := c.PostForm("caption")
	var captionPtr *string
	if captionText != "" {
		captionPtr = &captionText
	}
	attType := c.DefaultPostForm("attachmentType", "REVISION_REFERENCE")

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FILE_READ_ERROR",
				"message": "Gagal membaca berkas: " + err.Error(),
			},
		})
		return
	}
	defer f.Close()

	att, err := h.svc.UploadSampleAttachment(
		c.Request.Context(),
		actor,
		sampleID,
		verNum,
		f,
		fileHeader.Filename,
		fileHeader.Size,
		fileHeader.Header.Get("Content-Type"),
		captionPtr,
		attType,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Foto referensi sampel berhasil diunggah ke Amazon S3",
		"data":    att,
	})
}


func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrOrderNotFound), errors.Is(err, ErrSampleNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrUnauthorizedPortalAccess),
		errors.Is(err, ErrClientTenantRequired),
		errors.Is(err, ErrForbiddenTenantCrossAccess):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}
}

func getActor(c *gin.Context) *authn.UserIdentity {
	role, _ := authn.RoleFrom(c)
	userID, _ := authn.UserIDFrom(c)
	var clientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		clientID = &cid
	}
	return &authn.UserIdentity{
		ID:       userID,
		RoleCode: role,
		ClientID: clientID,
		IsActive: true,
	}
}

