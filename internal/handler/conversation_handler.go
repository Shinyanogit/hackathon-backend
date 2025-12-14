package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type ConversationHandler struct {
	svc service.ConversationService
}

func NewConversationHandler(svc service.ConversationService) *ConversationHandler {
	return &ConversationHandler{svc: svc}
}

type ConversationResponse struct {
	ConversationID uint64 `json:"conversationId"`
	ItemID         uint64 `json:"itemId"`
	SellerUID      string `json:"sellerUid"`
	BuyerUID       string `json:"buyerUid"`
	HasUnread      bool   `json:"hasUnread,omitempty"`
}

type MessageRequest struct {
	Body          string  `json:"body"`
	SenderName    string  `json:"senderName"`
	SenderIconUrl *string `json:"senderIconUrl"`
}

func (h *ConversationHandler) CreateFromItem(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	itemIDParam := c.Param("id")
	itemID, err := strconv.ParseUint(itemIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	cv, err := h.svc.CreateOrGet(c.Request().Context(), itemID, uid)
	if err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
		}
		if err.Error() == "cannot chat with yourself" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
	}
	return c.JSON(http.StatusOK, ConversationResponse{
		ConversationID: cv.ID,
		ItemID:         cv.ItemID,
		SellerUID:      cv.SellerUID,
		BuyerUID:       cv.BuyerUID,
	})
}

func (h *ConversationHandler) List(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convs, err := h.svc.ListByUser(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch conversations"))
	}
	resp := make([]ConversationResponse, 0, len(convs))
	for _, cv := range convs {
		resp = append(resp, ConversationResponse{
			ConversationID: cv.ID,
			ItemID:         cv.ItemID,
			SellerUID:      cv.SellerUID,
			BuyerUID:       cv.BuyerUID,
			HasUnread:      cv.HasUnread,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ConversationHandler) Get(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convIDParam := c.Param("id")
	convID, err := strconv.ParseUint(convIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid conversation id"))
	}
	cv, err := h.svc.Get(c.Request().Context(), convID, uid)
	if err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "conversation not found"))
		}
		if err.Error() == "forbidden" {
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not a participant"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch conversation"))
	}
	return c.JSON(http.StatusOK, ConversationResponse{
		ConversationID: cv.ID,
		ItemID:         cv.ItemID,
		SellerUID:      cv.SellerUID,
		BuyerUID:       cv.BuyerUID,
	})
}

func (h *ConversationHandler) MarkRead(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convIDParam := c.Param("id")
	convID, err := strconv.ParseUint(convIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid conversation id"))
	}
	if err := h.svc.MarkRead(c.Request().Context(), convID, uid); err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to mark read"))
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ConversationHandler) ListMessages(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convIDParam := c.Param("id")
	convID, err := strconv.ParseUint(convIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid conversation id"))
	}
	msgs, err := h.svc.ListMessages(c.Request().Context(), convID, uid)
	if err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "conversation not found"))
		}
		if err.Error() == "forbidden" {
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not a participant"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch messages"))
	}
	return c.JSON(http.StatusOK, msgs)
}

func (h *ConversationHandler) CreateMessage(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convIDParam := c.Param("id")
	convID, err := strconv.ParseUint(convIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid conversation id"))
	}
	var req MessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	if err := h.svc.CreateMessage(c.Request().Context(), convID, uid, req.Body, req.SenderName, req.SenderIconUrl); err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "conversation not found"))
		}
		if err.Error() == "forbidden" {
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not a participant"))
		}
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
}
