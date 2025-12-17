package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type TreePointHandler struct {
	svc service.TreePointService
}

func NewTreePointHandler(svc service.TreePointService) *TreePointHandler {
	return &TreePointHandler{svc: svc}
}

func (h *TreePointHandler) Get(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	total, balance, err := h.svc.Get(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", err.Error()))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":   total,
		"balance": balance,
	})
}

type usePointRequest struct {
	Points float64 `json:"points"`
}

func (h *TreePointHandler) Use(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	var req usePointRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	if req.Points <= 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "points must be positive"))
	}
	total, balance, err := h.svc.Deduct(c.Request().Context(), uid, req.Points)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":   total,
		"balance": balance,
	})
}
