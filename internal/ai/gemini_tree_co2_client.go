package ai

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/co2ctx"
	"google.golang.org/genai"
)

type TreeCO2Client struct {
	model      string
	httpClient *http.Client
}

func NewTreeCO2Client(httpClient *http.Client) *TreeCO2Client {
	model := os.Getenv("GEMINI_TREE_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &TreeCO2Client{model: model, httpClient: httpClient}
}

// Estimate takes title/description and image URL, calls Gemini, and returns co2 kg.
func (c *TreeCO2Client) Estimate(ctx context.Context, title, description, imageURL string) (float64, error) {
	rid := co2ctx.RID(ctx)
	itemID := co2ctx.ItemID(ctx)
	if imageURL == "" {
		return 0, fmt.Errorf("image url required")
	}
	start := time.Now()
	img, mime, err := c.fetchImage(ctx, imageURL)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_fail err=%v", rid, itemID, err)
		return 0, err
	}
	fetchDur := time.Since(start)
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=client_init err=%v", rid, itemID, err)
		return 0, err
	}

	prompt := `あなたはリユース品のCO2削減量を概算する推定器です。
入力（タイトル/説明/画像）から「新品購入を回避できた場合の推定CO2削減量(kgCO2e)」を1つだけ推定してください。
最終回答は「数値1つのみ」を必ず返してください。単位は gCO2e で考え、例: 300
それ以外の説明文や記号、改行、空白は出さないでください。
<number> は 0〜5000 の範囲、整数または小数1桁まで。不明なら 0 を返してください。`

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromText(fmt.Sprintf("タイトル: %s\n説明: %s", title, description)),
		&genai.Part{
			InlineData: &genai.Blob{
				MIMEType: mime,
				Data:     img,
			},
		},
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}
	temp := float32(0)
	config := &genai.GenerateContentConfig{
		Temperature: &temp,
	}
	genStart := time.Now()
	log.Printf("[co2] rid=%s item=%d stage=gemini_start model=%s", rid, itemID, c.model)
	res, err := client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=gemini_fail model=%s err=%v", rid, itemID, c.model, err)
		return 0, fmt.Errorf("gemini generate: %w", err)
	}
	genDur := time.Since(genStart)
	log.Printf("[co2] rid=%s item=%d stage=gemini_done model=%s genMs=%d", rid, itemID, c.model, genDur.Milliseconds())
	log.Printf("[co2] rid=%s item=%d stage=parse_start", rid, itemID)
	val, unit, err := ParseCO2WithUnit(res.Text())
	if err != nil {
		text := strings.ReplaceAll(res.Text(), "\n", " ")
		if len(text) > 80 {
			text = text[:80]
		}
		log.Printf("[co2] rid=%s item=%d stage=parse_fail len=%d text=%q err=%v", rid, itemID, len(res.Text()), text, err)
		return 0, err
	}
	log.Printf("[co2] rid=%s item=%d stage=parse_ok value=%.3f unit=%s fetchMs=%d genMs=%d totalMs=%d", rid, itemID, val, unit, fetchDur.Milliseconds(), genDur.Milliseconds(), time.Since(start).Milliseconds())
	return val, nil
}

func (c *TreeCO2Client) fetchImage(ctx context.Context, url string) ([]byte, string, error) {
	rid := co2ctx.RID(ctx)
	itemID := co2ctx.ItemID(ctx)
	fetchStart := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_start err=%v", rid, itemID, err)
		return nil, "", err
	}
	log.Printf("[co2] rid=%s item=%d stage=image_fetch_start url=%s", rid, itemID, url)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_err err=%v", rid, itemID, err)
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_status status=%d", rid, itemID, resp.StatusCode)
		return nil, "", fmt.Errorf("fetch image status %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, 5*1024*1024+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_read err=%v", rid, itemID, err)
		return nil, "", err
	}
	if len(data) > 5*1024*1024 {
		log.Printf("[co2] rid=%s item=%d stage=image_fetch_too_large bytes=%d", rid, itemID, len(data))
		return nil, "", fmt.Errorf("image too large")
	}
	mime := http.DetectContentType(data)
	log.Printf("[co2] rid=%s item=%d stage=image_fetch_done status=%d contentType=%s bytes=%d durMs=%d", rid, itemID, resp.StatusCode, mime, len(data), time.Since(fetchStart).Milliseconds())
	return data, mime, nil
}
