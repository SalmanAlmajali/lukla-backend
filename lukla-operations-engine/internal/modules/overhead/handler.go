package overhead

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler menangani endpoint HTTP untuk modul overhead pabrik (BPMN 09).
type Handler struct {
	svc *Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan seluruh endpoint overhead dan laporan COGS ke router Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, internalAuthz gin.HandlerFunc) {
	overhead := rg.Group("/overhead")
	overhead.Use(authnMiddleware, internalAuthz)
	{
		overhead.GET("/categories", h.ListCategories)
		overhead.POST("/categories", h.CreateCategory)

		overhead.GET("/periods", h.ListPeriods)
		overhead.POST("/periods", h.CreatePeriod)
		overhead.GET("/periods/:id", h.GetPeriodByID)
		overhead.POST("/periods/:id/expenses", h.RecordExpense)
		overhead.DELETE("/periods/:id/expenses/:expenseId", h.DeleteExpense)

		overhead.POST("/periods/:id/calculate", h.CalculateOverhead)
		overhead.POST("/periods/:id/lock", h.LockPeriod)
		overhead.GET("/periods/:id/summary", h.GetPeriodSummaryReport)
	}

	reports := rg.Group("/reports")
	reports.Use(authnMiddleware, internalAuthz)
	{
		reports.GET("/cogs/:orderId", h.GetOrderCOGSReport)
	}
}

// ListCategories handles GET /api/v1/overhead/categories.
func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    categories,
	})
}

// CreateCategory handles POST /api/v1/overhead/categories.
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateOverheadCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload kategori overhead tidak valid: " + err.Error(),
			},
		})
		return
	}

	cat, err := h.svc.CreateCategory(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Kategori overhead berhasil ditambahkan",
		"data":    cat,
	})
}

// ListPeriods handles GET /api/v1/overhead/periods.
func (h *Handler) ListPeriods(c *gin.Context) {
	var yearPtr *int
	if yStr := c.Query("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil {
			yearPtr = &y
		}
	}

	var statusPtr *string
	if st := c.Query("status"); st != "" {
		statusPtr = &st
	}

	periods, err := h.svc.ListPeriods(c.Request.Context(), yearPtr, statusPtr)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    periods,
	})
}

// CreatePeriod handles POST /api/v1/overhead/periods.
func (h *Handler) CreatePeriod(c *gin.Context) {
	var req CreateOverheadPeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload pembukaan periode tidak valid: " + err.Error(),
			},
		})
		return
	}

	period, err := h.svc.CreatePeriod(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Periode overhead bulanan berhasil dibuka",
		"data":    period,
	})
}

// GetPeriodByID handles GET /api/v1/overhead/periods/:id.
func (h *Handler) GetPeriodByID(c *gin.Context) {
	id := c.Param("id")
	period, err := h.svc.GetPeriodByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    period,
	})
}

// RecordExpense handles POST /api/v1/overhead/periods/:id/expenses.
func (h *Handler) RecordExpense(c *gin.Context) {
	id := c.Param("id")
	var req RecordOverheadExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload pengeluaran overhead tidak valid: " + err.Error(),
			},
		})
		return
	}

	item, err := h.svc.RecordExpense(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Pengeluaran overhead berhasil disimpan",
		"data":    item,
	})
}

// DeleteExpense handles DELETE /api/v1/overhead/periods/:id/expenses/:expenseId.
func (h *Handler) DeleteExpense(c *gin.Context) {
	id := c.Param("id")
	expenseID := c.Param("expenseId")

	if err := h.svc.DeleteExpense(c.Request.Context(), id, expenseID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Item pengeluaran berhasil dihapus",
	})
}

// CalculateOverhead handles POST /api/v1/overhead/periods/:id/calculate.
func (h *Handler) CalculateOverhead(c *gin.Context) {
	id := c.Param("id")
	var req CalculateOverheadRequest
	_ = c.ShouldBindJSON(&req)

	result, err := h.svc.CalculateOverhead(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kalkulasi alokasi overhead per unit berhasil dihitung",
		"data":    result,
	})
}

// LockPeriod handles POST /api/v1/overhead/periods/:id/lock.
func (h *Handler) LockPeriod(c *gin.Context) {
	id := c.Param("id")
	userID, _ := authn.UserIDFrom(c)

	detail, err := h.svc.LockPeriod(c.Request.Context(), id, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Periode overhead bulanan berhasil dikunci",
		"data":    detail,
	})
}

// GetPeriodSummaryReport handles GET /api/v1/overhead/periods/:id/summary.
func (h *Handler) GetPeriodSummaryReport(c *gin.Context) {
	id := c.Param("id")
	report, err := h.svc.GetPeriodSummaryReport(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

// GetOrderCOGSReport handles GET /api/v1/reports/cogs/:orderId.
func (h *Handler) GetOrderCOGSReport(c *gin.Context) {
	orderID := c.Param("orderId")
	report, err := h.svc.GetOrderCOGSReport(c.Request.Context(), orderID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    report,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, ErrPeriodNotFound),
		errors.Is(err, ErrExpenseNotFound),
		errors.Is(err, ErrOrderNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrCategoryCodeExists),
		errors.Is(err, ErrPeriodAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CONFLICT",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrPeriodAlreadyLocked):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERIOD_LOCKED",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInvalidMonth),
		errors.Is(err, ErrInvalidYear),
		errors.Is(err, ErrInvalidAmount),
		errors.Is(err, ErrInvalidWeights):
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

