package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/service"
	"gorm.io/gorm"
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

type DeleteMessageRequest struct {
	MessageID uint64 `json:"messageId"`
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

func (h *ConversationHandler) DeleteMessage(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	convIDParam := c.Param("id")
	msgIDParam := c.Param("msgId")
	convID, err := strconv.ParseUint(convIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid conversation id"))
	}
	msgID, err := strconv.ParseUint(msgIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid message id"))
	}
	if err := h.svc.DeleteMessage(c.Request().Context(), convID, msgID, uid); err != nil {
		if err == service.ErrNotFound || err.Error() == gorm.ErrRecordNotFound.Error() {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "message not found"))
		}
		if err.Error() == "forbidden" {
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not allowed"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to delete message"))
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

type ThreadResponse struct {
	ConversationID *uint64         `json:"conversationId"`
	Messages       []ThreadMessage `json:"messages"`
}

type ThreadMessage struct {
	ID              uint64  `json:"id"`
	ConversationID  uint64  `json:"conversationId"`
	SenderUID       string  `json:"senderUid"`
	SenderName      string  `json:"senderName"`
	SenderIconURL   *string `json:"senderIconUrl,omitempty"`
	ParentMessageID *uint64 `json:"parentMessageId,omitempty"`
	Depth           int     `json:"depth"`
	Body            string  `json:"body"`
	CreatedAt       string  `json:"createdAt"`
}

type PostMessageRequest struct {
	Text            string  `json:"text"`
	ParentMessageID *uint64 `json:"parentMessageId"`
	SenderName      string  `json:"senderName"`
	SenderIconURL   *string `json:"senderIconUrl"`
}

func toThreadMessage(m model.Message) ThreadMessage {
	return ThreadMessage{
		ID:              m.ID,
		ConversationID:  m.ConversationID,
		SenderUID:       m.SenderUID,
		SenderName:      m.SenderName,
		SenderIconURL:   m.SenderIconURL,
		ParentMessageID: m.ParentMessageID,
		Depth:           m.Depth,
		Body:            m.Body,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
	}
}

func (h *ConversationHandler) GetThread(c echo.Context) error {
	itemIDParam := c.Param("id")
	itemID, err := strconv.ParseUint(itemIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	cv, msgs, err := h.svc.ThreadByItem(c.Request().Context(), itemID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch thread"))
	}
	resp := ThreadResponse{
		ConversationID: nil,
		Messages:       make([]ThreadMessage, 0, len(msgs)),
	}
	if cv != nil {
		id := cv.ID
		resp.ConversationID = &id
	}
	for _, m := range msgs {
		resp.Messages = append(resp.Messages, toThreadMessage(m))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ConversationHandler) PostMessageToItem(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	itemIDParam := c.Param("id")
	itemID, err := strconv.ParseUint(itemIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	var req PostMessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	msg, cv, err := h.svc.PostMessageToItem(c.Request().Context(), itemID, uid, req.Text, req.SenderName, req.SenderIconURL, req.ParentMessageID)
	if err != nil {
		switch err.Error() {
		case "seller cannot create root message":
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", err.Error()))
		case "depth exceeded", "parent not found", "parent not in conversation", "text is required":
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to post message"))
	}
	resp := toThreadMessage(*msg)
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"conversationId": cv.ID,
		"message":        resp,
	})
}
