package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type NotificationHandler struct {
	svc service.NotificationService
}

func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

type NotificationResponse struct {
	ID             uint64  `json:"id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	ItemID         *uint64 `json:"itemId,omitempty"`
	ConversationID *uint64 `json:"conversationId,omitempty"`
	PurchaseID     *uint64 `json:"purchaseId,omitempty"`
	Read           bool    `json:"read"`
	CreatedAt      string  `json:"createdAt"`
}

func toNotificationResponse(n model.Notification) NotificationResponse {
	return NotificationResponse{
		ID:             n.ID,
		Type:           n.Type,
		Title:          n.Title,
		Body:           n.Body,
		ItemID:         n.ItemID,
		ConversationID: n.ConversationID,
		PurchaseID:     n.PurchaseID,
		Read:           n.ReadAt != nil,
		CreatedAt:      n.CreatedAt.Format(time.RFC3339),
	}
}

func (h *NotificationHandler) List(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	unreadOnly := c.QueryParam("unread_only") != "false"
	limit := 20
	if lStr := c.QueryParam("limit"); lStr != "" {
		if lParsed, err := strconv.Atoi(lStr); err == nil && lParsed > 0 {
			limit = lParsed
		}
	}
	list, unreadCount, err := h.svc.List(c.Request().Context(), uid, unreadOnly, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch notifications"))
	}
	resp := make([]NotificationResponse, 0, len(list))
	for _, n := range list {
		resp = append(resp, toNotificationResponse(n))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": resp,
		"unreadCount":   unreadCount,
	})
}

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	if err := h.svc.MarkAllRead(c.Request().Context(), uid); err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to mark read"))
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
