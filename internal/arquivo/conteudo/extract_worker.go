package conteudo

import (
	"context"
	"errors"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/docintel"
	"github.com/automatiza-mg/seizeiro/internal/llm"
	"github.com/riverqueue/river"
)

type Extractor interface {
	ExtractConteudo(ctx context.Context, arquivoID int64) error
}

// ExtractConteudoWorker processa tasks de extração de conteúdo textual.
type ExtractConteudoWorker struct {
	extractor Extractor
	river.WorkerDefaults[arquivo.ExtractArgs]
}

func NewExtractConteudoWorker(extractor Extractor) *ExtractConteudoWorker {
	return &ExtractConteudoWorker{
		extractor: extractor,
	}
}

// Timeout limita a duração de cada tentativa do job. O PollResult do OCR já
// desiste após 5 minutos; a margem cobre o download do storage e a
// persistência do resultado.
func (w *ExtractConteudoWorker) Timeout(*river.Job[arquivo.ExtractArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *ExtractConteudoWorker) Work(ctx context.Context, job *river.Job[arquivo.ExtractArgs]) error {
	err := w.extractor.ExtractConteudo(ctx, job.Args.ArquivoID)
	if err != nil {
		if isPermanentError(err) {
			return river.JobCancel(err)
		}
		return err
	}
	return nil
}

// Indica se o erro nunca será resolvido com uma nova tentativa.
func isPermanentError(err error) bool {
	// O arquivo não existe mais.
	if errors.Is(err, arquivo.ErrNotFound) {
		return true
	}
	// O conteúdo não existe mais.
	if errors.Is(err, ErrNotFound) {
		return true
	}
	// A dimensão do embedding não casa com o schema do banco (VECTOR(1536)).
	if _, ok := errors.AsType[*llm.DimensionMismatchError](err); ok {
		return true
	}
	// Não há método de extração para o MIME resolvido.
	if _, ok := errors.AsType[*arquivo.UnsupportedError](err); ok {
		return true
	}
	// A análise de OCR falhou.
	if _, ok := errors.AsType[*docintel.AnalyzeError](err); ok {
		return true
	}
	// Erros HTTP que não podem ser resolvidos com retry.
	if se, ok := errors.AsType[*docintel.StatusError](err); ok {
		return !se.Retryable()
	}
	return false
}
