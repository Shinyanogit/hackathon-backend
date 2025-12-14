package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/service"
)

type ItemHandler struct {
	svc service.ItemService
}

func NewItemHandler(svc service.ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
}

type ItemResponse struct {
	ID           uint64  `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Price        uint    `json:"price"`
	ImageURL     *string `json:"imageUrl"`
	CategorySlug string  `json:"categorySlug"`
	SellerUID    string  `json:"sellerUid"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type ItemListResponse struct {
	Items []ItemResponse `json:"items"`
	Total int64          `json:"total"`
}

type CreateItemRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Price        uint    `json:"price"`
	ImageURL     *string `json:"imageUrl"`
	CategorySlug string  `json:"categorySlug"`
	SellerUID    string  `json:"sellerUid"`
}

func (h *ItemHandler) Create(c echo.Context) error {
	var req CreateItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	sellerUID, _ := c.Get("uid").(string)
	item, err := h.svc.Create(c.Request().Context(), req.Title, req.Description, req.Price, req.ImageURL, req.CategorySlug, sellerUID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
	}
	return c.JSON(http.StatusCreated, toItemResponse(item))
}

func (h *ItemHandler) Get(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid id"))
	}
	item, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch item"))
	}
	return c.JSON(http.StatusOK, toItemResponse(item))
}

func (h *ItemHandler) List(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	category := c.QueryParam("category")
	query := c.QueryParam("query")
	sellerUID := c.QueryParam("sellerUid")
	items, total, err := h.svc.List(c.Request().Context(), limit, offset, category, query, sellerUID)
	if err != nil {
		c.Logger().Errorf("list items error: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch items"))
	}
	resp := ItemListResponse{
		Items: make([]ItemResponse, 0, len(items)),
		Total: total,
	}
	for i := range items {
		resp.Items = append(resp.Items, toItemResponse(&items[i]))
	}
	return c.JSON(http.StatusOK, resp)
}

func toItemResponse(item *model.Item) ItemResponse {
	return ItemResponse{
		ID:           item.ID,
		Title:        item.Title,
		Description:  item.Description,
		Price:        item.Price,
		ImageURL:     item.ImageURL,
		CategorySlug: item.CategorySlug,
		SellerUID:    item.SellerUID,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *ItemHandler) ListMine(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	items, err := h.svc.ListBySeller(c.Request().Context(), uid)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to fetch items"))
	}
	resp := ItemListResponse{
		Items: make([]ItemResponse, 0, len(items)),
		Total: int64(len(items)),
	}
	for i := range items {
		resp.Items = append(resp.Items, toItemResponse(&items[i]))
	}
	return c.JSON(http.StatusOK, resp)
}

type UpdateItemRequest struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Price        *uint   `json:"price"`
	ImageURL     *string `json:"imageUrl"`
	CategorySlug *string `json:"categorySlug"`
}

func (h *ItemHandler) Update(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid id"))
	}
	var req UpdateItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	title := ""
	if req.Title != nil {
		title = *req.Title
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	var price uint
	if req.Price != nil {
		price = *req.Price
	}
	imageURL := req.ImageURL
	category := ""
	if req.CategorySlug != nil {
		category = *req.CategorySlug
	}
	item, err := h.svc.UpdateOwned(c.Request().Context(), id, uid, title, description, price, imageURL, category)
	if err != nil {
		if err == service.ErrNotFound {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found or not owner"))
		}
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", err.Error()))
	}
	return c.JSON(http.StatusOK, toItemResponse(item))
}
