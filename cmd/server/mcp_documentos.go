package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerDocumentosTools adiciona as tools MCP que expõem operações de
// documento do WSSEI. As tools recebem `plataforma` e `plataforma_id` como
// argumentos; o usuário/senha/órgão do SEI são lidos do cadastro persistido.
func registerDocumentosTools(server *mcp.Server, app *application) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "documento_baixar_anexo",
		Description: "Baixa o conteúdo binário de um documento externo (anexo) do SEI e devolve o arquivo codificado em base64.",
	}, app.toolBaixarAnexo)
}

// resolveWSSEIClientByPlataforma devolve um [*wssei.Client] autenticado a partir
// dos identificadores externos do usuário do chatbot, reaproveitando a
// instância via [application.wsseiClients] quando possível.
//
// Retorna [chatbotauth.ErrNotFound] (já tratado pelos toolers) quando o usuário
// não tem cadastro ativo.
func resolveWSSEIClientByPlataforma(
	ctx context.Context,
	app *application,
	plataforma, plataformaID string,
) (*wssei.Client, error) {
	if plataforma == "" || plataformaID == "" {
		return nil, fmt.Errorf("plataforma e plataforma_id são obrigatórios")
	}

	usuario, err := app.chatbotauth.GetUsuario(ctx, plataforma, plataformaID)
	if err != nil {
		return nil, fmt.Errorf("get usuario: %w", err)
	}

	return app.wsseiClients.Get(usuario), nil
}

// toolNotFoundResult devolve um resultado MCP de erro com mensagem amigável
// quando o usuário do chatbot não está cadastrado. Outras falhas continuam
// sendo retornadas como erro Go.
func toolNotFoundResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{
			Text: "Usuário do chatbot não cadastrado. Gere um link de cadastro com chatbot_gerar_link_cadastro.",
		}},
	}
}

// BaixarAnexoInput agrupa as entradas da tool [toolBaixarAnexo].
type BaixarAnexoInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Protocolo    int    `json:"protocolo" jsonschema:"protocolo do documento externo (anexo)"`
}

// BaixarAnexoOutput devolve o conteúdo do anexo codificado em base64,
// acompanhado do content-type e do tamanho original em bytes.
type BaixarAnexoOutput struct {
	ContentType string `json:"content_type" jsonschema:"content-type retornado pelo SEI"`
	Bytes       int    `json:"bytes" jsonschema:"tamanho do anexo em bytes"`
	DataBase64  string `json:"data_base64" jsonschema:"conteúdo do anexo codificado em base64"`
}

func (app *application) toolBaixarAnexo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in BaixarAnexoInput,
) (*mcp.CallToolResult, BaixarAnexoOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), BaixarAnexoOutput{}, nil
		}
		return nil, BaixarAnexoOutput{}, err
	}

	body, contentType, err := client.BaixarAnexo(ctx, in.Protocolo)
	if err != nil {
		return nil, BaixarAnexoOutput{}, fmt.Errorf("baixar anexo: %w", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, BaixarAnexoOutput{}, fmt.Errorf("ler anexo: %w", err)
	}

	return nil, BaixarAnexoOutput{
		ContentType: contentType,
		Bytes:       len(data),
		DataBase64:  base64.StdEncoding.EncodeToString(data),
	}, nil
}
