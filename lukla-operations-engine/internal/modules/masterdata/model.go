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

// Article represents a manufactured garment or bag model under a specific client.
type Article struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"clientId"`
	ClientName  string     `json:"clientName,omitempty"`
	Title       string     `json:"title"`
	Code        string     `json:"code"`
	Category    string     `json:"category"`
	Description *string    `json:"description"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt,omitempty"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// ListArticlesFilter defines query filtering and pagination for listing articles.
type ListArticlesFilter struct {
	ClientID string `form:"clientId"`
	Category string `form:"category"`
	Search   string `form:"search"`
	Page     int    `form:"page"`
	Limit    int    `form:"limit"`
}

// CreateArticleRequest defines the request body for creating a new article.
type CreateArticleRequest struct {
	ClientID    string  `json:"clientId" binding:"required"`
	Title       string  `json:"title" binding:"required"`
	Code        string  `json:"code" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Description *string `json:"description"`
}

// ArticleDetail represents detailed article information including its components.
type ArticleDetail struct {
	Article
	Components []Component `json:"components"`
}

// Component represents an operational step / component in manufacturing an article.
type Component struct {
	ID             string     `json:"id"`
	ArticleID      string     `json:"articleId"`
	Name           string     `json:"name"`
	Code           *string    `json:"code"`
	SequenceNumber int        `json:"sequenceNumber"`
	StandardSAM    *float64   `json:"standardSAM"`
	CreatedAt      time.Time  `json:"createdAt,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt,omitempty"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

// CreateComponentRequest defines the payload for creating a new component.
type CreateComponentRequest struct {
	Name           string   `json:"name" binding:"required"`
	Code           *string  `json:"code"`
	SequenceNumber *int     `json:"sequenceNumber"`
	StandardSAM    *float64 `json:"standardSAM"`
}

// OperatorStatus represents the employment or availability status of an operator.
type OperatorStatus string

const (
	OperatorStatusActive   OperatorStatus = "ACTIVE"
	OperatorStatusInactive OperatorStatus = "INACTIVE"
	OperatorStatusOnLeave  OperatorStatus = "ON_LEAVE"
)

// Operator represents a sewing operator / tailor in the factory.
type Operator struct {
	ID           string         `json:"id"`
	OperatorCode string         `json:"operatorCode"`
	Name         string         `json:"name"`
	Phone        *string        `json:"phone,omitempty"`
	JoinDate     string         `json:"joinDate"` // Format: YYYY-MM-DD
	Status       OperatorStatus `json:"status"`
	DefaultLine  *string        `json:"defaultLine,omitempty"`
	SkillTier    *string        `json:"skillTier,omitempty"`
	UserID       *string        `json:"userId,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt,omitempty"`
	DeletedAt    *time.Time     `json:"deletedAt,omitempty"`
}

// ListOperatorsFilter defines query parameters for filtering operators.
type ListOperatorsFilter struct {
	Status string `form:"status"`
	Line   string `form:"line"`
	Search string `form:"search"`
}

