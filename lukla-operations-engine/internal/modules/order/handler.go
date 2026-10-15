package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP untuk manajemen pesanan internal pabrik.
type Handler struct {
	svc *Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan rute internal pesanan ke router Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, internalAuthz gin.HandlerFunc) {
	ordersGroup := rg.Group("/orders", authnMiddleware, internalAuthz)
	{
		ordersGroup.GET("", h.ListOrders)
		ordersGroup.GET("/:id", h.GetOrderDetail)
		ordersGroup.POST("/:id/complete", h.CompleteOrder)
	}
}

// ListOrders handles GET /api/v1/orders.
func (h *Handler) ListOrders(c *gin.Context) {
	clientID := c.Query("clientId")
	status := c.Query("status")

	list, err := h.svc.ListOrders(c.Request.Context(), clientID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
	})
}

// GetOrderDetail handles GET /api/v1/orders/:id.
func (h *Handler) GetOrderDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.GetOrderDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ORDER_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// CompleteOrder handles POST /api/v1/orders/:id/complete.
func (h *Handler) CompleteOrder(c *gin.Context) {
	id := c.Param("id")

	var req CompleteOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// body opsional, abaikan jika kosong
	}

	detail, err := h.svc.CompleteOrder(c.Request.Context(), id, req.Notes)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ORDER_NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		if errors.Is(err, ErrOrderAlreadyCompleted) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ORDER_ALREADY_COMPLETED",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pesanan berhasil ditandai selesai dan notifikasi telah dikirim ke klien",
		"data":    detail,
	})
}

