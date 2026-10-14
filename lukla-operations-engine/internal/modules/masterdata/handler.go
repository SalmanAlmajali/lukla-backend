package masterdata

import (
	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/httpx"
)

// Handler handles HTTP requests for master data entities.
type Handler struct {
	svc *Service
}

// NewHandler constructs a masterdata Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListClients handles GET /clients with pagination and search.
func (h *Handler) ListClients(c *gin.Context) {
	var filter ListClientsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ValidationError(c, "Parameter pagination atau search tidak valid.", err.Error())
		return
	}

	clients, meta, err := h.svc.ListClients(c.Request.Context(), filter)
	if err != nil {
		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.Paginated(c, clients, meta)
}

// RegisterRoutes registers masterdata routes on the given router group.
func RegisterRoutes(r *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, authzMiddleware gin.HandlerFunc) {
	r.GET("/clients", authnMiddleware, authzMiddleware, h.ListClients)
}
