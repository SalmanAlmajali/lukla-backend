// Package authn provides JWT-based authentication middleware and helpers.
package authn

import (
	"context"
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

// UserIdentity represents minimal user account details loaded for request context.
type UserIdentity struct {
	ID       string
	RoleCode string
	ClientID *string
	IsActive bool
}

// UserLoader is a function to load a UserIdentity by their ID.
type UserLoader func(ctx context.Context, userID string) (*UserIdentity, error)

// Middleware returns a Gin middleware that validates the JWT Bearer token in the Authorization header.
// Optional loaders allow resolving the user's role and client_id from database into the context.
func Middleware(tokenSecret string, loaders ...UserLoader) gin.HandlerFunc {
	var loader UserLoader
	if len(loaders) > 0 {
		loader = loaders[0]
	}

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

		if loader != nil {
			u, err := loader(c.Request.Context(), sub)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "Gagal memverifikasi identitas pengguna.",
					},
				})
				return
			}
			if u == nil || !u.IsActive {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Pengguna tidak ditemukan atau sudah dinonaktifkan.",
					},
				})
				return
			}

			c.Set("role", u.RoleCode)
			if u.ClientID != nil {
				c.Set("client_id", *u.ClientID)
			}
			c.Set("claims", Claims{
				UserID: u.ID,
				Roles:  []string{u.RoleCode},
				TenantID: func() string {
					if u.ClientID != nil {
						return *u.ClientID
					}
					return ""
				}(),
			})
		}

		c.Next()
	}
}

// OptionalMiddleware attempts to extract identity from the JWT Bearer token if provided.
// If the token is missing, expired, or invalid, it gracefully allows the request to continue
// without aborting, allowing endpoints with flexible access (like AI Copilot) to operate safely.
func OptionalMiddleware(tokenSecret string, loaders ...UserLoader) gin.HandlerFunc {
	var loader UserLoader
	if len(loaders) > 0 {
		loader = loaders[0]
	}

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
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(tokenSecret), nil
		})

		if err != nil || !token.Valid {
			c.Next()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Next()
			return
		}

		sub, err := claims.GetSubject()
		if err != nil || sub == "" {
			if subVal, exists := claims["sub"].(string); exists && subVal != "" {
				sub = subVal
			}
		}

		if sub == "" {
			c.Next()
			return
		}

		c.Set("user_id", sub)
		c.Set("claims", Claims{UserID: sub})

		if loader != nil {
			u, err := loader(c.Request.Context(), sub)
			if err == nil && u != nil && u.IsActive {
				c.Set("role", u.RoleCode)
				if u.ClientID != nil {
					c.Set("client_id", *u.ClientID)
				}
				c.Set("claims", Claims{
					UserID: u.ID,
					Roles:  []string{u.RoleCode},
					TenantID: func() string {
						if u.ClientID != nil {
							return *u.ClientID
						}
						return ""
					}(),
				})
			}
		}

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

// RoleFrom retrieves the authenticated user's role code from context.
func RoleFrom(c *gin.Context) (string, bool) {
	v, ok := c.Get("role")
	if !ok {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}

// ClientIDFrom retrieves the authenticated user's client ID from context.
func ClientIDFrom(c *gin.Context) (string, bool) {
	v, ok := c.Get("client_id")
	if !ok {
		return "", false
	}
	cid, ok := v.(string)
	return cid, ok
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
