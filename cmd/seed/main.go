package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/shinyyama/hackathon-backend/internal/config"
	"github.com/shinyyama/hackathon-backend/internal/db"
)

type seedItem struct {
	Title        string
	Description  string
	Price        int64
	CategorySlug string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
}

func run() (err error) {
	ctx := context.Background()
	_ = godotenv.Load()

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

	items := buildSeedItems()

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

	if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE items`); err != nil {
		return fmt.Errorf("truncate items: %w", err)
	}

	for idx, it := range items {
		imageURL := picsumURL(it.CategorySlug, idx+1, 1)

		if _, err := insertItem(ctx, tx, it, imageURL); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("seeded %d items", len(items))
	return nil
}

func buildSeedItems() []seedItem {
	type cat struct {
		Slug   string
		Titles []string
		Price  int64
	}
	categories := []cat{
		{Slug: "fashion", Price: 4200, Titles: []string{"リラックスフィットフーディ", "オーガニックコットンTシャツ", "デニムクラシックジーンズ", "ライトナイロンパーカー"}},
		{Slug: "beauty-cosmetics", Price: 3600, Titles: []string{"モイスチャーセラム", "シアーティントリップ", "ナチュラルUVミルク", "バランシングトナー"}},
		{Slug: "phones-tablets-pcs", Price: 24000, Titles: []string{"14インチモバイルノート", "軽量タブレット64GB", "ワイヤレスメカニカルキーボード", "静音ワイヤレスマウス"}},
		{Slug: "home-interior", Price: 7800, Titles: []string{"無垢材サイドテーブル", "コットンラグ 140x200", "スタッキングシェルフ", "フェイクグリーンポトス"}},
		{Slug: "gaming-goods", Price: 8800, Titles: []string{"ワイヤレスゲームパッド", "ゲーミングヘッドセット", "大型マウスパッド", "メカニカルキーボードRGB"}},
		{Slug: "books-magazines-comics", Price: 1400, Titles: []string{"SF小説アンソロジー", "旅雑誌最新号", "ビジネス書まとめ読み", "コミック新装版"}},
		{Slug: "sports", Price: 6200, Titles: []string{"ランニングシューズ", "吸汗速乾Tシャツ", "トレーニングマット", "ステンレスボトル"}},
		{Slug: "outdoor-travel", Price: 9200, Titles: []string{"コンパクトチェア", "ダウンブランケット", "チタンマグセット", "バックパック28L"}},
		{Slug: "kitchen-daily", Price: 3200, Titles: []string{"セラミックフライパン", "ステンレスボトル", "二重ガラスマグ", "ウッドカッティングボード"}},
		{Slug: "food-drink", Price: 2400, Titles: []string{"シングルオリジンコーヒー豆", "クラフトティーアソート", "グルテンフリーパスタ", "ダークチョコレートセット"}},
		{Slug: "toys-hobbies", Price: 4800, Titles: []string{"ブロックキット中級", "カードゲーム拡張セット", "プラモデルスターター", "パズル1000ピース"}},
		{Slug: "health-fitness", Price: 5400, Titles: []string{"フォームローラー", "ヨガブロック2個セット", "エクササイズバンド", "プロテインシェイカー"}},
		{Slug: "baby-kids", Price: 3600, Titles: []string{"オーガニックベビーケット", "シリコンスタイ", "ソフトスニーカー キッズ", "知育ブロックセット"}},
		{Slug: "pets", Price: 2800, Titles: []string{"コーデュラ首輪 M", "シリコン給水ボトル", "おもちゃロープ3本セット", "クールマット"}},
		{Slug: "automotive", Price: 6400, Titles: []string{"車内用スマホホルダー", "コンパクト掃除機12V", "折りたたみ収納ボックス", "マイクロファイバークロスセット"}},
		{Slug: "music", Price: 7600, Titles: []string{"ワイヤレスイヤホン", "エントリーオーディオインターフェース", "コンデンサーマイク", "スタジオモニターヘッドホン"}},
		{Slug: "camera-photo", Price: 12800, Titles: []string{"ミラーレス用単焦点レンズ", "カメラ用スリングバッグ", "カーボントラベルトライポッド"}},
		{Slug: "office-supplies", Price: 2600, Titles: []string{"人間工学マウスパッド", "A4ドキュメントスタンド", "ワイヤレステンキー", "LEDデスクライト"}},
		{Slug: "diy-tools", Price: 5800, Titles: []string{"コードレスドライバー", "マルチツールセット", "安全ゴーグル", "作業グローブ"}},
		{Slug: "collectibles", Price: 7200, Titles: []string{"ビンテージポスター複製", "コレクタブルフィギュア", "限定トレカスリーブ"}},
		{Slug: "art-crafts", Price: 4200, Titles: []string{"アクリル絵具セット", "キャンバスパネル3枚", "カリグラフィーペンセット", "水彩紙スケッチブック"}},
		{Slug: "others", Price: 3000, Titles: []string{"ケーブルオーガナイザー", "マルチポーチ", "トラベルアダプター", "ノイズリダクション耳栓"}},
	}

	var items []seedItem
	for _, c := range categories {
		for i, t := range c.Titles {
			price := c.Price + int64((i+1)*100)
			desc := fmt.Sprintf("%s（%s）。新品に近い自宅保管品です。即購入OK、返品不可。", t, c.Slug)
			items = append(items, seedItem{
				Title:        t,
				Description:  desc,
				Price:        price,
				CategorySlug: c.Slug,
			})
		}
	}
	return items
}

func insertItem(ctx context.Context, tx *sql.Tx, item seedItem, imageURL string) (int64, error) {
	title := strings.TrimSpace(item.Title)
	description := strings.TrimSpace(item.Description)
	category := strings.TrimSpace(item.CategorySlug)
	imageURL = strings.TrimSpace(imageURL)

	res, err := tx.ExecContext(ctx,
		`INSERT INTO items (title, description, price, category_slug, image_url) VALUES (?, ?, ?, ?, ?)`,
		title, description, item.Price, category, imageURL,
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
	// item_images テーブルがないためダミー
	return nil
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

func picsumURL(slug string, itemIndex int, k int) string {
	return fmt.Sprintf("https://picsum.photos/seed/%s-%d-%d/600/600", slug, itemIndex, k)
}
