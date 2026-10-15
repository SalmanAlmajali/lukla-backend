package notification

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler menangani request HTTP untuk modul notifikasi.
type Handler struct {
	svc *Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan endpoint notifikasi ke router Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, internalAuthz gin.HandlerFunc) {
	// 1. Rute Portal Klien (membutuhkan authn dan identitas client_id)
	portalGroup := rg.Group("/portal", authnMiddleware)
	{
		portalGroup.GET("/notifications", h.ListNotifications)
		portalGroup.PUT("/notifications/:id/read", h.MarkRead)
		portalGroup.PUT("/notifications/read-all", h.MarkReadAll)
	}

	// 2. Rute Internal Staf Pabrik (membutuhkan authn dan hak akses internal)
	internalGroup := rg.Group("/notifications", authnMiddleware, internalAuthz)
	{
		internalGroup.POST("/send", h.SendNotification)
	}
}

// resolveClientID mengekstrak client_id dari context pengguna portal.
func (h *Handler) resolveClientID(c *gin.Context) (string, error) {
	role, _ := authn.RoleFrom(c)
	clientID, ok := authn.ClientIDFrom(c)

	if role != "client" || !ok || clientID == "" {
		return "", ErrForbiddenClientPortal
	}

	return clientID, nil
}

// ListNotifications handles GET /api/v1/portal/notifications.
func (h *Handler) ListNotifications(c *gin.Context) {
	clientID, err := h.resolveClientID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN_CLIENT_PORTAL",
				"message": err.Error(),
			},
		})
		return
	}

	unreadOnly, _ := strconv.ParseBool(c.DefaultQuery("unreadOnly", "false"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	list, unreadCount, err := h.svc.ListClientNotifications(c.Request.Context(), clientID, unreadOnly, limit)
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
		"success":     true,
		"unreadCount": unreadCount,
		"data":        list,
	})
}

// MarkRead handles PUT /api/v1/portal/notifications/:id/read.
func (h *Handler) MarkRead(c *gin.Context) {
	clientID, err := h.resolveClientID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN_CLIENT_PORTAL",
				"message": err.Error(),
			},
		})
		return
	}

	id := c.Param("id")
	if err := h.svc.MarkAsRead(c.Request.Context(), clientID, id); err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOTIFICATION_NOT_FOUND",
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
		"message": "Notifikasi telah ditandai sebagai dibaca",
	})
}

// MarkReadAll handles PUT /api/v1/portal/notifications/read-all.
func (h *Handler) MarkReadAll(c *gin.Context) {
	clientID, err := h.resolveClientID(c)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN_CLIENT_PORTAL",
				"message": err.Error(),
			},
		})
		return
	}

	if err := h.svc.MarkAllAsRead(c.Request.Context(), clientID); err != nil {
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
		"message": "Seluruh notifikasi berhasil ditandai sebagai dibaca",
	})
}

// SendNotification handles POST /api/v1/notifications/send.
func (h *Handler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
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

	notif, err := h.svc.SendNotification(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrMissingRequiredField) || errors.Is(err, ErrInvalidEventType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
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

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Notifikasi berhasil dikirim",
		"data":    notif,
	})
}

