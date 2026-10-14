package identity

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
	"github.com/lukla/operations-engine/internal/platform/httpx"
)

// Handler handles HTTP requests for the identity module.
type Handler struct {
	svc *Service
}

// NewHandler constructs an identity Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ValidationError(c, "Email dan password wajib diisi dengan format email yang valid.", err.Error())
		return
	}

	data, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.Unauthorized(c, "UNAUTHORIZED", err.Error())
			return
		}
		if errors.Is(err, ErrAccountInactive) || errors.Is(err, ErrNoAppAccess) {
			httpx.Forbidden(c, "FORBIDDEN", err.Error())
			return
		}
		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OKWithMessage(c, "Login berhasil", data)
}

// Me handles GET /auth/me for the authenticated user.
func (h *Handler) Me(c *gin.Context) {
	userID, ok := authn.UserIDFrom(c)
	if !ok || userID == "" {
		httpx.Unauthorized(c, "UNAUTHORIZED", "Pengguna tidak terautentikasi.")
		return
	}

	profile, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrAccountInactive) {
			httpx.Unauthorized(c, "UNAUTHORIZED", err.Error())
			return
		}
		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)
		httpx.InternalServerError(c, rid)
		return
	}

	httpx.OK(c, profile)
}

// RegisterRoutes registers identity module routes on the given router group.
func RegisterRoutes(r *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	r.POST("/auth/login", h.Login)
	r.GET("/auth/me", authnMiddleware, h.Me)
}
