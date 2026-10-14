// Package authn provides JWT-based authentication middleware and helpers.
package authn

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims carries the authenticated identity extracted from the token.
type Claims struct {
	UserID   string
	TenantID string
	Roles    []string
}

// Middleware returns a Gin middleware that validates the JWT Bearer token in the Authorization header.
func Middleware(tokenSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString = strings.TrimSpace(parts[1])
			}
		}

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Header Authorization Bearer token diperlukan.",
				},
			})
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(tokenSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Token tidak valid atau sudah kedaluwarsa.",
				},
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Klaim token tidak valid.",
				},
			})
			return
		}

		sub, err := claims.GetSubject()
		if err != nil || sub == "" {
			// fallback check direct map key
			if subVal, exists := claims["sub"].(string); exists && subVal != "" {
				sub = subVal
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Subject (user ID) pada token tidak ditemukan.",
					},
				})
				return
			}
		}

		c.Set("user_id", sub)
		c.Set("claims", Claims{UserID: sub})

		c.Next()
	}
}

// UserIDFrom retrieves the authenticated user ID from context.
func UserIDFrom(c *gin.Context) (string, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
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

// ErrUnauthenticated is returned when no valid credentials exist.
var ErrUnauthenticated = errors.New("pengguna tidak terautentikasi")
