package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var _ Storage = (*FilesystemStorage)(nil)

// FilesystemStorage é uma implementação de [Storage] que armazena os objetos como
// arquivos confinados a um diretório raiz do sistema de arquivos.
//
// O uso de [os.Root] garante que keys não consigam escapar do diretório raiz
// (ex: usando "..").
type FilesystemStorage struct {
	root *os.Root
}

// NewFilesystemStorage cria um [FilesystemStorage] que armazena os objetos sob o
// diretório root, que deve existir.
func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}
	return &FilesystemStorage{root: r}, nil
}

// Get abre o arquivo identificado por key. Retorna [ErrNotFound] caso não exista.
func (s *FilesystemStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	f, err := s.root.Open(key)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	return f, nil
}

// Put grava o conteúdo lido de r no arquivo identificado por key, criando os
// diretórios intermediários quando necessário.
//
// O contentType é ignorado: o sistema de arquivos não armazena metadados de tipo.
func (s *FilesystemStorage) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	if dir := filepath.Dir(key); dir != "." {
		if err := s.root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}

	f, err := s.root.Create(key)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	if _, err := io.Copy(f, r); err != nil {
		// Junta o erro de cópia com o de fechamento para não perder nenhum dos dois.
		return errors.Join(fmt.Errorf("copy: %w", err), f.Close())
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}

// Delete remove o arquivo identificado por key.
// É idempotente: caso o arquivo não exista, retorna nil.
func (s *FilesystemStorage) Delete(ctx context.Context, key string) error {
	err := s.root.Remove(key)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove: %w", err)
	}

	return nil
}
