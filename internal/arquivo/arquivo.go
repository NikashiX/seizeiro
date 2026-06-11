package arquivo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type Service struct {
	pool    *pgxpool.Pool
	q       *postgres.Queries
	storage blob.Storage
	river   *river.Client[pgx.Tx]
}

func NewService(pool *pgxpool.Pool, storage blob.Storage, river *river.Client[pgx.Tx]) *Service {
	return &Service{
		pool:    pool,
		q:       postgres.New(pool),
		storage: storage,
		river:   river,
	}
}

type Arquivo struct {
	ID           int64     `json:"id"`
	HashSHA256   []byte    `json:"hash_sha256"`
	ChaveStorage string    `json:"chave_storage"`
	TamanhoBytes int64     `json:"tamanho_bytes"`
	CriadoEm     time.Time `json:"criado_em"`
}

// CreateArquivo processa e adiciona um novo arquivo ao banco de dados e storage.
// O parâmetro contentType é opcional e, se não for informado, a função
// [http.DetectContentType] será usada como fallback.
//
// Se o contentType informado não for suportado, retorna [*UnsupportedError].
func (s *Service) CreateArquivo(ctx context.Context, r io.ReadSeeker, contentType string) (*Arquivo, error) {
	if contentType == "" {
		var err error
		contentType, err = detectContentType(r)
		if err != nil {
			return nil, fmt.Errorf("detect content type: %w", err)
		}
	}

	h := sha256.New()
	size, err := io.Copy(h, r)
	if err != nil {
		return nil, fmt.Errorf("copy to hash: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek from hash: %w", err)
	}

	checksum := h.Sum(nil)

	// Verifica se o arquivo já existe no nosso banco de dados.
	_, err = s.q.GetArquivoByHashSHA256(ctx, checksum)
	if err == nil {
		return nil, ErrAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get arquivo by hash: %w", err)
	}

	storageKey := fmt.Sprintf("arquivos/%x", checksum)

	err = s.storage.Put(ctx, storageKey, contentType, r)
	if err != nil {
		return nil, fmt.Errorf("storage put: %w", err)
	}

	// Salva o arquivo e enfileira a extração de conteúdo na mesma transação:
	// a task de extração existe se, e somente se, o arquivo foi criado.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row, err := s.q.WithTx(tx).SaveArquivo(ctx, postgres.SaveArquivoParams{
		HashSHA256:   checksum,
		ChaveStorage: storageKey,
		MimeType:     contentType,
		TamanhoBytes: size,
	})
	if err != nil {
		return nil, fmt.Errorf("save arquivo: %w", err)
	}

	_, err = s.river.InsertTx(ctx, tx, ExtractArgs{ArquivoID: row.ID}, nil)
	if err != nil {
		return nil, fmt.Errorf("insert extract task: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &Arquivo{
		ID:           row.ID,
		HashSHA256:   row.HashSHA256,
		ChaveStorage: row.ChaveStorage,
		TamanhoBytes: row.TamanhoBytes,
		CriadoEm:     row.CriadoEm.Time,
	}, nil
}

func detectContentType(r io.ReadSeeker) (string, error) {
	data := make([]byte, 512)
	if _, err := r.Read(data); err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek from data: %w", err)
	}
	return http.DetectContentType(data), nil
}
