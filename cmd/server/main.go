package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/arquivo/conteudo"
	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/config"
	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/automatiza-mg/seizeiro/internal/docintel"
	"github.com/automatiza-mg/seizeiro/internal/llm"
	"github.com/automatiza-mg/seizeiro/internal/mailer"
	"github.com/automatiza-mg/seizeiro/internal/postgres/migrations"
	"github.com/automatiza-mg/seizeiro/internal/sei"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	_ = godotenv.Load()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type application struct {
	cfg         *config.Config
	pool        *pgxpool.Pool
	views       fs.FS
	chatbotauth *chatbotauth.Service
	scraper     *sei.Scraper
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.NewFromEnv()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	pool, err := database.New(ctx, cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	if err := riverUp(ctx, pool); err != nil {
		return fmt.Errorf("river up: %w", err)
	}

	embedder, err := llm.NewOpenAIEmbedder(llm.OpenAIParams{
		APIKey:     cfg.OpenAI.APIKey,
		BaseURL:    cfg.OpenAI.BaseURL,
		Model:      cfg.OpenAI.EmbeddingModel,
		Dimensions: cfg.OpenAI.EmbeddingDimensions,
		BatchSize:  cfg.OpenAI.EmbeddingBatchSize,
	})
	if err != nil {
		return fmt.Errorf("embedder: %w", err)
	}

	tokenCounter, err := llm.NewTokenCounter(cfg.OpenAI.EmbeddingModel)
	if err != nil {
		return fmt.Errorf("token counter: %w", err)
	}

	storage, err := newStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	ocr := docintel.NewClient(cfg.DocIntel.Endpoint, cfg.DocIntel.Key)

	smtpMailer, err := mailer.NewSMTPMailer(mailer.SMTPConfig{
		User:        cfg.SMTP.User,
		Password:    cfg.SMTP.Password,
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		FromAddress: cfg.SMTP.FromAddress,
	})
	if err != nil {
		return fmt.Errorf("smtp mailer: %w", err)
	}

	workers := river.NewWorkers()
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}

	conteudoService := conteudo.NewService(pool, ocr, storage, embedder, tokenCounter, riverClient)
	_ = arquivo.NewService(pool, storage, riverClient)

	river.AddWorker(workers, conteudo.NewExtractConteudoWorker(conteudoService))
	river.AddWorker(workers, conteudo.NewChunkConteudoWorker(conteudoService))
	river.AddWorker(workers, mailer.NewWorker(smtpMailer))

	if err := riverClient.Start(ctx); err != nil {
		return fmt.Errorf("river start: %w", err)
	}

	encKey, err := cfg.Key()
	if err != nil {
		return fmt.Errorf("config key: %w", err)
	}

	chatAuth, err := chatbotauth.NewService(pool, encKey)
	if err != nil {
		return fmt.Errorf("chatbot auth: %w", err)
	}

	app := &application{
		cfg:         cfg,
		pool:        pool,
		views:       os.DirFS("web/views"),
		chatbotauth: chatAuth,
		scraper:     sei.NewScraper(cfg.SEI.BaseURL),
	}

	srv := &http.Server{
		Addr:    ":4000",
		Handler: app.routes(),
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	if err := srv.Shutdown(stopCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}

	if err := riverClient.Stop(stopCtx); err != nil {
		return fmt.Errorf("river stop: %w", err)
	}

	return nil
}

// Cria o backend de armazenamento de acordo com a configuração.
func newStorage(cfg config.Storage) (blob.Storage, error) {
	if cfg.AzureAccount != "" {
		return blob.NewAzureStorage(cfg.AzureAccount, cfg.AzureContainer)
	}
	return blob.NewFilesystemStorage(cfg.FilesystemRoot)
}

// Aplica as migrações do River adaptando [pgxpool.Pool] para [sql.DB].
func riverUp(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrations.RiverUp(ctx, db)
}
