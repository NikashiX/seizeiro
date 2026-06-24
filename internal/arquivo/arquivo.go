// Package arquivo orquestra o download, hashing e armazenamento deduplicado
// de arquivos baixados do SEI.
//
// O fluxo síncrono — [Service.Baixar] — sempre baixa o anexo via WSSEI,
// calcula o SHA-256 em streaming e:
//
//   - reusa o blob existente quando o hash já está em `arquivos`;
//   - grava no storage e em `arquivos` quando o hash é novo;
//   - registra (UPSERT) o último hash conhecido em `documentos_anexo`.
//
// O enfileiramento assíncrono é responsabilidade dos workers do pacote
// [internal/tasks].
package arquivo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Resultado descreve o arquivo disponibilizado pelo download.
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

// WSSEIResolver obtém o cliente WSSEI autenticado para o usuário do chatbot
// identificado por (plataforma, plataformaID). Retorna [ErrUsuarioNotFound]
// quando o usuário não está cadastrado.
type WSSEIResolver interface {
	ResolveByPlataforma(ctx context.Context, plataforma, plataformaID string) (*wssei.Client, error)
}

// ErrUsuarioNotFound indica que o usuário do chatbot não está cadastrado.
// Implementações de [WSSEIResolver] devem retornar este erro para que o
// worker decida pular o webhook (sem destinatário válido para notificar).
var ErrUsuarioNotFound = errors.New("arquivo: usuario do chatbot nao cadastrado")

// Service encapsula o pipeline de download + hash + dedup + persistência.
type Service struct {
	pool          *pgxpool.Pool
	q             *postgres.Queries
	storage       blob.Storage
	links         LinkBuilder
	wsseiResolver WSSEIResolver
	urlTTL        time.Duration
}

// Config agrupa as dependências do [Service].
type Config struct {
	Pool          *pgxpool.Pool
	Storage       blob.Storage
	Links         LinkBuilder
	WSSEIResolver WSSEIResolver
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
		pool:          cfg.Pool,
		q:             postgres.New(cfg.Pool),
		storage:       cfg.Storage,
		links:         cfg.Links,
		wsseiResolver: cfg.WSSEIResolver,
		urlTTL:        ttl,
	}
}

// WSSEIResolver expõe o resolver injetado para que workers possam resolver
// o cliente WSSEI a partir de (plataforma, plataforma_id).
func (s *Service) WSSEIResolver() WSSEIResolver {
	return s.wsseiResolver
}

// Baixar executa o fluxo síncrono de download + hash + persistência + URL,
// usando o cliente WSSEI já autenticado.
func (s *Service) Baixar(
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

	// Quando o SEI devolve application/json, o conteúdo útil é o campo
	// `data` do envelope WSSEI (uma string HTML). Extraímos o HTML e
	// passamos a tratar o anexo como text/html — o hash é calculado sobre
	// o HTML final, não sobre o JSON original.
	buf, contentType, err := readBody(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("arquivo: ler conteúdo: %w", err)
	}

	hashHex := sha256Hex(buf.Bytes())
	tamanho := int64(buf.Len())

	existing, err := s.q.GetArquivo(ctx, hashHex)
	switch {
	case err == nil:
		// Arquivo já existe no banco: reusa storage_key e atualiza o
		// vínculo id_protocolo->hash.
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

	storageKey := hashHex
	if err := s.storage.Put(ctx, storageKey, buf, contentType); err != nil {
		return nil, fmt.Errorf("arquivo: put no storage: %w", err)
	}

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

// readBody consome `r` por completo. Quando `contentType` é
// `application/json`, interpreta os bytes como um envelope WSSEI e
// retorna o HTML extraído com `text/html; charset=utf-8`. Caso contrário,
// devolve o conteúdo bruto sem modificação.
//
// Se o JSON não for um envelope reconhecível ou o campo `data` não puder
// ser interpretado como string, o conteúdo original é mantido como
// `application/json` para que o usuário consiga inspecionar o payload bruto.
func readBody(r io.Reader, contentType string) (*bytes.Buffer, string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}

	if isJSONContentType(contentType) {
		if html, ok := extractHTMLFromEnvelope(raw); ok {
			return bytes.NewBuffer(html), "text/html; charset=utf-8", nil
		}
	}

	return bytes.NewBuffer(raw), contentType, nil
}

// isJSONContentType retorna true quando `contentType` indica JSON,
// tolerando parâmetros (charset) e o tipo `application/problem+json`.
func isJSONContentType(contentType string) bool {
	media := strings.ToLower(contentType)
	if i := strings.IndexByte(media, ';'); i >= 0 {
		media = media[:i]
	}
	media = strings.TrimSpace(media)
	return media == "application/json" || strings.HasSuffix(media, "+json")
}

// envelopeWSSEI representa o envelope JSON padrão do WSSEI usado em
// respostas como `/documento/baixar/anexo/{id}` quando o anexo é HTML
// (gerado pelo editor SEI). O campo `data` é parseado como bruto e depois
// tentamos extrair uma string.
type envelopeWSSEI struct {
	Sucesso bool            `json:"sucesso"`
	Data    json.RawMessage `json:"data"`
}

// extractHTMLFromEnvelope tenta extrair o HTML embutido no campo `data` do
// envelope WSSEI. Retorna (html, true) em caso de sucesso ou (nil, false)
// quando o envelope não pôde ser interpretado.
func extractHTMLFromEnvelope(raw []byte) ([]byte, bool) {
	var env envelopeWSSEI
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, false
	}
	if !env.Sucesso || len(env.Data) == 0 {
		return nil, false
	}
	var s string
	if err := json.Unmarshal(env.Data, &s); err != nil {
		// Quando `data` não é string (ex.: objeto/array), o conteúdo
		// não casa com o cenário "anexo HTML"; devolvemos o JSON cru.
		return nil, false
	}
	if s == "" {
		return nil, false
	}
	return []byte(s), true
}

// sha256Hex devolve o SHA-256 hex de `data`.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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
