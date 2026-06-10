// Package llm fornece abstrações para modelos de linguagem, incluindo a geração
// de embeddings.
package llm

import "context"

// Embedder gera representações vetoriais (embeddings) de textos.
type Embedder interface {
	// EmbedDocuments gera um embedding para cada texto, preservando a ordem de
	// entrada: o embedding de texts[i] está em result[i].
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery gera o embedding de um único texto de consulta.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
