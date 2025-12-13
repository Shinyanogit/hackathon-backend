package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shinyyama/hackathon-backend/internal/config"
	"github.com/shinyyama/hackathon-backend/internal/db"
)

type seedItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int64    `json:"price"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Images      []string `json:"images"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
}

func run() (err error) {
	ctx := context.Background()
	_ = godotenv.Load()

	seedPath, err := findSeedPath()
	if err != nil {
		return err
	}
	items, err := loadSeed(seedPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	gdb, err := db.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("sql db: %w", err)
	}

	canSeed, err := shouldSeed(ctx, sqlDB)
	if err != nil {
		return err
	}
	if !canSeed {
		log.Printf("items already exist; skipping seed (set FORCE_SEED=true to override)")
		return nil
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for idx, it := range items {
		categoryID, err := ensureCategory(ctx, tx, strings.TrimSpace(it.Category))
		if err != nil {
			return err
		}

		images := ensurePicsumImages(idx, len(it.Images))

		itemID, err := insertItem(ctx, tx, it, categoryID, images[0])
		if err != nil {
			return err
		}

		if err := insertImages(ctx, tx, itemID, images); err != nil {
			return err
		}
		if err := insertTags(ctx, tx, itemID, it.Tags); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("seeded %d items from %s", len(items), seedPath)
	return nil
}

func loadSeed(path string) ([]seedItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seed file: %w", err)
	}
	var items []seedItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	if len(items) == 0 {
		return nil, errors.New("seed file is empty")
	}
	return items, nil
}

func findSeedPath() (string, error) {
	candidates := []string{
		"items_seed_01.json",
		filepath.Join("cmd", "seed", "items_seed_01.json"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "items_seed_01.json"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("items_seed_01.json not found (checked: %s)", strings.Join(candidates, ", "))
}

func ensureCategory(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	if name == "" {
		return 0, errors.New("category name is required")
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO categories (name) VALUES (?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, name)
	if err != nil {
		return 0, fmt.Errorf("insert category %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("category last insert id: %w", err)
	}
	return id, nil
}

func ensureTag(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	if name == "" {
		return 0, errors.New("tag name is required")
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, name)
	if err != nil {
		return 0, fmt.Errorf("insert tag %q: %w", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("tag last insert id: %w", err)
	}
	return id, nil
}

func insertItem(ctx context.Context, tx *sql.Tx, item seedItem, categoryID int64, firstImage string) (int64, error) {
	title := strings.TrimSpace(item.Title)
	description := strings.TrimSpace(item.Description)
	imageURL := strings.TrimSpace(firstImage)

	res, err := tx.ExecContext(ctx,
		`INSERT INTO items (title, description, price, category_id, image_url) VALUES (?, ?, ?, ?, ?)`,
		title, description, item.Price, categoryID, imageURL,
	)
	if err != nil {
		return 0, fmt.Errorf("insert item %q: %w", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("item last insert id: %w", err)
	}
	return id, nil
}

func insertImages(ctx context.Context, tx *sql.Tx, itemID int64, images []string) error {
	for _, img := range images {
		url := strings.TrimSpace(img)
		if url == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO item_images (item_id, image_url) VALUES (?, ?)`, itemID, url); err != nil {
			return fmt.Errorf("insert image %q for item %d: %w", url, itemID, err)
		}
	}
	return nil
}

func insertTags(ctx context.Context, tx *sql.Tx, itemID int64, tags []string) error {
	seen := make(map[string]struct{})
	for _, tag := range tags {
		name := strings.TrimSpace(tag)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		tagID, err := ensureTag(ctx, tx, name)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO item_tags (item_id, tag_id) VALUES (?, ?)`, itemID, tagID); err != nil {
			return fmt.Errorf("link tag %q to item %d: %w", name, itemID, err)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func ensurePicsumImages(itemIdx int, desired int) []string {
	if desired < 1 {
		desired = 1
	}
	if desired > 3 {
		desired = 3
	}
	urls := make([]string, desired)
	for i := 0; i < desired; i++ {
		urls[i] = fmt.Sprintf("https://picsum.photos/seed/item-%d-%d/800/600", itemIdx, i+1)
	}
	return urls
}

func shouldSeed(ctx context.Context, db *sql.DB) (bool, error) {
	var cnt int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&cnt); err != nil {
		return false, fmt.Errorf("count items: %w", err)
	}
	if cnt == 0 {
		return true, nil
	}
	force := os.Getenv("FORCE_SEED")
	return strings.EqualFold(force, "true"), nil
}
