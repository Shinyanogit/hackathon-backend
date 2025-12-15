package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shinyyama/hackathon-backend/internal/ai"
	"github.com/shinyyama/hackathon-backend/internal/repository"
)

type AIHandler struct {
	itemRepo       repository.ItemRepository
	convRepo       repository.ConversationRepository
	apiKey         string
	client         *http.Client
	imageClient    *ai.GeminiImageClient
	storage        *storage.Client
	storageBucket  string
	maxUploadBytes int64
}

func NewAIHandler(
	itemRepo repository.ItemRepository,
	convRepo repository.ConversationRepository,
	geminiAPIKey string,
	imageClient *ai.GeminiImageClient,
	storageClient *storage.Client,
	storageBucket string,
) *AIHandler {
	return &AIHandler{
		itemRepo:       itemRepo,
		convRepo:       convRepo,
		apiKey:         geminiAPIKey,
		imageClient:    imageClient,
		storage:        storageClient,
		storageBucket:  storageBucket,
		maxUploadBytes: 5 * 1024 * 1024, // 5MB
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

	// recent conversation messages (for context)
	var conversationText string
	if h.convRepo != nil {
		if cv, err := h.convRepo.FindByItem(c.Request().Context(), itemID); err == nil {
			if msgs, err := h.convRepo.ListMessages(c.Request().Context(), cv.ID); err == nil {
				if len(msgs) > 10 {
					msgs = msgs[len(msgs)-10:]
				}
				var b strings.Builder
				for _, m := range msgs {
					fmt.Fprintf(&b, "[%s] %s\n", m.SenderUID, m.Body)
				}
				conversationText = b.String()
			}
		}
	}

	parts := []map[string]string{
		{
			"text": "You are a helpful shopping assistant for a flea market app. Answer concisely in Japanese. If the question is unrelated, politely guide the user back to the item.",
		},
		{
			"text": fmt.Sprintf("商品データ:\nタイトル: %s\n説明: %s\n価格: %d\nカテゴリ: %s", item.Title, item.Description, item.Price, item.CategorySlug),
		},
		{
			"text": fmt.Sprintf("質問: %s", req.Question),
		},
	}
	if conversationText != "" {
		parts = append(parts, map[string]string{
			"text": fmt.Sprintf("最近のDM:\n%s", conversationText),
		})
	}
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": parts,
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.5,
			"maxOutputTokens": 512,
		},
	}

	body, _ := json.Marshal(payload)
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "models/gemini-2.5-flash"
	}
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent?key=%s", model, os.Getenv("GEMINI_API_KEY"))
	resp, err := h.client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return c.JSON(http.StatusBadGateway, NewErrorResponse("upstream_error", "failed to call gemini"))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		_, _ = io.CopyN(&buf, resp.Body, 2048)
		log.Printf("gemini upstream error: status=%d url=%s body=%q", resp.StatusCode, redactKey(endpoint), buf.String())
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
	log.Printf("gemini success: model=%s answer_len=%d preview=%q", os.Getenv("GEMINI_MODEL"), len(answer), truncate(answer, 200))
	return c.JSON(http.StatusOK, map[string]string{"answer": answer})
}

func redactKey(url string) string {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return url
	}
	return strings.ReplaceAll(url, key, "REDACTED")
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (h *AIHandler) EnhanceImage(c echo.Context) error {
	uid, _ := c.Get("uid").(string)
	if uid == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("unauthorized", "missing uid"))
	}
	if h.imageClient == nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "image enhancer is not configured"))
	}
	if h.storage == nil || h.storageBucket == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "storage is not configured"))
	}

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "image is required"))
	}
	if fileHeader.Size > h.maxUploadBytes {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "image too large (max 5MB)"))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "failed to read image"))
	}
	defer file.Close()

	limited := io.LimitReader(file, h.maxUploadBytes+1)
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, limited); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "failed to read image"))
	}
	if int64(buf.Len()) > h.maxUploadBytes {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "image too large (max 5MB)"))
	}
	imageBytes := buf.Bytes()
	if len(imageBytes) == 0 {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "image is empty"))
	}

	itemIDStr := strings.TrimSpace(c.FormValue("itemId"))
	category := strings.TrimSpace(c.FormValue("category"))
	if itemIDStr != "" {
		itemID, convErr := strconv.ParseUint(itemIDStr, 10, 64)
		if convErr != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid itemId"))
		}
		item, err := h.itemRepo.FindByID(c.Request().Context(), itemID)
		if err != nil {
			return c.JSON(http.StatusNotFound, NewErrorResponse("not_found", "item not found"))
		}
		if item.CategorySlug != "" {
			category = item.CategorySlug
		}
	}

	mode := strings.TrimSpace(strings.ToLower(c.FormValue("mode")))
	if mode == "" || mode == "auto" {
		mode = mapCategoryToMode(category)
	} else {
		switch mode {
		case "fashion-look", "tech-gadget", "outdoor-gear":
			// allowed
		default:
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid mode"))
		}
	}

	background := strings.TrimSpace(strings.ToLower(c.FormValue("background")))
	if background == "" {
		background = defaultBackgroundForMode(mode)
	} else if !isValidBackground(background) {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "invalid background"))
	}

	strength := 1
	if sVal := strings.TrimSpace(c.FormValue("strength")); sVal != "" {
		parsed, convErr := strconv.Atoi(sVal)
		if convErr != nil || parsed < 0 || parsed > 2 {
			return c.JSON(http.StatusBadRequest, NewErrorResponse("bad_request", "strength must be 0,1,2"))
		}
		strength = parsed
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		exts, _ := mime.ExtensionsByType(fileHeader.Header.Get("Content-Type"))
		if len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".jpg"
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	now := time.Now().UnixMilli()
	ctx, cancel := context.WithTimeout(c.Request().Context(), 60*time.Second)
	defer cancel()

	origPath := fmt.Sprintf("items/%s/%d_orig%s", uid, now, ext)
	origURL, err := uploadWithToken(ctx, h.storage, h.storageBucket, origPath, contentType, imageBytes)
	if err != nil {
		log.Printf("upload original failed: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to store original image"))
	}

	prompt := ai.BuildEnhancePrompt(mode)
	result, err := h.imageClient.Enhance(ctx, ai.ImageEnhanceRequest{
		Image:      imageBytes,
		MimeType:   contentType,
		Prompt:     prompt,
		Strength:   strength,
		Background: background,
		Mode:       mode,
	})
	if err != nil {
		log.Printf("gemini enhance failed: %v", err)
		return c.JSON(http.StatusBadGateway, NewErrorResponse("upstream_error", "failed to enhance image"))
	}

	enhancedPath := fmt.Sprintf("items/%s/%d_enh_%s.jpg", uid, now, mode)
	enhURL, err := uploadWithToken(ctx, h.storage, h.storageBucket, enhancedPath, "image/jpeg", result.Image)
	if err != nil {
		log.Printf("upload enhanced failed: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("internal_error", "failed to store enhanced image"))
	}

	elapsed := result.ElapsedMs
	if elapsed == 0 {
		elapsed = time.Since(time.UnixMilli(now)).Milliseconds()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"originalUrl": origURL,
		"enhancedUrl": enhURL,
		"meta": map[string]interface{}{
			"mode":       mode,
			"strength":   strength,
			"background": background,
			"elapsedMs":  elapsed,
		},
	})
}

func uploadWithToken(ctx context.Context, client *storage.Client, bucket, objectPath, contentType string, data []byte) (string, error) {
	token := uuid.NewString()
	obj := client.Bucket(bucket).Object(objectPath)
	w := obj.NewWriter(ctx)
	if contentType != "" {
		w.ContentType = contentType
	}
	if w.Metadata == nil {
		w.Metadata = map[string]string{}
	}
	w.Metadata["firebaseStorageDownloadTokens"] = token

	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	escapedPath := url.PathEscape(objectPath)
	publicURL := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media&token=%s",
		bucket, escapedPath, token)
	return publicURL, nil
}

func mapCategoryToMode(category string) string {
	cat := strings.ToLower(category)
	switch {
	case strings.Contains(cat, "outdoor"), strings.Contains(cat, "camp"), strings.Contains(cat, "sports"):
		return "outdoor-gear"
	case strings.Contains(cat, "phone"), strings.Contains(cat, "pc"), strings.Contains(cat, "tablet"),
		strings.Contains(cat, "tech"), strings.Contains(cat, "electronic"), strings.Contains(cat, "camera"),
		strings.Contains(cat, "audio"):
		return "tech-gadget"
	default:
		return "fashion-look"
	}
}

func defaultBackgroundForMode(mode string) string {
	switch mode {
	case "tech-gadget":
		return "light_gray"
	case "outdoor-gear":
		return "original"
	default:
		return "white"
	}
}

func isValidBackground(bg string) bool {
	switch bg {
	case "white", "light_gray", "original":
		return true
	default:
		return false
	}
}
