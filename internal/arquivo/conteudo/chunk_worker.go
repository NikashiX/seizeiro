package conteudo

import (
	"context"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/riverqueue/river"
)

type Chunker interface {
	ChunkConteudo(ctx context.Context, conteudoID int64) error
}

// ChunkConteudoWorker processa tasks de chunking e embedding de conteúdo.
type ChunkConteudoWorker struct {
	chunker Chunker
	river.WorkerDefaults[arquivo.ChunkArgs]
}

func NewChunkConteudoWorker(chunker Chunker) *ChunkConteudoWorker {
	return &ChunkConteudoWorker{chunker: chunker}
}

// Timeout limita a duração de cada tentativa: o embedding de um documento
// grande exige várias chamadas à API em lotes.
func (w *ChunkConteudoWorker) Timeout(*river.Job[arquivo.ChunkArgs]) time.Duration {
	return 5 * time.Minute
}

func (w *ChunkConteudoWorker) Work(ctx context.Context, job *river.Job[arquivo.ChunkArgs]) error {
	err := w.chunker.ChunkConteudo(ctx, job.Args.ConteudoID)
	if err != nil {
		if isPermanentError(err) {
			return river.JobCancel(err)
		}
		return err
	}
	return nil
}
