package conteudo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/docintel"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
)

// newTestWorker cria um rivertest.Worker para o ExtractConteudoWorker e uma
// transação para o ciclo de vida dos jobs, desfeita ao final do teste.
func newTestWorker(tb testing.TB, f *fixture) (*rivertest.Worker[arquivo.ExtractArgs, pgx.Tx], pgx.Tx) {
	tb.Helper()

	worker := rivertest.NewWorker(tb, riverpgxv5.New(f.pool), &river.Config{}, NewExtractConteudoWorker(f.service))

	tx, err := f.pool.Begin(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		// tb.Context() já está cancelado durante o cleanup.
		_ = tx.Rollback(context.Background())
	})

	return worker, tx
}

func TestExtractConteudoWorker_Work(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	id := f.createArquivo(t, "%PDF-1.4 conteúdo binário", "application/pdf")
	worker, tx := newTestWorker(t, f)

	result, err := worker.Work(t.Context(), t, tx, arquivo.ExtractArgs{ArquivoID: id}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.EventKind != river.EventKindJobCompleted {
		t.Fatalf("want event %q, got %q", river.EventKindJobCompleted, result.EventKind)
	}
	if result.Job.State != rivertype.JobStateCompleted {
		t.Fatalf("want state %q, got %q", rivertype.JobStateCompleted, result.Job.State)
	}

	// O conteúdo deve ter sido extraído e persistido.
	row := f.getConteudo(t, id, MetodoOCR)
	if row.Conteudo != f.ocr.text {
		t.Fatalf("want conteudo %q, got %q", f.ocr.text, row.Conteudo)
	}
}

func TestExtractConteudoWorker_Work_PermanentError(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.ocr.analyzeErr = &docintel.AnalyzeError{Status: docintel.StatusFailed}

	id := f.createArquivo(t, "%PDF-1.4 conteúdo binário", "application/pdf")
	worker, tx := newTestWorker(t, f)

	// Erros de cancelamento não são retornados como erro pelo rivertest.
	result, err := worker.Work(t.Context(), t, tx, arquivo.ExtractArgs{ArquivoID: id}, nil)
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

func TestExtractConteudoWorker_Work_TransientError(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.ocr.analyzeErr = &docintel.StatusError{StatusCode: http.StatusInternalServerError}

	id := f.createArquivo(t, "%PDF-1.4 conteúdo binário", "application/pdf")
	worker, tx := newTestWorker(t, f)

	// Erros transitórios são retornados pelo rivertest e o job fica
	// agendado para retry.
	result, err := worker.Work(t.Context(), t, tx, arquivo.ExtractArgs{ArquivoID: id}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.EventKind != river.EventKindJobFailed {
		t.Fatalf("want event %q, got %q", river.EventKindJobFailed, result.EventKind)
	}
	// O job deve permanecer elegível para retry: "available" quando o
	// scheduled_at do backoff já passou, "retryable" quando está no futuro.
	if result.Job.State != rivertype.JobStateAvailable && result.Job.State != rivertype.JobStateRetryable {
		t.Fatalf("want state %q or %q, got %q",
			rivertype.JobStateAvailable, rivertype.JobStateRetryable, result.Job.State)
	}
}

func TestExtractConteudoWorker_Work_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	worker, tx := newTestWorker(t, f)

	result, err := worker.Work(t.Context(), t, tx, arquivo.ExtractArgs{ArquivoID: 123456789}, nil)
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

func TestIsPermanent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "arquivo não encontrado",
			err:  arquivo.ErrNotFound,
			want: true,
		},
		{
			name: "arquivo não encontrado (wrapped)",
			err:  fmt.Errorf("get arquivo: %w", arquivo.ErrNotFound),
			want: true,
		},
		{
			name: "mime type não suportado",
			err:  &arquivo.UnsupportedError{ContentType: "video/mp4"},
			want: true,
		},
		{
			name: "análise falhou",
			err:  &docintel.AnalyzeError{Status: docintel.StatusFailed},
			want: true,
		},
		{
			name: "status HTTP 400",
			err:  &docintel.StatusError{StatusCode: http.StatusBadRequest},
			want: true,
		},
		{
			name: "status HTTP 429 (retryable)",
			err:  &docintel.StatusError{StatusCode: http.StatusTooManyRequests},
			want: false,
		},
		{
			name: "status HTTP 500 (retryable)",
			err:  &docintel.StatusError{StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "erro genérico",
			err:  errors.New("network unreachable"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isPermanentError(tt.err); got != tt.want {
				t.Fatalf("isPermanent(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
