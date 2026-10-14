package payroll

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
	payrollGroup := rg.Group("/payroll")
	payrollGroup.Use(authnMiddleware)
	{
		periods := payrollGroup.Group("/periods")
		{
			periods.GET("", h.ListPeriods)
			periods.POST("", h.CreatePeriod)
			periods.GET("/:id", h.GetPeriod)
			periods.POST("/:id/calculate", h.CalculatePayroll)
			periods.GET("/:id/slips", h.ListSlips)
		}

		slips := payrollGroup.Group("/slips")
		{
			slips.GET("/:id", h.GetSlipDetail)
		}
	}
}

// ListPeriods handles GET /api/v1/payroll/periods.
func (h *Handler) ListPeriods(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListPeriodsFilter{
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	periods, total, err := h.svc.ListPeriods(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    periods,
	})
}

// CreatePeriod handles POST /api/v1/payroll/periods.
func (h *Handler) CreatePeriod(c *gin.Context) {
	actor := getActor(c)
	var req CreatePayrollPeriodRequest
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

	period, err := h.svc.CreatePeriod(c.Request.Context(), actor.RoleCode, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    period,
	})
}

// GetPeriod handles GET /api/v1/payroll/periods/:id.
func (h *Handler) GetPeriod(c *gin.Context) {
	id := c.Param("id")
	period, err := h.svc.GetPeriod(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    period,
	})
}

// CalculatePayroll handles POST /api/v1/payroll/periods/:id/calculate.
func (h *Handler) CalculatePayroll(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")

	var req CalculatePayrollRequest
	_ = c.ShouldBindJSON(&req) // optional request body

	period, err := h.svc.CalculatePayroll(c.Request.Context(), actor.RoleCode, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kalkulasi payroll berhasil dijalankan untuk periode " + period.PeriodCode,
		"data":    period,
	})
}

// ListSlips handles GET /api/v1/payroll/periods/:id/slips.
func (h *Handler) ListSlips(c *gin.Context) {
	id := c.Param("id")
	operatorID := c.Query("operatorId")
	search := c.Query("search")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ListSlipsFilter{
		OperatorID: operatorID,
		Search:     search,
		Limit:      limit,
		Offset:     offset,
	}

	slips, total, err := h.svc.ListSlips(c.Request.Context(), id, filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		"data":    slips,
	})
}

// GetSlipDetail handles GET /api/v1/payroll/slips/:id.
func (h *Handler) GetSlipDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.GetSlipDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPeriodNotFound), errors.Is(err, ErrSlipNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrForbiddenPayroll):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrPeriodCodeExists):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERIOD_CODE_EXISTS",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrPeriodAlreadyLocked):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PAYROLL_PERIOD_LOCKED",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrInvalidMealAllowance), errors.Is(err, ErrInvalidPeriodDates):
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

