package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/caarlos0/env/v9"
	"github.com/google/uuid"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	GeminiAPIKey   string `env:"GEMINI_API_KEY,required"`
	GeminiModel    string `env:"GEMINI_MODEL" envDefault:"models/imagemodel@006"`
	StorageBucket  string `env:"STORAGE_BUCKET,required"`
	DBHost         string `env:"DB_HOST,required"`
	DBUser         string `env:"DB_USER,required"`
	DBPassword     string `env:"DB_PASSWORD,required"`
	DBName         string `env:"DB_NAME,required"`
	DBPort         string `env:"DB_PORT" envDefault:"3306"`
	TimeoutSeconds int    `env:"TIMEOUT_SECONDS" envDefault:"300"`
	UpdateImages   bool   `env:"UPDATE_IMAGES" envDefault:"false"`
	ForceSeed      bool   `env:"FORCE_SEED" envDefault:"false"`
}

type Prompt struct {
	Slug   string
	Prompt string
}

type geminiResponse struct {
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

func main() {
	ctx := context.Background()
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to parse env: %v", err)
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	db, err := connectDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql db: %v", err)
	}
	defer sqlDB.Close()

	log.Printf("gemini model: %s", cfg.GeminiModel)
	log.Printf("gemini endpoint: https://generativelanguage.googleapis.com/v1beta/%s:generateContent", cfg.GeminiModel)
	log.Printf("gemini api key set: %v", cfg.GeminiAPIKey != "")

	if err := db.AutoMigrate(&model.Item{}); err != nil {
		log.Printf("warn: automigrate failed: %v", err)
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	defer storageClient.Close()

	if cfg.UpdateImages {
		if err := updateExistingItems(ctx, cfg, db, storageClient); err != nil {
			log.Fatalf("update existing items failed: %v", err)
		}
	} else {
		prompts := []Prompt{
			{Slug: "fashion-look", Prompt: "Product-style photo of a folded beige cardigan on a clean white background, soft daylight, minimal shadows."},
			{Slug: "tech-gadget", Prompt: "Sleek smartphone and laptop flatlay on a light wooden desk, airy lighting, modern aesthetic."},
			{Slug: "outdoor-gear", Prompt: "Neatly arranged camping gear (backpack, lantern, boots) on rustic wood, warm daylight, crisp focus."},
		}

		for _, p := range prompts {
			log.Printf("processing slug=%s", p.Slug)
			imageBytes, err := generateImage(ctx, cfg.GeminiAPIKey, cfg.GeminiModel, p)
			if err != nil {
				log.Printf("gemini failed (%s), fallback to placeholder: %v", p.Slug, err)
				imageBytes, err = fetchPlaceholder(ctx, p.Slug)
				if err != nil {
					log.Fatalf("failed to fetch placeholder: %v", err)
				}
			}

			path := fmt.Sprintf("items/sample/%s.png", p.Slug)
			publicURL, err := uploadWithToken(ctx, storageClient, cfg.StorageBucket, path, imageBytes)
			if err != nil {
				log.Fatalf("upload failed for %s: %v", p.Slug, err)
			}

			if err := upsertItem(ctx, db, p.Slug, publicURL); err != nil {
				log.Fatalf("upsert failed for %s: %v", p.Slug, err)
			}

			log.Printf("done slug=%s url=%s", p.Slug, publicURL)
		}
	}

	log.Println("seed-images completed successfully")
}

func connectDB(cfg Config) (*gorm.DB, error) {
	var dsn string
	if strings.HasPrefix(cfg.DBHost, "/cloudsql/") {
		log.Printf("db connect via unix socket: %s", cfg.DBHost)
		dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBName)
	} else {
		log.Printf("db connect via tcp: %s:%s", cfg.DBHost, cfg.DBPort)
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	}
	return gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func generateImage(ctx context.Context, apiKey, model string, p Prompt) ([]byte, error) {
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": p.Prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"mimeType": "image/png",
		},
	}

	body, _ := json.Marshal(reqBody)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/%s:generateContent?key=%s",
		url.PathEscape(model), url.QueryEscape(apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(b))
	}

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}

	for _, c := range gr.Candidates {
		for _, part := range c.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				return base64.StdEncoding.DecodeString(part.InlineData.Data)
			}
		}
	}

	return nil, errors.New("no inlineData found in gemini response")
}

func fetchPlaceholder(ctx context.Context, seed string) ([]byte, error) {
	url := fmt.Sprintf("https://picsum.photos/seed/%s/800/600", url.PathEscape(seed))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("placeholder status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func uploadWithToken(ctx context.Context, client *storage.Client, bucketName, objectPath string, data []byte) (string, error) {
	token := uuid.NewString()
	obj := client.Bucket(bucketName).Object(objectPath)
	w := obj.NewWriter(ctx)
	w.ContentType = "image/png"
	w.Metadata = map[string]string{
		"firebaseStorageDownloadTokens": token,
	}
	if _, err := w.Write(data); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	escapedPath := url.PathEscape(objectPath)
	publicURL := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media&token=%s",
		bucketName, escapedPath, token)
	return publicURL, nil
}

func upsertItem(ctx context.Context, db *gorm.DB, slug, imageURL string) error {
	var existing model.Item
	err := db.WithContext(ctx).Where("title = ?", slug).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		desc := fmt.Sprintf("Sample item image for %s", slug)
		item := model.Item{
			Title:        slug,
			Description:  desc,
			Price:        0,
			CategorySlug: "sample",
			ImageURL:     &imageURL,
		}
		return db.WithContext(ctx).Create(&item).Error
	}

	existing.ImageURL = &imageURL
	return db.WithContext(ctx).Save(&existing).Error
}

func updateExistingItems(ctx context.Context, cfg Config, db *gorm.DB, storageClient *storage.Client) error {
	var items []model.Item
	q := db.WithContext(ctx).Model(&model.Item{})
	if !cfg.ForceSeed {
		q = q.Where("image_url LIKE ?", "https://picsum.photos/%")
	}
	if err := q.Find(&items).Error; err != nil {
		return err
	}
	log.Printf("update mode: target items=%d (force=%v)", len(items), cfg.ForceSeed)
	for _, it := range items {
		log.Printf("[item %d] start title=%s", it.ID, it.Title)
		prompt := Prompt{
			Slug: fmt.Sprintf("item-%d", it.ID),
			Prompt: fmt.Sprintf("Product photo for an online marketplace item titled '%s' in category '%s'. Clean studio lighting, simple background, high resolution.",
				it.Title, it.CategorySlug),
		}

		imageBytes, err := generateImage(ctx, cfg.GeminiAPIKey, cfg.GeminiModel, prompt)
		if err != nil {
			log.Printf("[item %d] gemini failed, fallback to picsum: %v", it.ID, err)
			imageBytes, err = fetchPlaceholder(ctx, prompt.Slug)
			if err != nil {
				log.Printf("[item %d] fallback failed: %v", it.ID, err)
				continue
			}
		} else {
			log.Printf("[item %d] gemini success", it.ID)
		}

		path := fmt.Sprintf("items/sample/%d.png", it.ID)
		publicURL, err := uploadWithToken(ctx, storageClient, cfg.StorageBucket, path, imageBytes)
		if err != nil {
			log.Printf("[item %d] upload failed: %v", it.ID, err)
			continue
		}
		log.Printf("[item %d] upload success: %s", it.ID, publicURL)

		if err := db.WithContext(ctx).Model(&model.Item{}).Where("id = ?", it.ID).Update("image_url", publicURL).Error; err != nil {
			log.Printf("[item %d] db update failed: %v", it.ID, err)
			continue
		}
		log.Printf("[item %d] db update success", it.ID)
	}
	return nil
}
