// cmd/seed-content — Seeds mkt_listings from public-domain content sources.
//
// Usage:
//
//	go run ./cmd/seed-content/ [--db homegrown_agent] [--target 200] [--dry-run]
//
// Flags:
//
//	--db string      Target database name (default: homegrown_agent)
//	--target int     Number of listings to insert (default: 200)
//	--dry-run        Print would-be inserts to stdout; no DB writes
//
// R2 uploads (required for the CK Foundation ingester):
//
//	OBJECT_STORAGE_ENDPOINT         e.g. https://<account>.r2.cloudflarestorage.com
//	OBJECT_STORAGE_BUCKET           bucket name
//	OBJECT_STORAGE_ACCESS_KEY_ID    R2 access key
//	OBJECT_STORAGE_SECRET_ACCESS_KEY R2 secret key
//	OBJECT_STORAGE_REGION           defaults to "auto"
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	mediaAdapters "github.com/homegrown-academy/homegrown-academy/internal/media/adapters"
	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/sources"
)

func main() {
	dbName := flag.String("db", "homegrown_agent", "target database name")
	target := flag.Int("target", 200, "number of listings to insert per source")
	dryRun := flag.Bool("dry-run", false, "print listings to stdout without writing to DB")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	cfg := sources.Config{
		DryRun:    *dryRun,
		Target:    *target,
		PageDelay: time.Second,
	}

	if !*dryRun {
		db, err := openDB(*dbName)
		if err != nil {
			slog.Error("open db", "err", err)
			os.Exit(1)
		}
		sqlDB, err := db.DB()
		if err != nil {
			slog.Error("get sql.DB", "err", err)
			os.Exit(1)
		}
		defer func() { _ = sqlDB.Close() }()
		cfg.DB = db
	}

	// R2 uploader — optional; CK Foundation and thumbnail ingesters skip when not configured.
	pdfUploader, imgStorage := buildUploaders(ctx)
	cfg.Storage = pdfUploader
	cfg.ThumbnailStorage = imgStorage

	// Ensure the platform publisher + creator rows exist before any ingester runs.
	if !*dryRun {
		if err := sources.EnsurePlatformEntities(cfg.DB); err != nil {
			slog.Error("ensure platform entities", "err", err)
			os.Exit(1)
		}
	}

	// ── Gutenberg (link-out, no R2) ───────────────────────────────────────────
	n, err := sources.RunGutenberg(ctx, cfg)
	if err != nil {
		slog.Error("gutenberg ingester failed", "err", err)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Printf("dry-run complete: %d gutenberg listings would be inserted\n", n)
	} else {
		slog.Info("gutenberg complete", "inserted", n)
	}

	// ── Core Knowledge Foundation (R2-hosted PDFs) ───────────────────────────
	ckN, ckErr := sources.RunCKFoundation(ctx, cfg)
	if ckErr != nil {
		slog.Error("ck_foundation ingester failed", "err", ckErr)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Printf("dry-run complete: %d ck_foundation listings would be inserted\n", ckN)
	} else {
		slog.Info("ck_foundation complete", "inserted", ckN)
	}

	// ── Thumbnail fetcher (Wikimedia / Smithsonian → R2) ─────────────────────
	thumbN, thumbErr := sources.RunThumbnailFetcher(ctx, cfg)
	if thumbErr != nil {
		slog.Error("thumbnail fetcher failed", "err", thumbErr)
		os.Exit(1)
	}
	if *dryRun {
		fmt.Printf("dry-run complete: %d listings would receive thumbnails\n", thumbN)
	} else {
		slog.Info("thumbnail fetcher complete", "updated", thumbN)
	}

	slog.Info("content seeding complete",
		"gutenberg", n, "ck_foundation", ckN, "thumbnails", thumbN)
}

// buildUploaders constructs the PDF uploader and raw image storage from OBJECT_STORAGE_* env vars.
// Returns (nil, nil) when the required env vars are absent — callers that need R2 will log and skip.
func buildUploaders(ctx context.Context) (sources.Uploader, sources.ThumbnailUploader) {
	endpoint := os.Getenv("OBJECT_STORAGE_ENDPOINT")
	bucket := os.Getenv("OBJECT_STORAGE_BUCKET")
	accessKey := os.Getenv("OBJECT_STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("OBJECT_STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("OBJECT_STORAGE_REGION")
	if region == "" {
		region = "auto"
	}

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		slog.Warn("OBJECT_STORAGE_* env vars not fully set — R2 uploads disabled")
		return nil, nil
	}

	stor, err := mediaAdapters.NewS3StorageAdapter(ctx, mediaAdapters.S3Config{
		Endpoint:        endpoint,
		Region:          region,
		Bucket:          bucket,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	})
	if err != nil {
		slog.Error("init R2 adapter", "err", err)
		return nil, nil
	}

	slog.Info("R2 uploader initialised", "bucket", bucket, "endpoint", maskEndpoint(endpoint))
	// stor satisfies sources.ThumbnailUploader (PutObject + PresignedGet) via structural typing.
	return newR2Uploader(stor), stor
}

// maskEndpoint returns a loggable version of the R2 endpoint with the account ID redacted.
func maskEndpoint(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "[invalid]"
	}
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) > 0 {
		parts[0] = "***"
	}
	u.Host = strings.Join(parts, ".")
	return u.String()
}

func deriveURL(dbName string) (string, error) {
	envMap, _ := godotenv.Read(".env")
	base := envMap["DATABASE_URL"]
	if base == "" {
		base = "postgres://homegrown:homegrown@localhost:5932/homegrown"
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

func openDB(dbName string) (*gorm.DB, error) {
	dsn, err := deriveURL(dbName)
	if err != nil {
		return nil, err
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}
