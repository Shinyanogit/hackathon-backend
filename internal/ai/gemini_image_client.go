package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GeminiImageClient struct {
	apiKey      string
	model       string
	httpClient  *http.Client
	contentType string
}

type ImageEnhanceRequest struct {
	Image      []byte
	MimeType   string
	Prompt     string
	Strength   int
	Background string
	Mode       string
}

type ImageEnhanceResult struct {
	Image     []byte
	ElapsedMs int64
}

func NewGeminiImageClient(apiKey, model string, httpClient *http.Client) *GeminiImageClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 55 * time.Second,
		}
	}
	if model == "" {
		model = "models/imagemodel@006"
	}
	return &GeminiImageClient{
		apiKey:      apiKey,
		model:       model,
		httpClient:  httpClient,
		contentType: "application/json",
	}
}

func (c *GeminiImageClient) Enhance(ctx context.Context, req ImageEnhanceRequest) (*ImageEnhanceResult, error) {
	if c == nil {
		return nil, errors.New("gemini client is nil")
	}
	if c.apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is not set")
	}
	if len(req.Image) == 0 {
		return nil, errors.New("image is required")
	}

	mimeType := strings.TrimSpace(req.MimeType)
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	parts := []map[string]interface{}{
		{"text": req.Prompt},
		{
			"inline_data": map[string]string{
				"mime_type": mimeType,
				"data":      base64.StdEncoding.EncodeToString(req.Image),
			},
		},
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": parts,
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"topK":            32,
			"topP":            0.8,
			"maxOutputTokens": 2048,
		},
	}

	payload, _ := json.Marshal(body)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent?key=%s",
		url.PathEscape(c.model), url.QueryEscape(c.apiKey))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", c.contentType)

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, truncate(string(resBody), 500))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData *struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(resBody, &parsed); err != nil {
		return nil, err
	}

	for _, cand := range parsed.Candidates {
		for _, part := range cand.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				img, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err == nil {
					return &ImageEnhanceResult{Image: img, ElapsedMs: elapsed}, nil
				}
			}
		}
	}

	return nil, errors.New("gemini response did not include inlineData image")
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
