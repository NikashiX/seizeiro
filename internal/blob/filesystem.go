package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FilesystemStorage armazena blobs no sistema de arquivos local. Cada blob é
// salvo em dois arquivos: `<root>/<key>` (conteúdo) e `<root>/<key>.ct`
// (content-type, opcional).
type FilesystemStorage struct {
	root string
}

// NewFilesystemStorage cria o storage e garante que o diretório raiz existe.
func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	if root == "" {
		return nil, fmt.Errorf("blob filesystem: root vazio")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("blob filesystem mkdir: %w", err)
	}
	return &FilesystemStorage{root: root}, nil
}

// path devolve o caminho absoluto do blob, garantindo que `key` não escapa
// do diretório raiz.
func (s *FilesystemStorage) path(key string) (string, error) {
	if key == "" || strings.ContainsAny(key, `\/`) || strings.Contains(key, "..") {
		// Permitimos apenas chaves planas (ex.: hash hex). Subdiretórios
		// não são necessários no caso de uso atual.
		return "", fmt.Errorf("blob filesystem: chave inválida")
	}
	return filepath.Join(s.root, key), nil
}

func (s *FilesystemStorage) Put(ctx context.Context, key string, r io.Reader, contentType string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("blob filesystem create: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("blob filesystem copy: %w", err)
	}

	if contentType != "" {
		if err := os.WriteFile(p+".ct", []byte(contentType), 0o644); err != nil {
			return fmt.Errorf("blob filesystem write ct: %w", err)
		}
	}

	return nil
}

func (s *FilesystemStorage) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("blob filesystem open: %w", err)
	}

	contentType := "application/octet-stream"
	if ct, err := os.ReadFile(p + ".ct"); err == nil {
		contentType = strings.TrimSpace(string(ct))
	}

	return f, contentType, nil
}

func (s *FilesystemStorage) Exists(ctx context.Context, key string) (bool, error) {
	p, err := s.path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("blob filesystem stat: %w", err)
}

// PresignedURL devolve string vazia: filesystem não tem URL externa; o
// servidor expõe uma rota interna que serve o conteúdo.
func (s *FilesystemStorage) PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, time.Time, error) {
	return "", time.Time{}, nil
}
