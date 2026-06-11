package arquivo

import (
	"context"
	"crypto/sha256"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
)

var ti *database.TestInstance

func newTestService(tb testing.TB) (*Service, *pgxpool.Pool) {
	tb.Helper()
	pool := ti.NewPool(tb)
	storage, err := blob.NewFilesystemStorage(tb.TempDir())
	if err != nil {
		tb.Fatal(err)
	}
	// Client insert-only: sem workers nem queues, apenas para enfileirar tasks.
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		tb.Fatal(err)
	}
	return NewService(pool, storage, riverClient), pool
}

func TestMain(m *testing.M) {
	ti = database.MustTestInstance()
	code := m.Run()

	if err := ti.Close(context.Background()); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestCreateArquivo_TamanhoBytes(t *testing.T) {
	t.Parallel()
	service, _ := newTestService(t)

	const content = "conteúdo de teste"
	arquivo, err := service.CreateArquivo(t.Context(), strings.NewReader(content), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	if want := int64(len(content)); arquivo.TamanhoBytes != want {
		t.Fatalf("want TamanhoBytes %d, got %d", want, arquivo.TamanhoBytes)
	}
}

func TestCreateArquivo_TamanhoBytes_Persisted(t *testing.T) {
	t.Parallel()
	service, _ := newTestService(t)

	const content = "conteúdo persistido"
	_, err := service.CreateArquivo(t.Context(), strings.NewReader(content), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	// Lê direto do banco de dados para garantir que o tamanho foi persistido,
	// e não apenas retornado pela função.
	checksum := sha256.Sum256([]byte(content))
	row, err := service.q.GetArquivoByHashSHA256(t.Context(), checksum[:])
	if err != nil {
		t.Fatal(err)
	}

	if want := int64(len(content)); row.TamanhoBytes != want {
		t.Fatalf("want TamanhoBytes %d, got %d", want, row.TamanhoBytes)
	}
}

func TestCreateArquivo_TamanhoBytes_Empty(t *testing.T) {
	t.Parallel()
	service, _ := newTestService(t)

	// O contentType é informado explicitamente pois a detecção automática
	// exige a leitura do conteúdo.
	arquivo, err := service.CreateArquivo(t.Context(), strings.NewReader(""), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	if arquivo.TamanhoBytes != 0 {
		t.Fatalf("want TamanhoBytes 0, got %d", arquivo.TamanhoBytes)
	}
}

func TestCreateArquivo_EnqueuesExtractTask(t *testing.T) {
	t.Parallel()
	service, pool := newTestService(t)

	arq, err := service.CreateArquivo(t.Context(), strings.NewReader("conteúdo de teste"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	// A criação do arquivo deve enfileirar exatamente uma task de extração
	// de conteúdo apontando para o arquivo criado.
	job := rivertest.RequireInserted(t.Context(), t, riverpgxv5.New(pool), ExtractArgs{}, nil)
	if job.Args.ArquivoID != arq.ID {
		t.Fatalf("want ArquivoID %d, got %d", arq.ID, job.Args.ArquivoID)
	}
}

func TestCreateArquivo_AlreadyExists(t *testing.T) {
	t.Parallel()
	service, pool := newTestService(t)

	const content = "conteúdo duplicado"
	_, err := service.CreateArquivo(t.Context(), strings.NewReader(content), "text/plain")
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateArquivo(t.Context(), strings.NewReader(content), "text/plain")
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// O upload duplicado não deve enfileirar uma segunda task de extração.
	// RequireInserted falha caso exista mais de uma task do mesmo kind.
	rivertest.RequireInserted(t.Context(), t, riverpgxv5.New(pool), ExtractArgs{}, nil)
}
