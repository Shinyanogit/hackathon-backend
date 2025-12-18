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
	return c.JSON(http.StatusOK, map[string]interface{}{"revenueYen": rev})
}

func (h *RevenueHandler) Withdraw(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	var body struct {
		AmountYen int64 `json:"amountYen"`
	}
	if bindErr := c.Bind(&body); bindErr != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	amtStr := c.FormValue("amountYen")
	if amtStr != "" {
		if parsed, err := strconv.ParseInt(amtStr, 10, 64); err == nil {
			body.AmountYen = parsed
		}
	}
	if body.AmountYen <= 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid amount"))
	}
	rev, err := h.svc.Deduct(c.Request().Context(), uid, body.AmountYen)
	if err != nil {
		if err.Error() == "insufficient balance" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "insufficient balance"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to withdraw"))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"revenueYen": rev})
}
