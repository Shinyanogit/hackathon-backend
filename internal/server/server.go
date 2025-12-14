package server

import (
	"context"
	"net/http"
	"net/url"
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
	e            *echo.Echo
	itemRepo     repository.ItemRepository
	convRepo     repository.ConversationRepository
	purchaseRepo repository.PurchaseRepository
	sha          string
	build        string
}

func New(db *gorm.DB, sha, buildTime string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) (bool, error) {
			low := strings.ToLower(origin)
			if strings.HasPrefix(low, "http://localhost:") || strings.HasPrefix(low, "http://127.0.0.1:") ||
				strings.HasPrefix(low, "https://localhost:") || strings.HasPrefix(low, "https://127.0.0.1:") {
				return true, nil
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false, nil
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return false, nil
			}
			host := u.Hostname()
			if strings.HasSuffix(host, "vercel.app") {
				return true, nil
			}
			return false, nil
		},
	}))

	itemRepo := repository.NewItemRepository(db)
	itemSvc := service.NewItemService(itemRepo)
	itemHandler := handler.NewItemHandler(itemSvc)

	convRepo := repository.NewConversationRepository(db)
	convSvc := service.NewConversationService(convRepo, itemRepo)
	convHandler := handler.NewConversationHandler(convSvc)

	purchaseRepo := repository.NewPurchaseRepository(db)
	purchaseSvc := service.NewPurchaseService(purchaseRepo, itemRepo, convRepo)
	purchaseHandler := handler.NewPurchaseHandler(purchaseSvc)

	aiHandler := handler.NewAIHandler(itemRepo, os.Getenv("GEMINI_API_KEY"))

	authMw, err := appmw.NewAuthMiddleware(context.Background())
	if err != nil {
		e.Logger.Fatalf("failed to init firebase auth: %v", err)
	}
	var userHandler *handler.UserHandler
	if authMw != nil && authMw.Client() != nil {
		userHandler = handler.NewUserHandler(authMw.Client(), authMw.EnvProjectID(), authMw.CredProjectID())
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
		api.PUT("/items/:id", itemHandler.Update, authMw.RequireAuth)
		api.GET("/me/items", itemHandler.ListMine, authMw.RequireAuth)
		api.GET("/me/purchases", purchaseHandler.ListMine, authMw.RequireAuth)
		api.GET("/me/sales", purchaseHandler.ListSales, authMw.RequireAuth)
		api.POST("/items/:id/conversations", convHandler.CreateFromItem, authMw.RequireAuth)
		api.POST("/items/:id/purchase", purchaseHandler.PurchaseItem, authMw.RequireAuth)
		api.GET("/items/:id/purchase", purchaseHandler.GetByItem, authMw.RequireAuth)
		api.POST("/items/:id/ask", aiHandler.AskItem, authMw.RequireAuth)
		api.GET("/items/:id/thread", convHandler.GetThread)
		api.POST("/items/:id/messages", convHandler.PostMessageToItem, authMw.RequireAuth)
		api.GET("/conversations", convHandler.List, authMw.RequireAuth)
		api.GET("/conversations/:id", convHandler.Get, authMw.RequireAuth)
		api.GET("/conversations/:id/messages", convHandler.ListMessages, authMw.RequireAuth)
		api.POST("/conversations/:id/messages", convHandler.CreateMessage, authMw.RequireAuth)
		api.DELETE("/conversations/:id/messages/:msgId", convHandler.DeleteMessage, authMw.RequireAuth)
		api.POST("/conversations/:id/read", convHandler.MarkRead, authMw.RequireAuth)
		api.POST("/purchases/:id/ship", purchaseHandler.MarkShipped, authMw.RequireAuth)
		api.POST("/purchases/:id/receive", purchaseHandler.MarkDelivered, authMw.RequireAuth)
		api.POST("/purchases/:id/cancel", purchaseHandler.Cancel, authMw.RequireAuth)
	} else {
		api.POST("/items", itemHandler.Create)
		api.PUT("/items/:id", itemHandler.Update)
		api.GET("/me/items", itemHandler.ListMine)
		api.GET("/me/purchases", purchaseHandler.ListMine)
		api.GET("/me/sales", purchaseHandler.ListSales)
		api.POST("/items/:id/conversations", convHandler.CreateFromItem)
		api.POST("/items/:id/purchase", purchaseHandler.PurchaseItem)
		api.GET("/items/:id/purchase", purchaseHandler.GetByItem)
		api.POST("/items/:id/ask", aiHandler.AskItem)
		api.GET("/items/:id/thread", convHandler.GetThread)
		api.POST("/items/:id/messages", convHandler.PostMessageToItem)
		api.GET("/conversations", convHandler.List)
		api.GET("/conversations/:id", convHandler.Get)
		api.GET("/conversations/:id/messages", convHandler.ListMessages)
		api.POST("/conversations/:id/messages", convHandler.CreateMessage)
		api.DELETE("/conversations/:id/messages/:msgId", convHandler.DeleteMessage)
		api.POST("/conversations/:id/read", convHandler.MarkRead)
		api.POST("/purchases/:id/ship", purchaseHandler.MarkShipped)
		api.POST("/purchases/:id/receive", purchaseHandler.MarkDelivered)
		api.POST("/purchases/:id/cancel", purchaseHandler.Cancel)
	}
	api.GET("/items", itemHandler.List)
	api.GET("/items/:id", itemHandler.Get)
	if userHandler != nil {
		api.GET("/users/:uid/public", userHandler.GetPublic)
	}

	return &Server{e: e, itemRepo: itemRepo, convRepo: convRepo, purchaseRepo: purchaseRepo, sha: sha, build: buildTime}
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
	if s.purchaseRepo != nil {
		s.purchaseRepo = repository.NewPurchaseRepository(db)
	}
}
