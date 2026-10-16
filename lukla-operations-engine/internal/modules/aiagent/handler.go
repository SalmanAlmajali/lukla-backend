package aiagent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/platform/authn"
)

// Handler handles HTTP endpoints for the AI Agent.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler instance.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts the AI agent endpoints to the Gin router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authnMiddleware gin.HandlerFunc) {
	aiGroup := rg.Group("/ai")
	{
		// Health check endpoint (public / internal)
		aiGroup.GET("/health", h.GetHealth)

		// Authenticated agent prompt endpoint
		aiGroup.POST("/agent", authnMiddleware, h.PostAgentPrompt)
		aiGroup.POST("/chat", authnMiddleware, h.PostAgentPrompt)
	}
}

// GetHealth returns the operational status of the AI agent service.
func (h *Handler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.svc.GetHealth(),
	})
}

// PostAgentPrompt processes inbound queries via streaming (default) or JSON.
func (h *Handler) PostAgentPrompt(c *gin.Context) {
	var req AgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": fmt.Sprintf("Format payload tidak valid: %v", err),
			},
		})
		return
	}

	// Enrich request with authenticated user context
	if role, ok := authn.RoleFrom(c); ok && role != "" {
		req.ContextRole = role
	} else if req.ContextRole == "" {
		req.ContextRole = RoleOwner
	}

	if cid, ok := authn.ClientIDFrom(c); ok && cid != "" {
		req.ClientID = &cid
	}

	isStream := req.Stream == nil || *req.Stream

	// Non-streaming JSON mode
	if !isStream {
		var collectedResult AgentFinalResult
		err := h.svc.StreamAgenticLoop(c.Request.Context(), req, func(event AgentStreamEvent) error {
			if event.Type == "done" {
				if res, ok := event.Data.(AgentFinalResult); ok {
					collectedResult = res
				}
			}
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "AI_EXECUTION_ERROR",
					"message": err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    collectedResult,
		})
		return
	}

	// Streaming mode: application/x-ndjson (Newline Delimited JSON)
	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		// Stream loop callback
		err := h.svc.StreamAgenticLoop(c.Request.Context(), req, func(event AgentStreamEvent) error {
			bytes, err := json.Marshal(event)
			if err != nil {
				return err
			}
			_, writeErr := w.Write(append(bytes, '\n'))
			if writeErr != nil {
				return writeErr
			}
			c.Writer.Flush()
			return nil
		})

		if err != nil {
			errEvent, _ := json.Marshal(AgentStreamEvent{
				Type: "error",
				Data: map[string]string{"message": err.Error()},
			})
			_, _ = w.Write(append(errEvent, '\n'))
			c.Writer.Flush()
		}

		return false // Stop stream after loop completes
	})
}
