// Package authn provides session-based authentication middleware and helpers.
//
// Authentication flow:
//  1. Frontend sends username/email + password over HTTPS.
//  2. Backend verifies password hash, account status, and permissions.
//  3. Backend creates a new session: generates a new session ID, stores the session record,
//     and sends an HttpOnly cookie to the browser.
//  4. Every subsequent API request is validated: session is active, user is active,
//     permission is granted, and tenant scope is correct.
package authn

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const sessionCookieName = "lukla_session"

// Claims carries the authenticated identity extracted from the session.
type Claims struct {
	UserID   string
	TenantID string
	Roles    []string
}

// Middleware validates the session cookie on every request.
// On success it stores Claims in the Gin context under the key "claims".
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: read session cookie, look up session in database, validate expiry/status,
		// load user + roles, set claims in context.
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil || cookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHENTICATED",
					"message": "Sesi tidak valid atau sudah berakhir.",
				},
			})
			return
		}

		// Placeholder — replace with real session lookup.
		_ = cookie
		c.Next()
	}
}

// ClaimsFrom retrieves Claims stored by the authn middleware.
func ClaimsFrom(c *gin.Context) (Claims, bool) {
	v, ok := c.Get("claims")
	if !ok {
		return Claims{}, false
	}
	claims, ok := v.(Claims)
	return claims, ok
}
