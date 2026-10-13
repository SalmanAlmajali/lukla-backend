// Package httpx provides consistent HTTP response helpers for all handlers.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OK writes a 200 JSON response with the given data payload.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Created writes a 201 JSON response with the created resource.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// NoContent writes a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
