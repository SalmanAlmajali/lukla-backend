package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard error envelope returned by all API endpoints.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains a stable machine-readable code and a safe user-facing message.
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// BadRequest writes a 400 response.
func BadRequest(c *gin.Context, code, message string) {
	writeError(c, http.StatusBadRequest, code, message)
}

// Unauthorized writes a 401 response.
func Unauthorized(c *gin.Context, code, message string) {
	writeError(c, http.StatusUnauthorized, code, message)
}

// Forbidden writes a 403 response.
func Forbidden(c *gin.Context, code, message string) {
	writeError(c, http.StatusForbidden, code, message)
}

// NotFound writes a 404 response.
func NotFound(c *gin.Context, code, message string) {
	writeError(c, http.StatusNotFound, code, message)
}

// Conflict writes a 409 response.
func Conflict(c *gin.Context, code, message string) {
	writeError(c, http.StatusConflict, code, message)
}

// InternalServerError writes a 500 response. The technical detail is NOT forwarded to the client.
func InternalServerError(c *gin.Context, requestID string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorDetail{
			Code:      "INTERNAL_ERROR",
			Message:   "Terjadi kesalahan internal. Silakan coba lagi atau hubungi administrator.",
			RequestID: requestID,
		},
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	rid, _ := requestID.(string)

	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: rid,
		},
	})
}
