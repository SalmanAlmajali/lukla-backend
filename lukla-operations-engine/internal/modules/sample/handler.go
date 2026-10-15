package sample

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler menangani request HTTP untuk manajemen permintaan sample dan time study internal pabrik.
type Handler struct {
	svc *Service
}

// NewHandler membuat instance Handler baru.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan seluruh endpoint internal sample ke router Gin.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc, internalAuthz gin.HandlerFunc) {
	samplesGroup := rg.Group("/samples", authnMiddleware, internalAuthz)
	{
		samplesGroup.POST("", h.CreateSampleRequest)
		samplesGroup.GET("", h.ListSampleRequests)
		samplesGroup.GET("/:id", h.GetSampleRequestByID)
		samplesGroup.POST("/:id/versions", h.CreateSampleVersion)
		samplesGroup.GET("/:id/versions/:versionNumber/time-studies", h.ListTimeStudies)
		samplesGroup.POST("/:id/versions/:versionNumber/time-studies", h.RecordTimeStudy)
		samplesGroup.PUT("/:id/versions/:versionNumber/commercial-rate", h.SetCommercialRate)
		samplesGroup.POST("/:id/submit-review", h.SubmitReview)
		samplesGroup.POST("/:id/convert-to-order", h.ConvertToOrder)
	}
}

// CreateSampleRequest handles POST /api/v1/samples.
func (h *Handler) CreateSampleRequest(c *gin.Context) {
	var payload CreateSampleRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload permintaan sample tidak valid: " + err.Error(),
			},
		})
		return
	}

	detail, err := h.svc.CreateSampleRequest(c.Request.Context(), payload)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrClientNotFound) || errors.Is(err, ErrArticleNotFound) {
			status = http.StatusBadRequest
			code = "VALIDATION_ERROR"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Permintaan sample v1 berhasil dibuat",
		"data":    detail,
	})
}

// ListSampleRequests handles GET /api/v1/samples.
func (h *Handler) ListSampleRequests(c *gin.Context) {
	clientID := c.Query("clientId")
	status := c.Query("status")
	search := c.Query("search")

	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	list, total, err := h.svc.ListSampleRequests(c.Request.Context(), clientID, status, search, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    list,
		"pagination": gin.H{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
		},
	})
}

// GetSampleRequestByID handles GET /api/v1/samples/:id.
func (h *Handler) GetSampleRequestByID(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.GetSampleRequestByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSampleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// CreateSampleVersion handles POST /api/v1/samples/:id/versions.
func (h *Handler) CreateSampleVersion(c *gin.Context) {
	id := c.Param("id")
	var payload CreateSampleVersionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload revisi versi sample tidak valid: " + err.Error(),
			},
		})
		return
	}

	ver, err := h.svc.CreateSampleVersion(c.Request.Context(), id, payload)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrSampleNotFound) || errors.Is(err, ErrVersionNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		} else if errors.Is(err, ErrSampleAlreadyConverted) {
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Versi revisi sample berhasil diterbitkan",
		"data":    ver,
	})
}

// ListTimeStudies handles GET /api/v1/samples/:id/versions/:versionNumber/time-studies.
func (h *Handler) ListTimeStudies(c *gin.Context) {
	id := c.Param("id")
	verNum, err := strconv.Atoi(c.Param("versionNumber"))
	if err != nil || verNum < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Nomor versi sample tidak valid",
			},
		})
		return
	}

	studies, err := h.svc.ListTimeStudies(c.Request.Context(), id, verNum)
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) || errors.Is(err, ErrSampleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    studies,
	})
}

// RecordTimeStudy handles POST /api/v1/samples/:id/versions/:versionNumber/time-studies.
func (h *Handler) RecordTimeStudy(c *gin.Context) {
	id := c.Param("id")
	verNum, err := strconv.Atoi(c.Param("versionNumber"))
	if err != nil || verNum < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Nomor versi sample tidak valid",
			},
		})
		return
	}

	var payload RecordTimeStudyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload time study tidak valid: " + err.Error(),
			},
		})
		return
	}

	study, err := h.svc.RecordTimeStudy(c.Request.Context(), id, verNum, payload)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrVersionNotFound) || errors.Is(err, ErrSampleNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		} else if errors.Is(err, ErrComponentNotFound) || errors.Is(err, ErrVersionAlreadyLocked) {
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Pencatatan time study berhasil disimpan",
		"data":    study,
	})
}

// SetCommercialRate handles PUT /api/v1/samples/:id/versions/:versionNumber/commercial-rate.
func (h *Handler) SetCommercialRate(c *gin.Context) {
	id := c.Param("id")
	verNum, err := strconv.Atoi(c.Param("versionNumber"))
	if err != nil || verNum < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Nomor versi sample tidak valid",
			},
		})
		return
	}

	var payload SetCommercialRatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload commercial rate tidak valid: " + err.Error(),
			},
		})
		return
	}

	ver, err := h.svc.SetCommercialRate(c.Request.Context(), id, verNum, payload)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrVersionNotFound) || errors.Is(err, ErrSampleNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		} else {
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commercial rate card berhasil ditetapkan",
		"data":    ver,
	})
}

// SubmitReview handles POST /api/v1/samples/:id/submit-review.
func (h *Handler) SubmitReview(c *gin.Context) {
	id := c.Param("id")
	var payload SubmitReviewPayload
	_ = c.ShouldBindJSON(&payload)

	err := h.svc.SubmitReview(c.Request.Context(), id, payload.Notes)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrSampleNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sample berhasil diajukan ke klien dan notifikasi telah dikirim",
	})
}

// ConvertToOrder handles POST /api/v1/samples/:id/convert-to-order.
func (h *Handler) ConvertToOrder(c *gin.Context) {
	id := c.Param("id")
	var payload ConvertSampleToOrderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Payload konversi sample ke order massal tidak valid: " + err.Error(),
			},
		})
		return
	}

	res, err := h.svc.ConvertToOrder(c.Request.Context(), id, payload)
	if err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_SERVER_ERROR"
		if errors.Is(err, ErrSampleNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		} else if errors.Is(err, ErrSampleNotApproved) || errors.Is(err, ErrSampleAlreadyConverted) {
			status = http.StatusBadRequest
			code = "BAD_REQUEST"
		} else {
			status = http.StatusBadRequest
			code = "VALIDATION_ERROR"
		}
		c.JSON(status, gin.H{
			"success": false,
			"error": gin.H{
				"code":    code,
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Sample berhasil dikonversi menjadi Order Produksi Massal",
		"data":    res,
	})
}

