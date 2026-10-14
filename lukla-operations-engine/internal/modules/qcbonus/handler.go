package qcbonus

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler handles HTTP requests for QC pool bonus endpoints (BPMN 08).
type Handler struct {
	svc *Service
}

// NewHandler creates a new qcbonus Handler instance.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers QC bonus endpoints on the Gin router.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	qcBonus := rg.Group("/qc-bonus")
	qcBonus.Use(authnMiddleware)
	{
		// Config (Owner only)
		qcBonus.GET("/config", h.GetBonusConfig)
		qcBonus.PUT("/config", h.UpdateBonusRate)

		// Recipients (Owner only)
		qcBonus.GET("/recipients", h.ListRecipients)
		qcBonus.PUT("/recipients", h.UpdateRecipients)

		// Periods & Calculation (Skenario Uji 3)
		qcBonus.GET("/periods", h.ListPeriods)
		qcBonus.POST("/periods", h.CreatePeriod)
		qcBonus.GET("/periods/:id", h.GetPeriodByID)
		qcBonus.POST("/periods/:id/calculate", h.CalculatePeriod)
		qcBonus.POST("/periods/:id/lock", h.LockPeriod)
	}
}

// GetBonusConfig handles GET /api/v1/qc-bonus/config.
func (h *Handler) GetBonusConfig(c *gin.Context) {
	actor := getActor(c)
	cfg, err := h.svc.GetBonusConfig(c.Request.Context(), actor)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cfg,
	})
}

// UpdateBonusRate handles PUT /api/v1/qc-bonus/config.
func (h *Handler) UpdateBonusRate(c *gin.Context) {
	actor := getActor(c)
	var req UpdateQCBonusRateRequest
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

	cfg, err := h.svc.UpdateBonusRate(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Tarif bonus QC berhasil diperbarui menjadi Rp %.0f per pcs", req.RatePerPcs),
		"data":    cfg,
	})
}

// ListRecipients handles GET /api/v1/qc-bonus/recipients.
func (h *Handler) ListRecipients(c *gin.Context) {
	actor := getActor(c)
	recipients, err := h.svc.ListRecipients(c.Request.Context(), actor)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    recipients,
	})
}

// UpdateRecipients handles PUT /api/v1/qc-bonus/recipients.
func (h *Handler) UpdateRecipients(c *gin.Context) {
	actor := getActor(c)
	var req UpdateQCBonusRecipientsRequest
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

	recipients, err := h.svc.UpdateRecipients(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Daftar penerima pool bonus berhasil diperbarui",
		"data":    recipients,
	})
}

// ListPeriods handles GET /api/v1/qc-bonus/periods.
func (h *Handler) ListPeriods(c *gin.Context) {
	actor := getActor(c)
	status := c.Query("status")
	summaries, err := h.svc.ListPeriods(c.Request.Context(), actor, status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summaries,
	})
}

// CreatePeriod handles POST /api/v1/qc-bonus/periods.
func (h *Handler) CreatePeriod(c *gin.Context) {
	actor := getActor(c)
	var req CreateQCBonusPeriodRequest
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

	detail, err := h.svc.CreatePeriod(c.Request.Context(), actor, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetPeriodByID handles GET /api/v1/qc-bonus/periods/:id.
func (h *Handler) GetPeriodByID(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")

	detail, err := h.svc.GetPeriodByID(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// CalculatePeriod handles POST /api/v1/qc-bonus/periods/:id/calculate (Trigger Skenario Uji 3).
func (h *Handler) CalculatePeriod(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")

	detail, err := h.svc.CalculatePeriod(c.Request.Context(), actor, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	message := fmt.Sprintf(
		"Kalkulasi pool bonus berhasil: %d pcs lolos x Rp %.0f = Rp %.0f, dibagi rata ke %d orang (Rp %.0f/orang)",
		detail.TotalPassedPcs,
		detail.RatePerPcsApplied,
		detail.TotalPoolAmount,
		detail.RecipientCount,
		detail.AmountPerRecipient,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    detail,
	})
}

// LockPeriod handles POST /api/v1/qc-bonus/periods/:id/lock.
func (h *Handler) LockPeriod(c *gin.Context) {
	actor := getActor(c)
	id := c.Param("id")
	var req LockQCBonusPeriodRequest
	_ = c.ShouldBindJSON(&req)

	detail, err := h.svc.LockPeriod(c.Request.Context(), actor, id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Periode bonus QC berhasil dikunci",
		"data":    detail,
	})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrConfigNotFound),
		errors.Is(err, ErrPeriodNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrPeriodAlreadyLocked):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "PERIOD_LOCKED",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrForbiddenOwnerOnly),
		errors.Is(err, ErrForbiddenQualityBonus):
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": err.Error(),
			},
		})
	case errors.Is(err, ErrPeriodDatesInvalid),
		errors.Is(err, ErrNoActiveRecipients),
		errors.Is(err, ErrInvalidBonusRate):
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

