// Package blob abstrai o armazenamento de objetos binários (arquivos
// baixados do SEI) sob a forma de uma interface única [Storage] com duas
// implementações: [FilesystemStorage] (uso em desenvolvimento) e
// [AzureStorage] (uso em produção).
//
// Cada objeto é identificado por uma chave (`key`) opaca — tipicamente o
// hash SHA-256 hex do conteúdo, garantindo deduplicação no nível de bytes.
package blob

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotFound é o erro retornado por [Storage.Get] quando a chave não existe.
var ErrNotFound = errors.New("blob: not found")

// Storage representa um backend de armazenamento de blobs.
type Storage interface {
	// Put grava o conteúdo de r sob a chave informada. O contentType é
	// persistido como metadado quando o backend suportar.
	Put(ctx context.Context, key string, r io.Reader, contentType string) error
	// Get devolve o conteúdo do blob e o content-type associado. Devolve
	// [ErrNotFound] quando a chave não existe.
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	// Exists verifica se a chave existe no backend sem baixar o conteúdo.
	Exists(ctx context.Context, key string) (bool, error)
	// PresignedURL devolve uma URL temporária para download direto do blob.
	// Retorna a URL e o instante de expiração. Backends que não suportam
	// URLs assinadas (ex.: filesystem) devolvem a string vazia, permitindo
	// que o chamador caia em uma rota interna que sirva os bytes.
	PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, time.Time, error)
}
