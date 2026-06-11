package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"golang.org/x/sync/errgroup"
)

const (
	defaultBatchSize = 256
	// maxConcurrentBatches limita o número de lotes processados em paralelo.
	maxConcurrentBatches = 3
)

var _ Embedder = (*OpenAIEmbedder)(nil)

// OpenAIEmbedder é um [Embedder] que gera embeddings usando a API da OpenAI.
type OpenAIEmbedder struct {
	client     openai.Client
	model      string
	dimensions int
	batchSize  int
}

// OpenAIParams agrupa as configurações para criar um [OpenAIEmbedder].
type OpenAIParams struct {
	// APIKey é a chave de API da OpenAI. Obrigatória.
	APIKey string
	// Model é o modelo de embedding (ex: "text-embedding-3-small"). Obrigatório.
	Model string
	// Dimensions é a dimensão esperada dos embeddings, que deve casar com o schema
	// do banco (ex: 1536). Obrigatória.
	Dimensions int
	// BatchSize limita a quantidade de textos por requisição. Quando zero, usa um
	// valor padrão.
	BatchSize int
	BaseURL   string
}

// Valida os parâmetros de forma defensiva, retornando [ErrInvalidEmbedderConfig]
// com contexto quando alguma condição não é satisfeita.
func (p OpenAIParams) validate() error {
	switch {
	case p.APIKey == "":
		return fmt.Errorf("%w: api key is required", ErrInvalidEmbedderConfig)
	case p.Model == "":
		return fmt.Errorf("%w: model is required", ErrInvalidEmbedderConfig)
	case p.Dimensions <= 0:
		return fmt.Errorf("%w: dimensions must be positive", ErrInvalidEmbedderConfig)
	case p.BatchSize < 0:
		return fmt.Errorf("%w: batch size must not be negative", ErrInvalidEmbedderConfig)
	}
	return nil
}

// NewOpenAIEmbedder cria um [OpenAIEmbedder] a partir de params.
//
// Retorna [ErrInvalidEmbedderConfig] caso os parâmetros sejam inválidos.
func NewOpenAIEmbedder(params OpenAIParams) (*OpenAIEmbedder, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	opts := []option.RequestOption{option.WithAPIKey(params.APIKey)}
	if params.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(params.BaseURL))
	}

	batchSize := params.BatchSize
	if batchSize == 0 {
		batchSize = defaultBatchSize
	}

	return &OpenAIEmbedder{
		client:     openai.NewClient(opts...),
		model:      params.Model,
		dimensions: params.Dimensions,
		batchSize:  batchSize,
	}, nil
}

// EmbedDocuments gera um embedding para cada texto, preservando a ordem de entrada.
//
// Os textos são enviados em lotes para reduzir o número de requisições, e os lotes
// são processados em paralelo (até [maxConcurrentBatches] simultâneos). Retorna
// [*DimensionMismatchError] caso algum embedding tenha dimensão diferente da configurada.
func (e *OpenAIEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	result := make([][]float32, len(texts))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentBatches)

	for start := 0; start < len(texts); start += e.batchSize {
		end := min(start+e.batchSize, len(texts))
		// Cada lote grava em uma fatia disjunta de result, sem sobreposição,
		// então as escritas concorrentes são seguras.
		batch := texts[start:end]
		dst := result[start:end]

		g.Go(func() error {
			return e.embedBatch(ctx, batch, dst)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

// EmbedQuery gera o embedding de um único texto de consulta.
func (e *OpenAIEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	out := make([][]float32, 1)
	if err := e.embedBatch(ctx, []string{text}, out); err != nil {
		return nil, err
	}
	return out[0], nil
}

// Gera os embeddings para batch e os grava em dst na ordem de batch.
//
// O resultado da API é reordenado pelo índice de cada embedding, pois a ordem da
// resposta não é garantida. dst deve ter o mesmo comprimento de batch.
func (e *OpenAIEmbedder) embedBatch(ctx context.Context, batch []string, dst [][]float32) error {
	res, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model:      e.model,
		Input:      openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: batch},
		Dimensions: openai.Int(int64(e.dimensions)),
	})
	if err != nil {
		return fmt.Errorf("create embedding: %w", err)
	}

	if len(res.Data) != len(batch) {
		return fmt.Errorf("create embedding: expected %d embeddings, got %d", len(batch), len(res.Data))
	}

	for _, emb := range res.Data {
		if emb.Index < 0 || int(emb.Index) >= len(dst) {
			return fmt.Errorf("create embedding: index %d out of range", emb.Index)
		}
		if len(emb.Embedding) != e.dimensions {
			return &DimensionMismatchError{Expected: e.dimensions, Got: len(emb.Embedding)}
		}
		dst[emb.Index] = toFloat32(emb.Embedding)
	}

	return nil
}

// Converte um slice de float64 para float32.
//
// A API da OpenAI retorna os embeddings como float64, mas o armazenamento via pgvector
// usa float32; a perda de precisão é aceitável para busca por similaridade.
func toFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
