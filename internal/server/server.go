package server

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/shinyyama/hackathon-backend/internal/handler"
	appmw "github.com/shinyyama/hackathon-backend/internal/middleware"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"github.com/shinyyama/hackathon-backend/internal/service"
	"gorm.io/gorm"
)

type Server struct {
	e        *echo.Echo
	itemRepo repository.ItemRepository
	convRepo repository.ConversationRepository
	sha      string
	build    string
}

func New(db *gorm.DB, sha, buildTime string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000", "*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	itemRepo := repository.NewItemRepository(db)
	itemSvc := service.NewItemService(itemRepo)
	itemHandler := handler.NewItemHandler(itemSvc)

	convRepo := repository.NewConversationRepository(db)
	convSvc := service.NewConversationService(convRepo, itemRepo)
	convHandler := handler.NewConversationHandler(convSvc)

	authMw, err := appmw.NewAuthMiddleware(context.Background())
	if err != nil {
		e.Logger.Fatalf("failed to init firebase auth: %v", err)
	}

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"ok":         "true",
			"git_sha":    sha,
			"build_time": buildTime,
		})
	})

	api := e.Group("/api")
	if authMw != nil {
		api.POST("/items", itemHandler.Create, authMw.RequireAuth)
		api.GET("/me/items", itemHandler.ListMine, authMw.RequireAuth)
		api.POST("/items/:id/conversations", convHandler.CreateFromItem, authMw.RequireAuth)
		api.GET("/conversations", convHandler.List, authMw.RequireAuth)
		api.GET("/conversations/:id", convHandler.Get, authMw.RequireAuth)
		api.GET("/conversations/:id/messages", convHandler.ListMessages, authMw.RequireAuth)
		api.POST("/conversations/:id/messages", convHandler.CreateMessage, authMw.RequireAuth)
	} else {
		api.POST("/items", itemHandler.Create)
		api.GET("/me/items", itemHandler.ListMine)
		api.POST("/items/:id/conversations", convHandler.CreateFromItem)
		api.GET("/conversations", convHandler.List)
		api.GET("/conversations/:id", convHandler.Get)
		api.GET("/conversations/:id/messages", convHandler.ListMessages)
		api.POST("/conversations/:id/messages", convHandler.CreateMessage)
	}
	api.GET("/items", itemHandler.List)
	api.GET("/items/:id", itemHandler.Get)

	return &Server{e: e, itemRepo: itemRepo, convRepo: convRepo, sha: sha, build: buildTime}
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) SetDB(db *gorm.DB) {
	if s.itemRepo != nil {
		s.itemRepo.SetDB(db)
	}
	if s.convRepo != nil {
		s.convRepo = repository.NewConversationRepository(db)
	}
}
