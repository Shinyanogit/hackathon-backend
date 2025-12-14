package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
)

type AIHandler struct {
	itemRepo repository.ItemRepository
	apiKey   string
	client   *http.Client
}

func NewAIHandler(itemRepo repository.ItemRepository, apiKey string) *AIHandler {
	return &AIHandler{
		itemRepo: itemRepo,
		apiKey:   apiKey,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type askRequest struct {
	Question string `json:"question"`
}

type geminiCandidate struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

func (h *AIHandler) AskItem(c echo.Context) error {
	if h.apiKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "GEMINI_API_KEY is not set"))
	}
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	id := c.Param("id")
	var req askRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid json"))
	}
	if req.Question == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "question is required"))
	}
	itemID, parseErr := strconv.ParseUint(id, 10, 64)
	if parseErr != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid item id"))
	}
	item, err := h.itemRepo.FindByID(c.Request().Context(), itemID)
	if err != nil {
		return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
	}
	if item.SellerUID != "" && item.SellerUID == uid {
		return c.JSON(http.StatusForbidden, NewErrorResponse("forbidden", "cannot ask about own item"))
	}

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": "You are a helpful shopping assistant for a flea market app. Answer concisely in Japanese. If the question is unrelated, politely guide the user back to the item.",
					},
					{
						"text": fmt.Sprintf("商品データ:\nタイトル: %s\n説明: %s\n価格: %d\nカテゴリ: %s", item.Title, item.Description, item.Price, item.CategorySlug),
					},
					{
						"text": fmt.Sprintf("質問: %s", req.Question),
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.5,
			"maxOutputTokens": 256,
		},
	}

	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=%s", os.Getenv("GEMINI_API_KEY"))
	resp, err := h.client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return c.JSON(http.StatusBadGateway, NewErrorResponse("upstream_error", "failed to call gemini"))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return c.JSON(http.StatusBadGateway, NewErrorResponse("upstream_error", "gemini returned error"))
	}
	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return c.JSON(http.StatusBadGateway, NewErrorResponse("upstream_error", "failed to parse gemini response"))
	}
	answer := ""
	if len(gResp.Candidates) > 0 && len(gResp.Candidates[0].Content.Parts) > 0 {
		answer = gResp.Candidates[0].Content.Parts[0].Text
	}
	if answer == "" {
		answer = "回答を生成できませんでした。"
	}
	return c.JSON(http.StatusOK, map[string]string{"answer": answer})
}
