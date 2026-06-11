package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewOpenAIEmbedder_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params OpenAIParams
	}{
		{
			name:   "missing api key",
			params: OpenAIParams{Model: "text-embedding-3-small", Dimensions: 1536},
		},
		{
			name:   "missing model",
			params: OpenAIParams{APIKey: "secret", Dimensions: 1536},
		},
		{
			name:   "non-positive dimensions",
			params: OpenAIParams{APIKey: "secret", Model: "text-embedding-3-small"},
		},
		{
			name:   "negative batch size",
			params: OpenAIParams{APIKey: "secret", Model: "text-embedding-3-small", Dimensions: 1536, BatchSize: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewOpenAIEmbedder(tt.params)
			if !errors.Is(err, ErrInvalidEmbedderConfig) {
				t.Fatalf("error = %v, want ErrInvalidEmbedderConfig", err)
			}
		})
	}
}

// fakeEmbeddingsServer responde a /embeddings gerando, para cada input, um vetor de
// dimensão dims cujo primeiro elemento codifica a posição global do input, permitindo
// verificar a ordem dos resultados. As respostas são embaralhadas para garantir que o
// embedder reordene pelo índice.
func fakeEmbeddingsServer(tb testing.TB, dims int, requests *[][]string) *httptest.Server {
	tb.Helper()

	// Os lotes são processados em paralelo, então o registro das requisições
	// precisa ser sincronizado.
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			tb.Errorf("decode request: %v", err)
		}
		if requests != nil {
			mu.Lock()
			*requests = append(*requests, body.Input)
			mu.Unlock()
		}

		type embedding struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
			Object    string    `json:"object"`
		}
		data := make([]embedding, len(body.Input))
		for i, in := range body.Input {
			vec := make([]float64, dims)
			// Codifica o conteúdo do input no primeiro elemento para validar a ordem.
			var val float64
			fmt.Sscanf(in, "%g", &val)
			vec[0] = val
			data[i] = embedding{Embedding: vec, Index: i, Object: "embedding"}
		}
		// Embaralha invertendo os índices para garantir que a resposta não esteja ordenada.
		for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
			data[i], data[j] = data[j], data[i]
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"model":  "text-embedding-3-small",
			"data":   data,
			"usage":  map[string]int{"prompt_tokens": 0, "total_tokens": 0},
		})
	}))
	tb.Cleanup(srv.Close)
	return srv
}

func TestOpenAIEmbedder_EmbedDocuments_Batching(t *testing.T) {
	t.Parallel()

	const dims = 3
	var requests [][]string
	srv := fakeEmbeddingsServer(t, dims, &requests)

	emb, err := NewOpenAIEmbedder(OpenAIParams{
		APIKey:     "secret",
		Model:      "text-embedding-3-small",
		Dimensions: dims,
		BatchSize:  2,
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	texts := []string{"1", "2", "3", "4", "5"}
	got, err := emb.EmbedDocuments(t.Context(), texts)
	if err != nil {
		t.Fatalf("embed documents: %v", err)
	}

	// Com BatchSize 2 e 5 textos, espera-se 3 requisições (2+2+1).
	if len(requests) != 3 {
		t.Fatalf("requests = %d, want 3", len(requests))
	}

	// O primeiro elemento de cada vetor deve corresponder ao texto de entrada,
	// confirmando a ordem mesmo com a resposta embaralhada.
	want := [][]float32{
		{1, 0, 0},
		{2, 0, 0},
		{3, 0, 0},
		{4, 0, 0},
		{5, 0, 0},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("embeddings mismatch (-want +got):\n%s", diff)
	}
}

func TestOpenAIEmbedder_EmbedDocuments_ConcurrentBatches(t *testing.T) {
	t.Parallel()

	const dims = 3
	var requests [][]string
	srv := fakeEmbeddingsServer(t, dims, &requests)

	emb, err := NewOpenAIEmbedder(OpenAIParams{
		APIKey:     "secret",
		Model:      "text-embedding-3-small",
		Dimensions: dims,
		BatchSize:  1,
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	// BatchSize 1 com 10 textos gera 10 lotes, mais que o limite de
	// concorrência, exercitando o processamento paralelo e a ordenação.
	texts := make([]string, 10)
	want := make([][]float32, 10)
	for i := range texts {
		texts[i] = fmt.Sprintf("%d", i+1)
		want[i] = []float32{float32(i + 1), 0, 0}
	}

	got, err := emb.EmbedDocuments(t.Context(), texts)
	if err != nil {
		t.Fatalf("embed documents: %v", err)
	}

	if len(requests) != len(texts) {
		t.Fatalf("requests = %d, want %d", len(requests), len(texts))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("embeddings mismatch (-want +got):\n%s", diff)
	}
}

func TestOpenAIEmbedder_EmbedDocuments_Empty(t *testing.T) {
	t.Parallel()

	emb, err := NewOpenAIEmbedder(OpenAIParams{
		APIKey:     "secret",
		Model:      "text-embedding-3-small",
		Dimensions: 3,
		BaseURL:    "http://invalid.invalid",
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	got, err := emb.EmbedDocuments(t.Context(), nil)
	if err != nil {
		t.Fatalf("embed documents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0", len(got))
	}
}

func TestOpenAIEmbedder_EmbedQuery(t *testing.T) {
	t.Parallel()

	const dims = 3
	srv := fakeEmbeddingsServer(t, dims, nil)

	emb, err := NewOpenAIEmbedder(OpenAIParams{
		APIKey:     "secret",
		Model:      "text-embedding-3-small",
		Dimensions: dims,
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	got, err := emb.EmbedQuery(t.Context(), "42")
	if err != nil {
		t.Fatalf("embed query: %v", err)
	}

	want := []float32{42, 0, 0}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("embedding mismatch (-want +got):\n%s", diff)
	}
}

func TestOpenAIEmbedder_EmbedDocuments_DimensionMismatch(t *testing.T) {
	t.Parallel()

	// O servidor gera vetores de dimensão 2, mas o embedder espera 3.
	srv := fakeEmbeddingsServer(t, 2, nil)

	emb, err := NewOpenAIEmbedder(OpenAIParams{
		APIKey:     "secret",
		Model:      "text-embedding-3-small",
		Dimensions: 3,
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	_, err = emb.EmbedDocuments(t.Context(), []string{"1"})
	mismatch, ok := errors.AsType[*DimensionMismatchError](err)
	if !ok {
		t.Fatalf("error = %v, want *DimensionMismatchError", err)
	}
	if mismatch.Expected != 3 || mismatch.Got != 2 {
		t.Fatalf("mismatch = %+v, want {Expected:3 Got:2}", mismatch)
	}
}

func TestToFloat32(t *testing.T) {
	t.Parallel()

	got := toFloat32([]float64{0, 1.5, -2.25})
	want := []float32{0, 1.5, -2.25}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("toFloat32 mismatch (-want +got):\n%s", diff)
	}
}
