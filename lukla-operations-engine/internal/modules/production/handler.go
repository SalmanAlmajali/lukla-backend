package production

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler handles HTTP requests for production lines and allocations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new production Handler instance.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers production endpoints with the given Gin router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, authnMiddleware gin.HandlerFunc) {
	lines := rg.Group("/lines")
	lines.Use(authnMiddleware)
	{
		lines.GET("", h.ListLines)
		lines.POST("", h.CreateLine)
		lines.GET("/:id", h.GetLineDetail)
		lines.PUT("/:id", h.UpdateLine)
	}

	allocs := rg.Group("/line-allocations")
	allocs.Use(authnMiddleware)
	{
		allocs.GET("", h.ListAllocations)
		allocs.POST("", h.CreateAllocation)
		allocs.POST("/:id/transfer", h.TransferOperator)
		allocs.GET("/:id/history", h.ListTransferHistories)
	}

	outputs := rg.Group("/production-outputs")
	outputs.Use(authnMiddleware)
	{
		outputs.GET("", h.ListOutputs)
		outputs.POST("", h.CreateOutput)
		outputs.POST("/batch", h.CreateOutputsBatch)
		outputs.GET("/:id", h.GetOutputDetail)
	}

	assemblies := rg.Group("/assemblies")
	assemblies.Use(authnMiddleware)
	{
		assemblies.GET("", h.ListAssemblies)
		assemblies.POST("", h.CreateAssembly)
		assemblies.GET("/queue-qc", h.GetQCQueue)
	}

	dashboard := rg.Group("/production-dashboard")
	dashboard.Use(authnMiddleware)
	{
		dashboard.GET("/realtime", h.GetRealtimeMetrics)
		dashboard.GET("/lines/:line_id", h.GetLineDailyProgress)
		dashboard.GET("/orders/:order_id", h.GetOrderProductionProgress)
	}
}

// ListLines handles GET /api/v1/lines.
func (h *Handler) ListLines(c *gin.Context) {
	actor := getActor(c)
	lines, err := h.svc.ListLines(c.Request.Context(), actor)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    lines,
	})
}

// GetLineDetail handles GET /api/v1/lines/:id.
func (h *Handler) GetLineDetail(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	detail, err := h.svc.GetLineDetail(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// CreateLine handles POST /api/v1/lines.
func (h *Handler) CreateLine(c *gin.Context) {
	actor := getActor(c)
	var req CreateLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter body request tidak valid: " + err.Error(),
			},
		})
		return
	}

	line, err := h.svc.CreateLine(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    line,
	})
}

// UpdateLine handles PUT /api/v1/lines/:id.
func (h *Handler) UpdateLine(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	var req UpdateLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter body request tidak valid: " + err.Error(),
			},
		})
		return
	}

	detail, err := h.svc.UpdateLine(c.Request.Context(), actor, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ListAllocations handles GET /api/v1/line-allocations.
func (h *Handler) ListAllocations(c *gin.Context) {
	actor := getActor(c)
	var filter ListAllocationsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Filter parameter tidak valid: " + err.Error(),
			},
		})
		return
	}

	allocs, err := h.svc.ListAllocations(c.Request.Context(), actor, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    allocs,
	})
}

// CreateAllocation handles POST /api/v1/line-allocations.
func (h *Handler) CreateAllocation(c *gin.Context) {
	actor := getActor(c)
	var req CreateAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter body alokasi tidak valid: " + err.Error(),
			},
		})
		return
	}

	alloc, err := h.svc.CreateAllocation(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    alloc,
	})
}

// TransferOperator handles POST /api/v1/line-allocations/:id/transfer.
func (h *Handler) TransferOperator(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	var req TransferOperatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter transfer tidak valid: " + err.Error(),
			},
		})
		return
	}

	result, err := h.svc.TransferOperator(c.Request.Context(), actor, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ListTransferHistories handles GET /api/v1/line-allocations/:id/history.
func (h *Handler) ListTransferHistories(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	logs, err := h.svc.ListTransferHistories(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
	})
}

// ListOutputs handles GET /api/v1/production-outputs.
func (h *Handler) ListOutputs(c *gin.Context) {
	actor := getActor(c)
	var filter ListOutputsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter filter output tidak valid: " + err.Error(),
			},
		})
		return
	}

	outputs, err := h.svc.ListOutputs(c.Request.Context(), actor, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    outputs,
	})
}

// GetOutputDetail handles GET /api/v1/production-outputs/:id.
func (h *Handler) GetOutputDetail(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	out, err := h.svc.GetOutputDetail(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    out,
	})
}

// CreateOutput handles POST /api/v1/production-outputs (single 30-min output by Mandor).
func (h *Handler) CreateOutput(c *gin.Context) {
	actor := getActor(c)
	var req CreateOutputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter body output tidak valid: " + err.Error(),
			},
		})
		return
	}

	out, err := h.svc.CreateOutput(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    out,
	})
}

// CreateOutputsBatch handles POST /api/v1/production-outputs/batch (batch 30-min output by Mandor).
func (h *Handler) CreateOutputsBatch(c *gin.Context) {
	actor := getActor(c)
	var req BatchCreateOutputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Parameter batch output tidak valid: " + err.Error(),
			},
		})
		return
	}

	outputs, err := h.svc.CreateOutputsBatch(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    outputs,
	})
}

// ListAssemblies handles GET /api/v1/assemblies.
func (h *Handler) ListAssemblies(c *gin.Context) {
	var filter ListAssembliesFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{"code": "INVALID_QUERY_PARAMS", "message": err.Error()},
		})
		return
	}
	records, err := h.svc.ListAssemblies(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    records,
	})
}

// CreateAssembly handles POST /api/v1/assemblies.
func (h *Handler) CreateAssembly(c *gin.Context) {
	actor := getActor(c)
	var req CreateAssemblyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{"code": "INVALID_REQUEST_BODY", "message": err.Error()},
		})
		return
	}
	rec, err := h.svc.CreateAssembly(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    rec,
	})
}

// GetQCQueue handles GET /api/v1/assemblies/queue-qc.
func (h *Handler) GetQCQueue(c *gin.Context) {
	var filter ListQCQueueFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{"code": "INVALID_QUERY_PARAMS", "message": err.Error()},
		})
		return
	}
	items, err := h.svc.GetQCQueue(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
	})
}

// GetRealtimeMetrics handles GET /api/v1/production-dashboard/realtime.
func (h *Handler) GetRealtimeMetrics(c *gin.Context) {
	metrics, err := h.svc.GetRealtimeMetrics(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// GetLineDailyProgress handles GET /api/v1/production-dashboard/lines/:line_id.
func (h *Handler) GetLineDailyProgress(c *gin.Context) {
	lineID := c.Param("line_id")
	date := c.Query("date")
	progress, err := h.svc.GetLineDailyProgress(c.Request.Context(), lineID, date)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    progress,
	})
}

// GetOrderProductionProgress handles GET /api/v1/production-dashboard/orders/:order_id.
func (h *Handler) GetOrderProductionProgress(c *gin.Context) {
	orderID := c.Param("order_id")
	progress, err := h.svc.GetOrderProductionProgress(c.Request.Context(), orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    progress,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbiddenProduction):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrLineNotFound),
		errors.Is(err, ErrAllocationNotFound),
		errors.Is(err, ErrOperatorNotFound),
		errors.Is(err, ErrOrderNotFound),
		errors.Is(err, ErrDestinationLineNotFound),
		errors.Is(err, ErrOutputNotFound),
		errors.Is(err, ErrAssemblyNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInvalidQuantity),
		errors.Is(err, ErrTimeSlotRequired),
		errors.Is(err, ErrWorkDateRequired),
		errors.Is(err, ErrEmptyBatchItems),
		errors.Is(err, ErrInvalidAssemblyQty),
		errors.Is(err, ErrAssembledDateRequired):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInsufficientComponents):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSUFFICIENT_COMPONENTS",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrArticleMismatch):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ARTICLE_MISMATCH",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrExceedsOrderTarget):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "EXCEEDS_ORDER_TARGET",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrLineCodeExists):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "LINE_CODE_EXISTS",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrLineCapacityExceeded),
		errors.Is(err, ErrFactoryCapacityExceeded):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CAPACITY_EXCEEDED",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrOperatorAlreadyAllocated):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "OPERATOR_ALREADY_ALLOCATED",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrSameLineTransfer):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "SAME_LINE_TRANSFER",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrAllocationNotActive):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ALLOCATION_NOT_ACTIVE",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrTransferReasonRequired):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "TRANSFER_REASON_REQUIRED",
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

