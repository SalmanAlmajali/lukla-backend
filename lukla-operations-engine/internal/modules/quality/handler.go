package quality

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler handles HTTP requests for QC and defect tracking endpoints (BPMN 06).
type Handler struct {
	svc *Service
}

// NewHandler creates a new quality Handler instance.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers quality endpoints on the Gin router.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	qc := rg.Group("/qc")
	qc.Use(authnMiddleware)
	{
		// Defect Categories Master
		qc.GET("/defect-categories", h.ListDefectCategories)
		qc.POST("/defect-categories", h.CreateDefectCategory)

		// QC Inspection Queue
		qc.GET("/queue", h.ListQCQueue)

		// QC Inspections
		qc.GET("/inspections", h.ListInspections)
		qc.POST("/inspections", h.CreateInspection)
		qc.GET("/inspections/:id", h.GetInspectionDetail)

		// QC Quality Analytics / Summary per Order
		qc.GET("/orders/:orderId/summary", h.GetOrderQualitySummary)
	}
}

// ListDefectCategories handles GET /api/v1/qc/defect-categories.
func (h *Handler) ListDefectCategories(c *gin.Context) {
	actor := getActor(c)
	var isActive *bool
	if activeStr := c.Query("isActive"); activeStr != "" {
		val := activeStr == "true" || activeStr == "1"
		isActive = &val
	}

	categories, err := h.svc.ListDefectCategories(c.Request.Context(), actor, isActive)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    categories,
	})
}

// CreateDefectCategory handles POST /api/v1/qc/defect-categories.
func (h *Handler) CreateDefectCategory(c *gin.Context) {
	actor := getActor(c)
	var req CreateDefectCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	cat, err := h.svc.CreateDefectCategory(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    cat,
	})
}

// ListQCQueue handles GET /api/v1/qc/queue.
func (h *Handler) ListQCQueue(c *gin.Context) {
	actor := getActor(c)
	orderID := c.Query("orderId")
	lineID := c.Query("lineId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListQCQueueFilter{
		OrderID: orderID,
		LineID:  lineID,
		Limit:   limit,
		Offset:  offset,
	}

	items, total, err := h.svc.ListQCQueue(c.Request.Context(), actor, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    items,
	})
}

// ListInspections handles GET /api/v1/qc/inspections.
func (h *Handler) ListInspections(c *gin.Context) {
	actor := getActor(c)
	orderID := c.Query("orderId")
	status := c.Query("status")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListQCInspectionsFilter{
		OrderID:   orderID,
		Status:    status,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	}

	summaries, total, err := h.svc.ListInspections(c.Request.Context(), actor, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    summaries,
	})
}

// CreateInspection handles POST /api/v1/qc/inspections.
func (h *Handler) CreateInspection(c *gin.Context) {
	actor := getActor(c)
	var req CreateQCInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	detail, err := h.svc.CreateInspection(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Pemeriksaan QC berhasil dicatat",
		"data":    detail,
	})
}

// GetInspectionDetail handles GET /api/v1/qc/inspections/:id.
func (h *Handler) GetInspectionDetail(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")

	detail, err := h.svc.GetInspectionDetail(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetOrderQualitySummary handles GET /api/v1/qc/orders/:orderId/summary.
func (h *Handler) GetOrderQualitySummary(c *gin.Context) {
	actor := getActor(c)
	orderID := c.Param("orderId")

	summary, err := h.svc.GetOrderQualitySummary(c.Request.Context(), actor, orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDefectCategoryNotFound),
		errors.Is(err, ErrInspectionNotFound),
		errors.Is(err, ErrAssemblyNotFound),
		errors.Is(err, ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrDefectCategoryCodeExists):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DUPLICATE_CODE",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrForbiddenQuality):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInvalidQuantity),
		errors.Is(err, ErrDefectDetailsRequired),
		errors.Is(err, ErrDefectQuantityMismatch):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_FAILED",
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

