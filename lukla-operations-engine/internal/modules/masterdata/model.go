// Package masterdata manages master data entities for Lukla Operations Engine:
// Clients, Articles, Components, Tailors/Operators, and Defect Categories.
package masterdata

import "time"

// Client represents a partner client / brand for garment and bag maklon.
type Client struct {
	ID         string     `json:"id"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Contact    *string    `json:"contact"`
	Address    *string    `json:"address"`
	ClientType string     `json:"clientType"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt,omitempty"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty"`
}

// ListClientsFilter defines pagination and search parameters for listing clients.
type ListClientsFilter struct {
	Search string `form:"search"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}

// ClientDetail represents detailed client information including summary counts.
type ClientDetail struct {
	Client
	ArticlesCount     int `json:"articlesCount"`
	ActiveOrdersCount int `json:"activeOrdersCount"`
}


