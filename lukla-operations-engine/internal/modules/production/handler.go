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
		errors.Is(err, ErrDestinationLineNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
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

