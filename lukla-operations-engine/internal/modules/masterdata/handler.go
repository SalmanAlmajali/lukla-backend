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

// ListArticles handles GET /articles with category and clientId filters and pagination.
func (h *Handler) ListArticles(c *gin.Context) {
	var filter ListArticlesFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ValidationError(c, "Parameter filter artikel tidak valid.", err.Error())
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	articles, meta, err := h.svc.ListArticles(c.Request.Context(), filter, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrInvalidCategory) {
			httpx.BadRequest(c, "INVALID_CATEGORY", err.Error())
			return
		}
		if errors.Is(err, ErrForbiddenArticleAccess) || errors.Is(err, ErrForbiddenGeneralAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.Paginated(c, articles, meta)
}

// GetArticleDetail handles GET /articles/:id.
func (h *Handler) GetArticleDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID artikel diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	detail, err := h.svc.GetArticleDetail(c.Request.Context(), id, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Artikel tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenArticleAccess) {
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

// CreateArticle handles POST /articles.
func (h *Handler) CreateArticle(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ValidationError(c, "Payload artikel tidak valid.", err.Error())
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	article, err := h.svc.CreateArticle(c.Request.Context(), req, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			httpx.BadRequest(c, "CLIENT_NOT_FOUND", "Klien tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrDuplicateArticleCode) {
			httpx.BadRequest(c, "DUPLICATE_CODE", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidCategory) {
			httpx.BadRequest(c, "INVALID_CATEGORY", err.Error())
			return
		}
		if errors.Is(err, ErrForbiddenArticleAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.Created(c, article)
}

// ListComponents handles GET /articles/:id/components.
func (h *Handler) ListComponents(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID artikel diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	components, err := h.svc.ListComponents(c.Request.Context(), id, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Artikel tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenArticleAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, components)
}

// CreateComponent handles POST /articles/:id/components.
func (h *Handler) CreateComponent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID artikel diperlukan.")
		return
	}

	var req CreateComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ValidationError(c, "Payload komponen tidak valid.", err.Error())
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	component, err := h.svc.CreateComponent(c.Request.Context(), id, req, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Artikel tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenArticleAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.Created(c, component)
}

// ListOperators handles GET /operators with status, line, and search filters.
// Hanya dapat diakses oleh staf internal (internal staff).
func (h *Handler) ListOperators(c *gin.Context) {
	var filter ListOperatorsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ValidationError(c, "Parameter filter operator tidak valid.", err.Error())
		return
	}

	role, _ := authn.RoleFrom(c)

	operators, err := h.svc.ListOperators(c.Request.Context(), filter, role)
	if err != nil {
		if errors.Is(err, ErrInvalidOperatorStatus) {
			httpx.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		if errors.Is(err, ErrForbiddenOperatorAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, operators)
}

// RegisterRoutes registers masterdata routes on the given router group.
func RegisterRoutes(r *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, authzMiddleware gin.HandlerFunc) {
	r.GET("/clients", authnMiddleware, authzMiddleware, h.ListClients)
	r.GET("/clients/:id", authnMiddleware, h.GetClientDetail)

	r.GET("/articles", authnMiddleware, h.ListArticles)
	r.POST("/articles", authnMiddleware, h.CreateArticle)
	r.GET("/articles/:id", authnMiddleware, h.GetArticleDetail)

	r.GET("/articles/:id/components", authnMiddleware, h.ListComponents)
	r.POST("/articles/:id/components", authnMiddleware, h.CreateComponent)

	r.GET("/operators", authnMiddleware, authzMiddleware, h.ListOperators)
}


