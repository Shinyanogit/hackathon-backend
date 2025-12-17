package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"
	"log"
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
	if imageURL == "" {
		return 0, fmt.Errorf("image url required")
	}
	start := time.Now()
	img, mime, err := c.fetchImage(ctx, imageURL)
	if err != nil {
		return 0, err
	}
	fetchDur := time.Since(start)
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return 0, err
	}

	prompt := `あなたはリユース品のCO2削減量を概算する推定器です。
入力（タイトル/説明/画像）から「新品購入を回避できた場合の推定CO2削減量(kgCO2e)」を1つだけ推定してください。
出力は次の形式の“数値だけ”にしてください: $<number>$
それ以外の文字は一切出さないでください（説明文、単位、改行、空白も不要）。
<number> は 0〜5000 の範囲、整数または小数1桁まで。
不明なら $0$。`

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
	res, err := client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return 0, err
	}
	genDur := time.Since(genStart)
	val, err := ParseCO2(res.Text())
	if err != nil {
		return 0, err
	}
	log.Printf("[co2] image fetch=%v gen=%v total=%v", fetchDur, genDur, time.Since(start))
	return val, nil
}

func (c *TreeCO2Client) fetchImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch image status %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, 5*1024*1024+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", err
	}
	if len(data) > 5*1024*1024 {
		return nil, "", fmt.Errorf("image too large")
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/jpeg"
	}
	return data, mime, nil
}
