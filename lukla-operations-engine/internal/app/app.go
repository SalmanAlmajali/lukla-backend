// Package app wires all application dependencies and manages the HTTP server lifecycle.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lukla/operations-engine/internal/modules/identity"
	"github.com/lukla/operations-engine/internal/modules/masterdata"
	"github.com/lukla/operations-engine/internal/modules/ratecard"
	"github.com/lukla/operations-engine/internal/platform/authn"
	"github.com/lukla/operations-engine/internal/platform/authz"
	"github.com/lukla/operations-engine/internal/platform/database"
)

// App holds the wired application.
type App struct {
	cfg    Config
	db     *sql.DB
	server *http.Server
}

// New loads configuration, initialises dependencies, and returns a ready App.
func New(ctx context.Context) (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// TODO: initialise logger (platform/logger) and pass it to modules.

	db, err := database.Open(ctx, database.Config{
		URL:             cfg.DB.URL,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	identityRepo := identity.NewPostgresRepository(db)
	identitySvc := identity.NewService(identityRepo, cfg.Auth.TokenSecret, cfg.Auth.AccessTokenTTL)
	identityHandler := identity.NewHandler(identitySvc)

	userLoader := func(ctx context.Context, userID string) (*authn.UserIdentity, error) {
		u, err := identityRepo.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, nil
		}
		return &authn.UserIdentity{
			ID:       u.ID,
			RoleCode: u.RoleCode,
			ClientID: u.ClientID,
			IsActive: u.IsActive,
		}, nil
	}
	authnMiddleware := authn.Middleware(cfg.Auth.TokenSecret, userLoader)
	internalAuthz := authz.RequireInternal()

	masterdataRepo := masterdata.NewPostgresRepository(db)
	masterdataSvc := masterdata.NewService(masterdataRepo)
	masterdataHandler := masterdata.NewHandler(masterdataSvc)

	ratecardRepo := ratecard.NewPostgresRepository(db)
	ratecardSvc := ratecard.NewService(ratecardRepo)
	ratecardHandler := ratecard.NewHandler(ratecardSvc)

	router := gin.New()
	router.Use(gin.Recovery())
	registerRoutes(router, identityHandler, masterdataHandler, ratecardHandler, authnMiddleware, internalAuthz)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{cfg: cfg, db: db, server: srv}, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled, then gracefully shuts down.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		_ = a.db.Close()
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		err := a.server.Shutdown(shutdownCtx)
		_ = a.db.Close()
		return err
	}
}

func registerRoutes(
	r *gin.Engine,
	identityHandler *identity.Handler,
	masterdataHandler *masterdata.Handler,
	ratecardHandler *ratecard.Handler,
	authnMiddleware gin.HandlerFunc,
	internalAuthz gin.HandlerFunc,
) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	identity.RegisterRoutes(v1, identityHandler, authnMiddleware)
	masterdata.RegisterRoutes(v1, masterdataHandler, authnMiddleware, internalAuthz)
	ratecardHandler.RegisterRoutes(v1, authnMiddleware)
}

