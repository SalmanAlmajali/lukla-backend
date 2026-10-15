// Package httpx provides HTTP response helpers and middleware.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware that handles Cross-Origin Resource Sharing (CORS).
// During development, it permits requests from any origin, supports cookies/authorization headers,
// reflects requested headers, and handles preflight OPTIONS requests with HTTP 204 No Content.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// When Access-Control-Allow-Credentials is true, browsers require the exact origin rather than wildcard "*"
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")

		reqHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
		if reqHeaders != "" {
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Content-Length, Accept, Origin, Cache-Control, X-Requested-With, Accept-Encoding, X-CSRF-Token")
		}

		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

		// Handle preflight OPTIONS request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

