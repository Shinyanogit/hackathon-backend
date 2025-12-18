package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type PurchaseHandler struct {
	svc    service.PurchaseService
	notify service.NotificationService
}

func NewPurchaseHandler(svc service.PurchaseService, notify service.NotificationService) *PurchaseHandler {
	return &PurchaseHandler{svc: svc, notify: notify}
}

type PurchaseResponse struct {
	ID             uint64  `json:"id"`
	ItemID         uint64  `json:"itemId"`
	BuyerUID       string  `json:"buyerUid"`
	SellerUID      string  `json:"sellerUid"`
	ConversationID uint64  `json:"conversationId"`
	Status         string  `json:"status"`
	PointsUsed     float64 `json:"pointsUsed"`
	PaidYen        int64   `json:"paidYen"`
	ShippingQRURL  string  `json:"shippingQrUrl"`
	ShippingNote   string  `json:"shippingNote"`
	ShippedAt      *string `json:"shippedAt,omitempty"`
	DeliveredAt    *string `json:"deliveredAt,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

func toPurchaseResponse(p *model.Purchase) PurchaseResponse {
	var shippedAt, deliveredAt *string
	if p.ShippedAt != nil {
		val := p.ShippedAt.Format(time.RFC3339)
		shippedAt = &val
	}
	if p.DeliveredAt != nil {
		val := p.DeliveredAt.Format(time.RFC3339)
		deliveredAt = &val
	}
	return PurchaseResponse{
		ID:             p.ID,
		ItemID:         p.ItemID,
		BuyerUID:       p.BuyerUID,
		SellerUID:      p.SellerUID,
		ConversationID: p.ConversationID,
		Status:         string(p.Status),
		PointsUsed:     p.PointsUsed,
		PaidYen:        p.PaidYen,
		ShippingQRURL:  p.ShippingQRURL,
		ShippingNote:   p.ShippingNote,
		ShippedAt:      shippedAt,
		DeliveredAt:    deliveredAt,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *PurchaseHandler) PurchaseItem(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	var body struct {
		PointsUsed float64 `json:"pointsUsed"`
	}
	_ = c.Bind(&body)
	itemIDParam := c.Param("id")
	itemID, err := strconv.ParseUint(itemIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	p, err := h.svc.PurchaseItem(c.Request().Context(), itemID, uid, body.PointsUsed)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
		case service.ErrAlreadyPurchased:
			return c.JSON(http.StatusConflict, NewErrorResponse("already_purchased", "item already purchased"))
		default:
			if err.Error() == "cannot buy your own item" {
				return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
			}
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
	}
	return c.JSON(http.StatusCreated, toPurchaseResponse(p))
}

func (h *PurchaseHandler) GetByItem(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	itemIDParam := c.Param("id")
	itemID, err := strconv.ParseUint(itemIDParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	p, err := h.svc.GetByItem(c.Request().Context(), itemID, uid)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "purchase not found"))
		case service.ErrForbidden:
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not allowed"))
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
	}
	if h.notify != nil && p != nil {
		_ = h.notify.MarkByPurchase(c.Request().Context(), uid, p.ID)
		if p.ConversationID != 0 {
			_ = h.notify.MarkByConversation(c.Request().Context(), uid, p.ConversationID)
		}
	}
	return c.JSON(http.StatusOK, toPurchaseResponse(p))
}

func (h *PurchaseHandler) MarkShipped(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	idParam := c.Param("id")
	purchaseID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid purchase id"))
	}
	p, err := h.svc.MarkShipped(c.Request().Context(), purchaseID, uid)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "purchase not found"))
		case service.ErrForbidden:
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not allowed"))
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
	}
	return c.JSON(http.StatusOK, toPurchaseResponse(p))
}

func (h *PurchaseHandler) MarkDelivered(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	idParam := c.Param("id")
	purchaseID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid purchase id"))
	}
	p, err := h.svc.MarkDelivered(c.Request().Context(), purchaseID, uid)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "purchase not found"))
		case service.ErrForbidden:
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not allowed"))
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
	}
	return c.JSON(http.StatusOK, toPurchaseResponse(p))
}

func (h *PurchaseHandler) Cancel(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	idParam := c.Param("id")
	purchaseID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid purchase id"))
	}
	p, err := h.svc.Cancel(c.Request().Context(), purchaseID, uid)
	if err != nil {
		switch err {
		case service.ErrNotFound:
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "purchase not found"))
		case service.ErrForbidden:
			return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "not allowed"))
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
		}
	}
	return c.JSON(http.StatusOK, toPurchaseResponse(p))
}

type PurchaseWithItemResponse struct {
	Purchase PurchaseResponse `json:"purchase"`
	Item     ItemResponse     `json:"item"`
}

func (h *PurchaseHandler) ListMine(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	list, err := h.svc.ListByBuyer(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch purchases"))
	}
	resp := make([]PurchaseWithItemResponse, 0, len(list))
	for _, row := range list {
		itemResp := ItemResponse{
			ID:           row.Purchase.ItemID,
			Title:        "",
			Description:  "",
			Price:        0,
			ImageURL:     nil,
			CategorySlug: "",
			SellerUID:    row.Purchase.SellerUID,
			CreatedAt:    "",
			UpdatedAt:    "",
		}
		if row.Item != nil {
			itemResp = toItemResponse(row.Item)
		}
		resp = append(resp, PurchaseWithItemResponse{
			Purchase: toPurchaseResponse(&row.Purchase),
			Item:     itemResp,
		})
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *PurchaseHandler) ListSales(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	list, err := h.svc.ListBySeller(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch sales"))
	}
	resp := make([]PurchaseWithItemResponse, 0, len(list))
	for _, row := range list {
		itemResp := ItemResponse{
			ID:           row.Purchase.ItemID,
			Title:        "",
			Description:  "",
			Price:        0,
			ImageURL:     nil,
			CategorySlug: "",
			SellerUID:    row.Purchase.SellerUID,
			CreatedAt:    "",
			UpdatedAt:    "",
		}
		if row.Item != nil {
			itemResp = toItemResponse(row.Item)
		}
		resp = append(resp, PurchaseWithItemResponse{
			Purchase: toPurchaseResponse(&row.Purchase),
			Item:     itemResp,
		})
	}
	return c.JSON(http.StatusOK, resp)
}
