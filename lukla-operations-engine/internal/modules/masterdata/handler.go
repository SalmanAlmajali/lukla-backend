package masterdata

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
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

// GetClientDetail handles GET /clients/:id.
func (h *Handler) GetClientDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID klien diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	detail, err := h.svc.GetClientDetail(c.Request.Context(), id, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Klien tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenClientAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, detail)
}

// RegisterRoutes registers masterdata routes on the given router group.
func RegisterRoutes(r *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, authzMiddleware gin.HandlerFunc) {
	r.GET("/clients", authnMiddleware, authzMiddleware, h.ListClients)
	r.GET("/clients/:id", authnMiddleware, h.GetClientDetail)
}

