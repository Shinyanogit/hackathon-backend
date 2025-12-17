package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type RevenueHandler struct {
	svc service.RevenueService
}

func NewRevenueHandler(svc service.RevenueService) *RevenueHandler {
	return &RevenueHandler{svc: svc}
}

func (h *RevenueHandler) Get(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	rev, err := h.svc.Get(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch revenue"))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"revenueCents": rev})
}

func (h *RevenueHandler) Withdraw(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	amtStr := c.FormValue("amountCents")
	if amtStr == "" {
		amtStr = c.QueryParam("amountCents")
	}
	amt, err := strconv.ParseInt(amtStr, 10, 64)
	if err != nil || amt <= 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid amount"))
	}
	rev, err := h.svc.Deduct(c.Request().Context(), uid, amt)
	if err != nil {
		if err.Error() == "insufficient balance" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "insufficient balance"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to withdraw"))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"revenueCents": rev})
}
