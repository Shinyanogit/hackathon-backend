package server

import (
	"context"
	"net/http"
	"os"
	"strings"

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

	// CORS allowlist (localhost + Vercel 等)。FRONTEND_ORIGINS をカンマ区切りで上書き可。
	allowed := map[string]struct{}{
		"http://localhost:3000":  {},
		"http://127.0.0.1:3000": {},
	}
	if env := os.Getenv("FRONTEND_ORIGINS"); env != "" {
		for _, o := range strings.Split(env, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowed[o] = struct{}{}
			}
		}
	}
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get(echo.HeaderOrigin)
			if _, ok := allowed[origin]; ok {
				res := c.Response().Header()
				res.Set(echo.HeaderAccessControlAllowOrigin, origin)
				res.Set(echo.HeaderVary, "Origin")
				res.Set(echo.HeaderAccessControlAllowMethods, "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				res.Set(echo.HeaderAccessControlAllowHeaders, "Content-Type, Authorization")
				res.Set(echo.HeaderAccessControlAllowCredentials, "true")
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusNoContent)
				}
			}
			return next(c)
		}
	})

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
		api.DELETE("/conversations/:id/messages/:msgId", convHandler.DeleteMessage, authMw.RequireAuth)
		api.POST("/conversations/:id/read", convHandler.MarkRead, authMw.RequireAuth)
	} else {
		api.POST("/items", itemHandler.Create)
		api.GET("/me/items", itemHandler.ListMine)
		api.POST("/items/:id/conversations", convHandler.CreateFromItem)
		api.GET("/conversations", convHandler.List)
		api.GET("/conversations/:id", convHandler.Get)
		api.GET("/conversations/:id/messages", convHandler.ListMessages)
		api.POST("/conversations/:id/messages", convHandler.CreateMessage)
		api.DELETE("/conversations/:id/messages/:msgId", convHandler.DeleteMessage)
		api.POST("/conversations/:id/read", convHandler.MarkRead)
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
