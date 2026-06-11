package conteudo

import (
	"context"
	"errors"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/llm"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
)

// newChunkTestWorker cria um rivertest.Worker para o ChunkConteudoWorker e uma
// transação para o ciclo de vida dos jobs, desfeita ao final do teste.
func newChunkTestWorker(tb testing.TB, f *fixture) (*rivertest.Worker[arquivo.ChunkArgs, pgx.Tx], pgx.Tx) {
	tb.Helper()

	worker := rivertest.NewWorker(tb, riverpgxv5.New(f.pool), &river.Config{}, NewChunkConteudoWorker(f.service))

	tx, err := f.pool.Begin(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})

	return worker, tx
}

func TestChunkConteudoWorker_Work(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	conteudoID := f.extractAndGetConteudoID(t, "texto puro para chunking", "text/plain")
	worker, tx := newChunkTestWorker(t, f)

	result, err := worker.Work(t.Context(), t, tx, arquivo.ChunkArgs{ConteudoID: conteudoID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.EventKind != river.EventKindJobCompleted {
		t.Fatalf("want event %q, got %q", river.EventKindJobCompleted, result.EventKind)
	}
	if result.Job.State != rivertype.JobStateCompleted {
		t.Fatalf("want state %q, got %q", rivertype.JobStateCompleted, result.Job.State)
	}
}

func TestChunkConteudoWorker_Work_PermanentError(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	// Dimensão divergente do schema VECTOR(1536) é um erro permanente.
	f.embedder.err = &llm.DimensionMismatchError{Expected: 1536, Got: 256}

	conteudoID := f.extractAndGetConteudoID(t, "texto puro para chunking", "text/plain")
	worker, tx := newChunkTestWorker(t, f)

	result, err := worker.Work(t.Context(), t, tx, arquivo.ChunkArgs{ConteudoID: conteudoID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.EventKind != river.EventKindJobCancelled {
		t.Fatalf("want event %q, got %q", river.EventKindJobCancelled, result.EventKind)
	}
	if result.Job.State != rivertype.JobStateCancelled {
		t.Fatalf("want state %q, got %q", rivertype.JobStateCancelled, result.Job.State)
	}
}

func TestChunkConteudoWorker_Work_TransientError(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.embedder.err = errors.New("api temporarily unavailable")

	conteudoID := f.extractAndGetConteudoID(t, "texto puro para chunking", "text/plain")
	worker, tx := newChunkTestWorker(t, f)

	result, err := worker.Work(t.Context(), t, tx, arquivo.ChunkArgs{ConteudoID: conteudoID}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.EventKind != river.EventKindJobFailed {
		t.Fatalf("want event %q, got %q", river.EventKindJobFailed, result.EventKind)
	}
	if result.Job.State != rivertype.JobStateAvailable && result.Job.State != rivertype.JobStateRetryable {
		t.Fatalf("want state %q or %q, got %q",
			rivertype.JobStateAvailable, rivertype.JobStateRetryable, result.Job.State)
	}
}
