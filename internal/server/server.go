package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/shinyyama/hackathon-backend/internal/handler"
	"github.com/shinyyama/hackathon-backend/internal/repository"
	"github.com/shinyyama/hackathon-backend/internal/service"
	"gorm.io/gorm"
)

type Server struct {
	e        *echo.Echo
	itemRepo repository.ItemRepository
}

func New(db *gorm.DB) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000", "*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	itemRepo := repository.NewItemRepository(db)
	itemSvc := service.NewItemService(itemRepo)
	itemHandler := handler.NewItemHandler(itemSvc)

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	})

	api := e.Group("/api")
	api.POST("/items", itemHandler.Create)
	api.GET("/items", itemHandler.List)
	api.GET("/items/:id", itemHandler.Get)

	return &Server{e: e, itemRepo: itemRepo}
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func (s *Server) SetDB(db *gorm.DB) {
	if s.itemRepo != nil {
		s.itemRepo.SetDB(db)
	}
}
