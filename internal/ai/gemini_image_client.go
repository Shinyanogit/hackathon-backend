package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

type GeminiImageClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
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
	return &GeminiImageClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
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

	modelName := strings.TrimSpace(c.model)
	if modelName == "" {
		modelName = "gemini-2.5-flash-image"
	}

	mimeType := strings.TrimSpace(req.MimeType)
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 55 * time.Second}
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     c.apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init genai client: %w", err)
	}

	parts := []*genai.Part{
		genai.NewPartFromText(req.Prompt),
		{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     req.Image,
			},
		},
	}
	contents := []*genai.Content{genai.NewContentFromParts(parts, genai.RoleUser)}

	start := time.Now()
	resp, err := client.Models.GenerateContent(ctx, modelName, contents, nil)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}

	for _, cand := range resp.Candidates {
		if cand == nil || cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
				continue
			}
			return &ImageEnhanceResult{Image: part.InlineData.Data, ElapsedMs: elapsed}, nil
		}
	}

	return nil, errors.New("gemini response did not include inlineData image")
}
