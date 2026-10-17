# Panduan Kickstart Backend: Go (Gin) — Lukla Operations Engine

Dokumen ini disusun untuk rekan pengembang backend yang akan mengimplementasikan sisi server menggunakan framework **Go (Gin)** (`github.com/gin-gonic/gin`).

Semua rute, skema request/response, dan hak akses merujuk secara ketat pada kontrak:
👉 **[docs/api/openapi-phase0-fondasi.yaml](file:///c:/Frontend/lukla-prototype/docs/api/openapi-phase0-fondasi.yaml)**

---

## 1. Arsitektur & Port Default

- **Port Default**: `:8080` (`http://localhost:8080`)
  > *Catatan*: Frontend telah dikonfigurasi dengan `NEXT_PUBLIC_API_URL=http://localhost:8080`. Frontend tidak memerlukan perubahan konfigurasi apa pun saat berganti dari Mock Server ke Gin Backend.
- **Mock Server Frontend**: Frontend memiliki mock server bawaan yang berjalan di port `8080` (`npm run mock`). Saat Gin backend siap dijalankan, matikan mock server dan jalankan Gin pada port `8080`.

---

## 2. Struktur Proyek Go (Gin) yang Direkomendasikan

```text
backend-lukla/
├── cmd/
│   └── api/
│       └── main.go                  # Entrypoint & router initialization
├── internal/
│   ├── config/                      # Environment & database config
│   ├── middleware/
│   │   ├── auth.go                  # Bearer token extractor & session check
│   │   ├── rbac.go                  # Role enforcement (8 roles)
│   │   └── cors.go                  # CORS handler for Next.js (port 3000)
│   ├── models/                      # Structs mirroring OpenAPI schemas
│   │   ├── user.go
│   │   ├── client.go
│   │   ├── article.go
│   │   ├── operator.go
│   │   ├── rate_card.go
│   │   └── order.go
│   └── handlers/                    # HTTP controllers
│       ├── auth_handler.go
│       ├── client_handler.go
│       ├── article_handler.go
│       ├── operator_handler.go
│       ├── rate_card_handler.go
│       └── order_handler.go
├── go.mod
└── go.sum
```

---

## 3. Konfigurasi CORS (Gin)

Frontend Next.js berjalan di `http://localhost:3000` (atau `3005`). Gin wajib mengizinkan preflight `OPTIONS` dan header otorisasi:

```go
package middleware

import (
	"time"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3005", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
```

---

## 4. Matriks 8 Peran Pengguna (RBAC) & Middleware

Daftar 8 peran baku di sistem:
1. `OWNER`: Akses penuh finansial, tarif, dan master data.
2. `TIM_SAMPLE`: Akses modul sample (View 02), breakdown time study, R&D.
3. `GUDANG`: Manajemen stok bahan baku, kain, aksesoris, dan bundle transit.
4. `KEPALA_PRODUKSI`: Manajemen SPK, penetapan tarif rate card, alokasi operator.
5. `MANDOR`: Input kehadiran penjahit, supervisi line, verifikasi target harian.
6. `PENJAHIT`: Eksekusi jahit, scan QR bundle, pelaporan kuantitas mandiri.
7. `QC_INSPECTOR`: Pemeriksaan kualitas end-line, input rework/reject.
8. `FINANCE_PAYROLL`: Kalkulasi payroll borongan berdasarkan rate card terdaftar.

Contoh middleware otorisasi berbasis peran di Gin:

```go
package middleware

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Asumsikan user role disimpan di context oleh AuthMiddleware
		roleVal, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Token sesi tidak valid"})
			c.Abort()
			return
		}

		userRole := roleVal.(string)
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false, 
			"error": "Akses ditolak: role Anda (" + userRole + ") tidak memiliki otorisasi.",
		})
		c.Abort()
	}
}
```

---

## 5. Ringkasan Endpoint & Grouping Router

```go
package main

import (
	"github.com/gin-gonic/gin"
	"backend-lukla/internal/handlers"
	"backend-lukla/internal/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	v1 := r.Group("/api/v1")
	{
		// 1. Auth
		auth := v1.Group("/auth")
		{
			auth.POST("/login", handlers.Login)
			auth.GET("/me", middleware.AuthRequired(), handlers.GetMe)
			auth.POST("/logout", handlers.Logout)
		}

		// 2. Clients
		clients := v1.Group("/clients")
		{
			clients.GET("", handlers.ListClients)
			clients.POST("", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.CreateClient)
			clients.GET("/:id", handlers.GetClientDetail)
			clients.PUT("/:id", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.UpdateClient)
		}

		// 3. Articles & Components
		articles := v1.Group("/articles")
		{
			articles.GET("", handlers.ListArticles)
			articles.POST("", middleware.RequireRoles("OWNER", "TIM_SAMPLE", "KEPALA_PRODUKSI"), handlers.CreateArticle)
			articles.GET("/:id", handlers.GetArticleDetail)
			articles.GET("/:id/components", handlers.ListArticleComponents)
			articles.POST("/:id/components", middleware.RequireRoles("OWNER", "TIM_SAMPLE", "KEPALA_PRODUKSI"), handlers.CreateComponent)
		}

		// 4. Operators
		operators := v1.Group("/operators")
		{
			operators.GET("", handlers.ListOperators)
			operators.POST("", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.CreateOperator)
			operators.GET("/:id", handlers.GetOperatorDetail)
			operators.PATCH("/:id/status", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI", "MANDOR"), handlers.UpdateOperatorStatus)
		}

		// 5. Rate Cards (BPMN 10)
		rateCards := v1.Group("/rate-cards")
		{
			rateCards.GET("", handlers.ListRateCards)
			rateCards.POST("", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.CreateRateCard)
			rateCards.PUT("/:id", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.UpdateRateCard) // Wajib catat ke RateCardHistory
			rateCards.GET("/:id/history", handlers.GetRateCardHistory)
		}

		// 6. Minimal Orders
		orders := v1.Group("/orders/minimal")
		{
			orders.GET("", handlers.ListMinimalOrders)
			orders.POST("", middleware.RequireRoles("OWNER", "KEPALA_PRODUKSI"), handlers.CreateMinimalOrder)
		}
	}

	return r
}
```

---

## 6. Verifikasi & Pengujian Otomatis

Untuk menguji apakah Gin backend Anda telah 100% mematuhi spesifikasi yang diharapkan oleh frontend:

1. Jalankan server Gin Anda di port `8080`:
   ```bash
   go run cmd/api/main.go
   ```
2. Dari root repositori ini, jalankan script pengujian otomatis:
   ```bash
   npx tsx scripts/test-phase0-api.ts
   ```
3. Jika seluruh 10 pengujian berstatus `[PASS]`, backend Gin Anda siap langsung dipasangkan dengan frontend tanpa penyesuaian tambahan!
