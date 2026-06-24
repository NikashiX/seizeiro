// Package arquivo orquestra o download, hashing e armazenamento deduplicado
// de arquivos baixados do SEI.
//
// O fluxo central — [Service.BaixarAnexo] — recebe o id_protocolo do
// documento, sempre baixa o conteúdo atual via WSSEI, calcula o SHA-256 em
// streaming e:
//
//   - reusa o blob existente quando o hash já está em `arquivos`;
//   - grava no storage e em `arquivos` quando o hash é novo;
//   - registra (UPSERT) o último hash conhecido em `documentos_anexo`.
//
// Sempre devolve [Resultado] com o hash, content-type, tamanho e a URL
// pública para o cliente baixar o blob.
package arquivo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Resultado descreve o arquivo disponibilizado ao cliente.
type Resultado struct {
	Hash        string    // SHA-256 hex
	ContentType string    // MIME devolvido pelo SEI
	Bytes       int64     // tamanho em bytes
	URL         string    // link público para download
	ExpiraEm    time.Time // expiração da URL (zero quando não expira)
	JaExistia   bool      // true quando o hash já estava no banco antes desta chamada
}

// LinkBuilder produz a URL pública para um blob. Quando o backend de storage
// devolve uma URL assinada (Azure SAS), o LinkBuilder não é consultado.
// Caso contrário (filesystem), o LinkBuilder gera a URL da rota interna que
// serve o arquivo.
type LinkBuilder interface {
	ArquivoURL(hash string) string
}

// Service encapsula o pipeline de download + hash + dedup + persistência.
type Service struct {
	pool     *pgxpool.Pool
	q        *postgres.Queries
	storage  blob.Storage
	links    LinkBuilder
	urlTTL   time.Duration
}

// Config agrupa as dependências do [Service].
type Config struct {
	Pool    *pgxpool.Pool
	Storage blob.Storage
	Links   LinkBuilder
	// URLTTL é o tempo de validade da URL assinada quando o backend
	// suportar (Azure). Defaults para 1 hora.
	URLTTL time.Duration
}

// NewService cria o Service.
func NewService(cfg Config) *Service {
	ttl := cfg.URLTTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &Service{
		pool:    cfg.Pool,
		q:       postgres.New(cfg.Pool),
		storage: cfg.Storage,
		links:   cfg.Links,
		urlTTL:  ttl,
	}
}

// BaixarAnexo executa o fluxo completo descrito no doc do pacote.
//
// Sempre baixa o anexo via `client.BaixarAnexo` (não há cache pré-download).
// A deduplicação acontece após o hash ser conhecido: se o blob já existe
// no banco, o body baixado é descartado e o storage não é regravado.
func (s *Service) BaixarAnexo(
	ctx context.Context,
	client *wssei.Client,
	idProtocolo int,
) (*Resultado, error) {
	if idProtocolo <= 0 {
		return nil, fmt.Errorf("arquivo: id_protocolo inválido: %d", idProtocolo)
	}

	body, contentType, err := client.BaixarAnexo(ctx, idProtocolo)
	if err != nil {
		return nil, fmt.Errorf("arquivo: baixar anexo do sei: %w", err)
	}
	defer body.Close()

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Calcula o hash enquanto bufferiza o conteúdo em memória. Para anexos
	// do SEI o tamanho típico é compatível com isso; se virar gargalo, dá
	// para trocar por um arquivo temporário + segunda leitura.
	hashHex, buf, err := readAndHash(body)
	if err != nil {
		return nil, fmt.Errorf("arquivo: ler conteúdo: %w", err)
	}

	tamanho := int64(buf.Len())

	// Verifica se o hash já existe no banco; se sim, evita regravar o blob.
	existing, err := s.q.GetArquivo(ctx, hashHex)
	switch {
	case err == nil:
		// Já existe: apenas atualiza o vínculo id_protocolo -> hash.
		if err := s.saveAnexo(ctx, idProtocolo, hashHex); err != nil {
			return nil, err
		}
		url, expira, err := s.buildURL(ctx, existing.StorageKey, existing.Hash)
		if err != nil {
			return nil, err
		}
		return &Resultado{
			Hash:        existing.Hash,
			ContentType: existing.ContentType,
			Bytes:       existing.TamanhoBytes,
			URL:         url,
			ExpiraEm:    expira,
			JaExistia:   true,
		}, nil
	case errors.Is(err, pgx.ErrNoRows):
		// segue fluxo de gravação
	default:
		return nil, fmt.Errorf("arquivo: get arquivo: %w", err)
	}

	// Grava o blob no storage. A chave é o próprio hash para alinhar com a
	// deduplicação.
	storageKey := hashHex
	if err := s.storage.Put(ctx, storageKey, buf, contentType); err != nil {
		return nil, fmt.Errorf("arquivo: put no storage: %w", err)
	}

	// Persiste em arquivos + documentos_anexo numa transação. ON CONFLICT
	// torna a operação idempotente em caso de corrida com outra chamada.
	if err := s.persist(ctx, hashHex, storageKey, contentType, tamanho, idProtocolo); err != nil {
		return nil, err
	}

	url, expira, err := s.buildURL(ctx, storageKey, hashHex)
	if err != nil {
		return nil, err
	}
	return &Resultado{
		Hash:        hashHex,
		ContentType: contentType,
		Bytes:       tamanho,
		URL:         url,
		ExpiraEm:    expira,
		JaExistia:   false,
	}, nil
}

// readAndHash consome `r` por completo, devolvendo o SHA-256 hex e um buffer
// reaproveitável para upload. Usa MultiWriter para evitar duas passadas.
func readAndHash(r io.Reader) (string, *bytes.Buffer, error) {
	h := sha256.New()
	buf := new(bytes.Buffer)
	mw := io.MultiWriter(h, buf)

	if _, err := io.Copy(mw, r); err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(h.Sum(nil)), buf, nil
}

// persist insere o arquivo e atualiza o vínculo id_protocolo->hash dentro de
// uma única transação.
func (s *Service) persist(
	ctx context.Context,
	hashHex, storageKey, contentType string,
	tamanho int64,
	idProtocolo int,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("arquivo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)

	err = q.SaveArquivo(ctx, postgres.SaveArquivoParams{
		Hash:         hashHex,
		StorageKey:   storageKey,
		ContentType:  contentType,
		TamanhoBytes: tamanho,
	})
	if err != nil {
		return fmt.Errorf("arquivo: save arquivo: %w", err)
	}

	err = q.SaveDocumentoAnexo(ctx, postgres.SaveDocumentoAnexoParams{
		IDProtocolo: int64(idProtocolo),
		Hash:        hashHex,
	})
	if err != nil {
		return fmt.Errorf("arquivo: save documento_anexo: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("arquivo: commit tx: %w", err)
	}
	return nil
}

// saveAnexo atualiza apenas o vínculo id_protocolo->hash quando o arquivo
// já existia no banco.
func (s *Service) saveAnexo(ctx context.Context, idProtocolo int, hashHex string) error {
	err := s.q.SaveDocumentoAnexo(ctx, postgres.SaveDocumentoAnexoParams{
		IDProtocolo: int64(idProtocolo),
		Hash:        hashHex,
	})
	if err != nil {
		return fmt.Errorf("arquivo: save documento_anexo: %w", err)
	}
	return nil
}

// buildURL prefere a URL assinada do storage (Azure SAS) e cai na rota
// interna do servidor quando o backend não suporta URL externa
// (filesystem).
func (s *Service) buildURL(ctx context.Context, storageKey, hashHex string) (string, time.Time, error) {
	url, expira, err := s.storage.PresignedURL(ctx, storageKey, s.urlTTL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("arquivo: presigned url: %w", err)
	}
	if url != "" {
		return url, expira, nil
	}
	return s.links.ArquivoURL(hashHex), time.Time{}, nil
}

// Get devolve o blob bruto do storage a partir do hash. Usado pela rota
// interna que serve arquivos quando o backend é filesystem.
func (s *Service) Get(ctx context.Context, hashHex string) (io.ReadCloser, string, error) {
	arq, err := s.q.GetArquivo(ctx, hashHex)
	if err != nil {
		return nil, "", err
	}
	body, _, err := s.storage.Get(ctx, arq.StorageKey)
	if err != nil {
		return nil, "", err
	}
	return body, arq.ContentType, nil
}
