package inventory

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	invGroup := rg.Group("/inventory")
	invGroup.Use(authnMiddleware)
	{
		// Materials Master
		invGroup.GET("/materials", h.ListMaterials)
		invGroup.POST("/materials", h.CreateMaterial)
		invGroup.GET("/materials/:id", h.GetMaterial)

		// Receipts (Inbound - FR-6.1)
		invGroup.GET("/receipts", h.ListReceipts)
		invGroup.POST("/receipts", h.CreateReceipt)
		invGroup.GET("/receipts/:id", h.GetReceipt)

		// Stock & Lots
		invGroup.GET("/stocks", h.ListStocks)
		invGroup.GET("/lots/:id", h.GetLot)

		// Issues (Outbound & Blockade - FR-6.2 / Skenario Uji 1)
		invGroup.GET("/issues", h.ListIssues)
		invGroup.POST("/issues", h.CreateIssue)
		invGroup.GET("/issues/:id", h.GetIssue)

		// Blocked Attempts Audit Trail
		invGroup.GET("/blocked-attempts", h.ListBlockedAttempts)

		// Cutting Scraps & Waste (FR-6.3)
		invGroup.GET("/scraps", h.ListScraps)
		invGroup.POST("/scraps", h.CreateScrap)
	}
}

// ListMaterials handles GET /api/v1/inventory/materials.
func (h *Handler) ListMaterials(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListMaterialsFilter{
		Category: category,
		Search:   search,
		Limit:    limit,
		Offset:   offset,
	}

	materials, total, err := h.svc.ListMaterials(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    materials,
	})
}

// CreateMaterial handles POST /api/v1/inventory/materials.
func (h *Handler) CreateMaterial(c *gin.Context) {
	actor := getActor(c)
	var req CreateMaterialRequest
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

	material, err := h.svc.CreateMaterial(c.Request.Context(), actor.RoleCode, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    material,
	})
}

// GetMaterial handles GET /api/v1/inventory/materials/:id.
func (h *Handler) GetMaterial(c *gin.Context) {
	id := c.Param("id")
	material, err := h.svc.GetMaterial(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    material,
	})
}

// CreateReceipt handles POST /api/v1/inventory/receipts.
func (h *Handler) CreateReceipt(c *gin.Context) {
	actor := getActor(c)
	var req CreateReceiptRequest
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

	receipt, err := h.svc.CreateReceipt(c.Request.Context(), actor.RoleCode, actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Bahan masuk berhasil dicatat dengan tag Client ID dan Order ID.",
		"data":    receipt,
	})
}

// GetReceipt handles GET /api/v1/inventory/receipts/:id.
func (h *Handler) GetReceipt(c *gin.Context) {
	id := c.Param("id")
	receipt, err := h.svc.GetReceipt(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    receipt,
	})
}

// ListReceipts handles GET /api/v1/inventory/receipts.
func (h *Handler) ListReceipts(c *gin.Context) {
	clientID := c.Query("clientId")
	orderID := c.Query("orderId")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListReceiptsFilter{
		ClientID:  clientID,
		OrderID:   orderID,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	}

	receipts, total, err := h.svc.ListReceipts(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    receipts,
	})
}

// ListStocks handles GET /api/v1/inventory/stocks.
func (h *Handler) ListStocks(c *gin.Context) {
	actor := getActor(c)
	clientID := c.Query("clientId")
	materialID := c.Query("materialId")
	orderID := c.Query("orderId")

	filter := ListStocksFilter{
		ClientID:   clientID,
		MaterialID: materialID,
		OrderID:    orderID,
	}

	stocks, err := h.svc.ListStocks(c.Request.Context(), actor.RoleCode, actor.ClientID, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(stocks),
		"data":    stocks,
	})
}

// GetLot handles GET /api/v1/inventory/lots/:id.
func (h *Handler) GetLot(c *gin.Context) {
	id := c.Param("id")
	lot, err := h.svc.GetLot(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    lot,
	})
}

// ListIssues handles GET /api/v1/inventory/issues.
func (h *Handler) ListIssues(c *gin.Context) {
	orderID := c.Query("orderId")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListIssuesFilter{
		OrderID: orderID,
		Limit:   limit,
		Offset:  offset,
	}

	issues, total, err := h.svc.ListIssues(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    issues,
	})
}

// CreateIssue handles POST /api/v1/inventory/issues (FR-6.2 & Skenario Uji 1).
func (h *Handler) CreateIssue(c *gin.Context) {
	actor := getActor(c)

	var req CreateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	detail, err := h.svc.CreateIssue(c.Request.Context(), actor.RoleCode, actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Bahan berhasil dikeluarkan ke bagian cutting untuk order " + detail.TargetOrderCode,
		"data":    detail,
	})
}

// GetIssue handles GET /api/v1/inventory/issues/:id.
func (h *Handler) GetIssue(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.GetIssue(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ListBlockedAttempts handles GET /api/v1/inventory/blocked-attempts.
func (h *Handler) ListBlockedAttempts(c *gin.Context) {
	actor := getActor(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListBlockedAttemptsFilter{
		Limit:  limit,
		Offset: offset,
	}

	records, total, err := h.svc.ListBlockedAttempts(c.Request.Context(), actor.RoleCode, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    records,
	})
}

// CreateScrap handles POST /api/v1/inventory/scraps.
func (h *Handler) CreateScrap(c *gin.Context) {
	actor := getActor(c)

	var req CreateScrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	record, err := h.svc.CreateScrap(c.Request.Context(), actor.RoleCode, actor.ID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Data sisa potongan cutting berhasil dicatat",
		"data":    record,
	})
}

// ListScraps handles GET /api/v1/inventory/scraps.
func (h *Handler) ListScraps(c *gin.Context) {
	orderID := c.Query("orderId")
	category := c.Query("scrapCategory")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListScrapsFilter{
		OrderID:       orderID,
		ScrapCategory: category,
		Limit:         limit,
		Offset:        offset,
	}

	scraps, total, err := h.svc.ListScraps(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    scraps,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrMaterialNotFound),
		errors.Is(err, ErrReceiptNotFound),
		errors.Is(err, ErrLotNotFound),
		errors.Is(err, ErrClientNotFound),
		errors.Is(err, ErrOrderNotFound),
		errors.Is(err, ErrIssueNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrForbiddenInventory):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrMaterialCodeExists):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MATERIAL_CODE_EXISTS",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrCrossContaminationBlocked),
		errors.Is(err, ErrClientOrderMismatch):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVENTORY_CLIENT_ORDER_MISMATCH",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInsufficientStock):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INSUFFICIENT_STOCK",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrLotNotAvailable):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "LOT_NOT_AVAILABLE",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrReceiptItemsEmpty),
		errors.Is(err, ErrIssueItemsEmpty),
		errors.Is(err, ErrInvalidQuantity),
		errors.Is(err, ErrMissingRequiredField),
		errors.Is(err, ErrScrapCategoryInvalid),
		errors.Is(err, ErrScrapDispositionInvalid):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
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

