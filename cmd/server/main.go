package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/config"
	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/automatiza-mg/seizeiro/internal/postgres/migrations"
	"github.com/automatiza-mg/seizeiro/internal/sei"
	"github.com/automatiza-mg/seizeiro/internal/sei/seiws"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/automatiza-mg/seizeiro/internal/tasks"
	"github.com/automatiza-mg/seizeiro/internal/webhook"
	"github.com/jackc/pgx/v5"
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
	cfg            *config.Config
	pool           *pgxpool.Pool
	views          fs.FS
	chatbotauth    *chatbotauth.Service
	scraper        *sei.Scraper
	chatbotWebhook *webhook.Notifier
	// wsseiClients reaproveita instâncias de [*wssei.Client] por usuário do
	// chatbot, evitando reautenticar no WSSEI a cada requisição.
	wsseiClients *wsseiClientCache
	// seiws é o cliente da API SOAP legada do SEI (SeiWS.php). É opcional:
	// quando as variáveis SEI_WS_URL/SEI_SIGLA_SISTEMA/SEI_IDENTIFICACAO_SERVICO
	// não estão configuradas, ele fica nil e as rotas correspondentes devolvem
	// um erro 503.
	seiws *seiws.Client
	// arquivos baixa anexos do SEI, deduplica por SHA-256 e devolve URL.
	arquivos *arquivo.Service
	// river enfileira jobs assíncronos (download de anexos, etc.).
	river *river.Client[pgx.Tx]
}

// ArquivoURL implementa [arquivo.LinkBuilder] gerando a URL pública da rota
// interna que serve arquivos quando o storage é filesystem.
func (app *application) ArquivoURL(hash string) string {
	return fmt.Sprintf("%s/arquivos/%s", app.cfg.BaseURL, hash)
}

// ResolveByPlataforma implementa [arquivo.WSSEIResolver]. Reaproveita o
// cache de clients WSSEI por usuário do chatbot.
func (app *application) ResolveByPlataforma(ctx context.Context, plataforma, plataformaID string) (*wssei.Client, error) {
	usuario, err := app.chatbotauth.GetUsuario(ctx, plataforma, plataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return nil, arquivo.ErrUsuarioNotFound
		}
		return nil, fmt.Errorf("get usuario: %w", err)
	}
	return app.wsseiClients.Get(usuario), nil
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

	// Em produção, aplica as migrações do schema da aplicação antes de subir
	// o resto do servidor. Em dev, esperamos que o desenvolvedor rode goose
	// manualmente (conforme CONTRIBUTING.md).
	if cfg.Production {
		if err := migrationsUp(ctx, pool); err != nil {
			return fmt.Errorf("migrations up: %w", err)
		}
	}

	if err := riverUp(ctx, pool); err != nil {
		return fmt.Errorf("river up: %w", err)
	}

	encKey, err := cfg.Key()
	if err != nil {
		return fmt.Errorf("config key: %w", err)
	}

	chatAuth, err := chatbotauth.NewService(pool, encKey)
	if err != nil {
		return fmt.Errorf("chatbot auth: %w", err)
	}

	storage, err := newStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	notifier := webhook.NewNotifier(cfg.ChatbotWebhook.URL, cfg.ChatbotWebhook.Secret)

	app := &application{
		cfg:            cfg,
		pool:           pool,
		views:          os.DirFS("web/views"),
		chatbotauth:    chatAuth,
		scraper:        sei.NewScraper(cfg.SEI.BaseURL),
		chatbotWebhook: notifier,
		wsseiClients:   newWSSEIClientCache(cfg.SEI.BaseURL),
		seiws:          newSEIWSClient(cfg.SEI),
	}

	app.arquivos = arquivo.NewService(arquivo.Config{
		Pool:          pool,
		Storage:       storage,
		Links:         app,
		WSSEIResolver: app,
		URLTTL:        time.Hour,
	})

	// River com worker de download de anexo.
	workers := river.NewWorkers()
	river.AddWorker(workers, tasks.NewBaixarAnexoWorker(app.arquivos, notifier))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 5},
		},
		Workers: workers,
	})
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}
	app.river = riverClient

	if err := riverClient.Start(ctx); err != nil {
		return fmt.Errorf("river start: %w", err)
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

	// Shutdown: contexto novo de 15s para não ser cancelado pelo SIGINT.
	// Para HTTP e River em paralelo.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer stopCancel()

	return errors.Join(
		srv.Shutdown(stopCtx),
		riverClient.Stop(stopCtx),
	)
}

// newStorage cria o backend de armazenamento de acordo com a configuração:
// Azure Blob quando STORAGE_AZURE_ACCOUNT está definido, ou filesystem local
// caso contrário.
func newStorage(cfg config.Storage) (blob.Storage, error) {
	if cfg.AzureAccount != "" {
		return blob.NewAzureStorage(cfg.AzureAccount, cfg.AzureKey, cfg.AzureContainer)
	}
	return blob.NewFilesystemStorage(cfg.FilesystemRoot)
}

// newSEIWSClient cria o cliente da API SOAP legada do SEI quando todas as
// variáveis necessárias estão configuradas. Retorna nil caso contrário,
// permitindo que a aplicação suba sem essa integração.
func newSEIWSClient(cfg config.SEI) *seiws.Client {
	if cfg.WSURL == "" || cfg.SiglaSistema == "" || cfg.IdentificacaoServico == "" {
		return nil
	}
	return seiws.NewClient(seiws.Config{
		URL:                  cfg.WSURL,
		SiglaSistema:         cfg.SiglaSistema,
		IdentificacaoServico: cfg.IdentificacaoServico,
	})
}

// Aplica as migrações do River adaptando [pgxpool.Pool] para [sql.DB].
func riverUp(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrations.RiverUp(ctx, db)
}

// Aplica as migrações do schema da aplicação adaptando [pgxpool.Pool] para
// [sql.DB].
func migrationsUp(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	return migrations.Up(ctx, db)
}

// Garante que a aplicação implementa as interfaces esperadas pelo
// internal/arquivo.
var (
	_ arquivo.LinkBuilder   = (*application)(nil)
	_ arquivo.WSSEIResolver = (*application)(nil)
)
