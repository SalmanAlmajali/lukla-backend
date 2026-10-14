package ratecard

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
	"github.com/lukla/operations-engine/internal/platform/httpx"
)

// Handler exposes HTTP controllers for rate cards and pricing audit trails.
type Handler struct {
	svc *Service
}

// NewHandler instantiates a new ratecard HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers rate card endpoints under the provided router group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	rateCards := r.Group("/rate-cards", authMiddleware)
	{
		rateCards.GET("", h.ListRateCards)
		rateCards.POST("", h.CreateRateCard)
		rateCards.GET("/:id", h.GetRateCardDetail)
		rateCards.PUT("/:id", h.UpdateRateCard)
		rateCards.GET("/:id/history", h.ListRateCardHistories)
	}
}

// ListRateCards handles GET /rate-cards with optional clientId and articleId filters.
func (h *Handler) ListRateCards(c *gin.Context) {
	var filter ListRateCardsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ValidationError(c, "Parameter filter rate card tidak valid.", err.Error())
		return
	}
	if filter.ClientID == "" {
		filter.ClientID = c.Query("client_id")
	}
	if filter.ArticleID == "" {
		filter.ArticleID = c.Query("article_id")
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	rateCards, err := h.svc.ListRateCards(c.Request.Context(), filter, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrForbiddenRateCardAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, rateCards)
}

// GetRateCardDetail handles GET /rate-cards/:id.
func (h *Handler) GetRateCardDetail(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID rate card diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	rc, err := h.svc.GetRateCardDetail(c.Request.Context(), id, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrRateCardNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Rate card tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenRateCardAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, rc)
}

// CreateRateCard handles POST /rate-cards.
func (h *Handler) CreateRateCard(c *gin.Context) {
	role, _ := authn.RoleFrom(c)
	if !canManageRateCards(role) {
		httpx.Forbidden(c, "FORBIDDEN", ErrForbiddenRateCardManage.Error())
		return
	}

	var req CreateRateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ValidationError(c, "Data permintaan rate card tidak valid.", err.Error())
		return
	}

	var requesterUserID *string
	if uid, ok := authn.UserIDFrom(c); ok && uid != "" {
		requesterUserID = &uid
	}

	rc, err := h.svc.CreateRateCard(c.Request.Context(), req, role, requesterUserID)
	if err != nil {
		if errors.Is(err, ErrForbiddenRateCardManage) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidRateValue) || errors.Is(err, ErrComponentMismatch) {
			httpx.BadRequest(c, "BAD_REQUEST", err.Error())
			return
		}
		if errors.Is(err, ErrClientNotFound) || errors.Is(err, ErrArticleNotFound) {
			httpx.NotFound(c, "NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrDuplicateRateCard) {
			httpx.Conflict(c, "DUPLICATE_RATE_CARD", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.Created(c, rc)
}

// UpdateRateCard handles PUT /rate-cards/:id.
func (h *Handler) UpdateRateCard(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID rate card diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	if !canManageRateCards(role) {
		httpx.Forbidden(c, "FORBIDDEN", ErrForbiddenRateCardManage.Error())
		return
	}

	var req UpdateRateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ValidationError(c, "Data pembaruan tarif tidak valid.", err.Error())
		return
	}

	var requesterUserID *string
	if uid, ok := authn.UserIDFrom(c); ok && uid != "" {
		requesterUserID = &uid
	}

	rc, err := h.svc.UpdateRateCard(c.Request.Context(), id, req, role, requesterUserID)
	if err != nil {
		if errors.Is(err, ErrForbiddenRateCardManage) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}
		if errors.Is(err, ErrRateCardNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Rate card tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrInvalidRateValue) || errors.Is(err, ErrChangeReasonRequired) {
			httpx.BadRequest(c, "BAD_REQUEST", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, rc)
}

// ListRateCardHistories handles GET /rate-cards/:id/history.
func (h *Handler) ListRateCardHistories(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httpx.BadRequest(c, "INVALID_PARAMETER", "ID rate card diperlukan.")
		return
	}

	role, _ := authn.RoleFrom(c)
	var requesterClientID *string
	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		requesterClientID = &cid
	}

	histories, err := h.svc.ListRateCardHistories(c.Request.Context(), id, role, requesterClientID)
	if err != nil {
		if errors.Is(err, ErrRateCardNotFound) {
			httpx.NotFound(c, "NOT_FOUND", "Rate card tidak ditemukan.")
			return
		}
		if errors.Is(err, ErrForbiddenRateCardAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, histories)
}

