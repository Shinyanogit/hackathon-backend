package ai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

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
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
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
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}
	temp := float32(0)
	config := &genai.GenerateContentConfig{
		Temperature: &temp,
	}
	res, err := client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return 0, fmt.Errorf("gemini generate: %w", err)
	}
	rawText := res.Text()
	val, _, err := ParseCO2WithUnit(rawText)
	if err != nil {
		text := strings.ReplaceAll(rawText, "\n", " ")
		if len(text) > 80 {
			text = text[:80]
		}
		return 0, err
	}
	return val, nil
}
