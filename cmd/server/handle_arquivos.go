package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/automatiza-mg/seizeiro/internal/blob"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

// GetArquivoRequest é o request da rota pública de download.
type GetArquivoRequest struct {
	Hash string `path:"hash" doc:"SHA-256 hex do arquivo (64 caracteres hex)"`
}

// registerArquivos registra a rota pública que serve arquivos armazenados. 
//
// Quando o storage tem URL assinada (Azure), idealmente o cliente nem chega
// aqui — recebe a SAS direto. Esta rota cobre o caso filesystem (sem URL
// externa) e funciona como fallback quando o cliente decide passar pelo
// servidor.
//
// A rota é pública: o hash SHA-256 (256 bits) já funciona como token de
// acesso difícil de adivinhar.
func registerArquivos(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-arquivo",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/arquivos/{hash}",
		Tags:        []string{"Arquivos"},
		Summary:     "Baixa um arquivo armazenado pelo SHA-256",
		Description: "Devolve o conteúdo binário do arquivo associado ao hash informado. " +
			"Quando o storage suporta URLs assinadas (Azure), pode responder com 302 redirect. " +
			"Sem autenticação: o hash SHA-256 atua como token de acesso.",
	}, func(ctx context.Context, in *GetArquivoRequest) (*huma.StreamResponse, error) {
		if len(in.Hash) != 64 {
			return nil, huma.Error400BadRequest("hash inválido")
		}

		body, contentType, err := app.arquivos.Get(ctx, in.Hash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, blob.ErrNotFound) {
				return nil, huma.Error404NotFound("arquivo não encontrado")
			}
			return nil, fmt.Errorf("get arquivo: %w", err)
		}

		return &huma.StreamResponse{
			Body: func(hctx huma.Context) {
				if contentType != "" {
					hctx.SetHeader("Content-Type", contentType)
				}
				defer body.Close()
				_, _ = io.Copy(hctx.BodyWriter(), body)
			},
		}, nil
	})
}
