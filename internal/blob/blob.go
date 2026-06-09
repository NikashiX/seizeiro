// Package blob fornece uma abstração de armazenamento de objetos.
package blob

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNotFound é retornado quando o objeto solicitado não existe no armazenamento.
	ErrNotFound = errors.New("blob: not found")
)

// Storage é uma interface mínima para armazenamento de arquivos.
type Storage interface {
	// Get retorna o conteúdo do objeto identificado por key. Retorna [ErrNotFound]
	// caso o objeto não exista. O chamador é responsável por fechar o
	// [io.ReadCloser] retornado.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Put grava o conteúdo lido de r no objeto identificado por key, sobrescrevendo
	// o objeto caso já exista.
	//
	// Algumas implementações ignoram o contentType.
	Put(ctx context.Context, key, contentType string, r io.Reader) error

	// Delete remove o objeto identificado por key.
	// É idempotente: caso o objeto não exista, retorna nil.
	Delete(ctx context.Context, key string) error
}
