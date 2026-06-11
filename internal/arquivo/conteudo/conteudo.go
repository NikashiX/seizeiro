// Package conteudo extrai o conteúdo textual de arquivos e o persiste no
// banco de dados.
//
// O método de extração é resolvido a partir do MIME type do arquivo:
//
//   - text/plain: o conteúdo é lido diretamente do storage ([MetodoPlain]).
//   - text/html: o conteúdo é convertido para Markdown ([MetodoHTMLMarkdown]).
//   - demais tipos: o texto é extraído via OCR usando a Azure Document
//     Intelligence ([MetodoOCR]).
package conteudo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"

	"github.com/automatiza-mg/seizeiro/internal/arquivo"
	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/automatiza-mg/seizeiro/internal/docintel"
	"github.com/automatiza-mg/seizeiro/internal/markdown"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MetodoOCR          = "ocr"
	MetodoPlain        = "plain"
	MetodoHTMLMarkdown = "html_markdown"

	FormatoPlain    = "plain"
	FormatoMarkdown = "markdown"
)

// OCR extrai texto de documentos via análise assíncrona.
type OCR interface {
	// AnalyzeDocument inicia a análise e retorna a location da operação.
	AnalyzeDocument(ctx context.Context, r io.Reader, contentType string) (string, error)
	// PollResult aguarda a conclusão da operação e retorna o texto extraído.
	PollResult(ctx context.Context, location string) (string, error)
}

// Garante que o cliente da Azure Document Intelligence implementa OCR.
var _ OCR = (*docintel.Client)(nil)

type Service struct {
	pool    *pgxpool.Pool
	q       *postgres.Queries
	ocr     OCR
	storage blob.Storage
}

func NewService(pool *pgxpool.Pool, ocr OCR, storage blob.Storage) *Service {
	return &Service{
		pool:    pool,
		q:       postgres.New(pool),
		ocr:     ocr,
		storage: storage,
	}
}

// ExtractConteudo extrai o conteúdo textual de um arquivo e o persiste em
// arquivos_conteudo. O método de extração é resolvido a partir do MIME type
// do arquivo. É idempotente: caso o conteúdo já tenha sido extraído com o
// mesmo método, retorna nil sem reprocessar.
//
// Retorna [arquivo.ErrNotFound] caso o arquivo não exista.
func (s *Service) ExtractConteudo(ctx context.Context, arquivoID int64) error {
	row, err := s.q.GetArquivo(ctx, arquivoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return arquivo.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get arquivo: %w", err)
	}

	mediaType, _, err := mime.ParseMediaType(row.MimeType)
	if err != nil {
		return &arquivo.UnsupportedError{ContentType: row.MimeType}
	}

	var method, format string
	switch mediaType {
	case "text/plain":
		method, format = MetodoPlain, FormatoPlain
	case "text/html":
		method, format = MetodoHTMLMarkdown, FormatoMarkdown
	default:
		method, format = MetodoOCR, FormatoMarkdown
	}

	// Evita reprocessar (e pagar por) uma extração já concluída.
	_, err = s.q.GetArquivoConteudo(ctx, postgres.GetArquivoConteudoParams{
		ArquivoID: arquivoID,
		Metodo:    method,
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("get arquivo conteudo: %w", err)
	}

	rc, err := s.storage.Get(ctx, row.ChaveStorage)
	if err != nil {
		return fmt.Errorf("storage get: %w", err)
	}
	defer rc.Close()

	var text string
	switch method {
	case MetodoPlain:
		b, err := io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("read all: %w", err)
		}
		text = string(b)
	case MetodoHTMLMarkdown:
		text, err = markdown.ConvertHTML(rc, row.MimeType, markdown.WithoutImg())
		if err != nil {
			return fmt.Errorf("convert html: %w", err)
		}
	case MetodoOCR:
		location, err := s.ocr.AnalyzeDocument(ctx, rc, row.MimeType)
		if err != nil {
			return fmt.Errorf("analyze document: %w", err)
		}

		text, err = s.ocr.PollResult(ctx, location)
		if err != nil {
			return fmt.Errorf("poll result: %w", err)
		}
	}

	_, err = s.q.SaveArquivoConteudo(ctx, postgres.SaveArquivoConteudoParams{
		ArquivoID: arquivoID,
		Metodo:    method,
		Formato:   format,
		Conteudo:  text,
	})
	if err != nil {
		if database.IsUniqueError(err, "arquivos_conteudo_arquivo_id_metodo_key") {
			return nil
		}
		return fmt.Errorf("save arquivo conteudo: %w", err)
	}

	return nil
}
