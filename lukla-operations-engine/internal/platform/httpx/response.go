// Package httpx provides consistent HTTP response helpers for all handlers.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard success envelope returned by API endpoints.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// PaginatedResponse is the standard paginated success envelope.
type PaginatedResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Meta    PaginationMeta `json:"meta"`
	Message string         `json:"message,omitempty"`
}

// PaginationMeta contains pagination metadata.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// OK writes a 200 JSON response with the given data payload wrapped in standard envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// OKWithMessage writes a 200 JSON response with data and a user message.
func OKWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Created writes a 201 JSON response with the created resource wrapped in standard envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// CreatedWithMessage writes a 201 JSON response with the created resource and a message.
func CreatedWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// Paginated writes a 200 JSON response with paginated data and meta information.
func Paginated(c *gin.Context, data any, meta PaginationMeta) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// NoContent writes a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
