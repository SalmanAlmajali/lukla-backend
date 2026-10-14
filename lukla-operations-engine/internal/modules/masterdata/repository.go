package masterdata

import "context"

// Repository defines data operations for master data entities.
type Repository interface {
	ListClients(ctx context.Context, filter ListClientsFilter) ([]Client, int64, error)
	FindClientByID(ctx context.Context, id string) (*ClientDetail, error)

	ListArticles(ctx context.Context, filter ListArticlesFilter) ([]Article, int64, error)
	FindArticleByID(ctx context.Context, id string) (*Article, error)
	CreateArticle(ctx context.Context, article *Article) error
	ArticleCodeExists(ctx context.Context, code string) (bool, error)

	ListComponentsByArticleID(ctx context.Context, articleID string) ([]Component, error)
	CreateComponent(ctx context.Context, comp *Component) error
}

