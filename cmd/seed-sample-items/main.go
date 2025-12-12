package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/shinyyama/hackathon-backend/internal/config"
	"github.com/shinyyama/hackathon-backend/internal/db"
	"github.com/shinyyama/hackathon-backend/internal/model"
	"github.com/shinyyama/hackathon-backend/internal/repository"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	gdb, err := db.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}

	itemRepo := repository.NewItemRepository(gdb)

	pattern := filepath.Join("..", "front", "public", "sample-items", "*.webp")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob sample items: %w", err)
	}
	if len(paths) == 0 {
		log.Printf("no sample items found at %s", pattern)
		return nil
	}

	basePrice := 2000
	inserted, skipped := 0, 0

	for idx, p := range paths {
		filename := filepath.Base(p)
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		title := toTitle(base)

		description := fmt.Sprintf("%s - sample item for the hackathon marketplace.", title)
		price := uint(basePrice + (idx*500)%5000)
		imageURL := "/sample-items/" + filename

		exists, err := itemRepo.FindByImageURL(ctx, imageURL)
		if err != nil {
			return fmt.Errorf("check existing %s: %w", filename, err)
		}
		if exists != nil {
			skipped++
			continue
		}

		item := &model.Item{
			Title:       title,
			Description: description,
			Price:       price,
			ImageURL:    &imageURL,
		}

		if err := itemRepo.Create(ctx, item); err != nil {
			return fmt.Errorf("insert %s: %w", filename, err)
		}
		inserted++
	}

	log.Printf("seed complete: inserted=%d skipped=%d total=%d", inserted, skipped, len(paths))
	return nil
}

func toTitle(base string) string {
	normalized := strings.NewReplacer("-", " ", "_", " ").Replace(base)
	parts := strings.Fields(normalized)
	for i, p := range parts {
		parts[i] = capitalize(p)
	}
	return strings.Join(parts, " ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}
