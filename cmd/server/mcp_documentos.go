package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/seiws"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/automatiza-mg/seizeiro/internal/soap"
	"github.com/automatiza-mg/seizeiro/internal/tasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerDocumentosTools adiciona as tools MCP que expõem operações de
// documento do WSSEI. As tools recebem `plataforma` e `plataforma_id` como
// argumentos; o usuário/senha/órgão do SEI são lidos do cadastro persistido.
func registerDocumentosTools(server *mcp.Server, app *application) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "documento_baixar_anexo",
		Description: "Enfileira o download de um documento externo (anexo) do SEI. " +
			"A tool retorna IMEDIATAMENTE com {status:'enfileirado'} (não devolve a URL na mesma resposta). " +
			"Quando o download terminar, o usuário receberá uma notificação assíncrona via webhook (no WhatsApp/Telegram) com a URL para baixar o arquivo. " +
			"REQUER o id interno do documento (campo `id_protocolo`). " +
			"NÃO use o protocolo formatado exibido ao usuário (ex.: 0107523). " +
			"Para obter o id interno a partir do protocolo formatado, chame antes `documento_consultar_soap` e use o campo `id_documento` da resposta.",
	}, app.toolBaixarAnexo)

	mcp.AddTool(server, &mcp.Tool{
		Name: "documento_listar_processo",
		Description: "Lista paginada de documentos de um processo do SEI a partir do id interno do procedimento. " +
			"Para cada documento, devolve `atributos.protocoloFormatado` (string exibida ao usuário, ex.: 0107523). " +
			"Para baixar o anexo de um item da lista, use o `protocoloFormatado` em `documento_consultar_soap` para obter o `id_documento` e então passe esse id em `documento_baixar_anexo`.",
	}, app.toolListarDocumentosProcesso)

	mcp.AddTool(server, &mcp.Tool{
		Name: "documento_consultar_soap",
		Description: "Consulta os metadados de um documento do SEI pela API SOAP legada (SeiWS.php), a partir do protocolo formatado (ex.: 0107523). " +
			"Use esta tool para converter o protocolo formatado em `id_documento` (id interno), que é o valor exigido por `documento_baixar_anexo` (campo `id_protocolo`). " +
			"Usa credenciais globais da aplicação (SiglaSistema/IdentificacaoServico) — não requer plataforma/plataforma_id.",
	}, app.toolConsultarDocumentoSOAP)
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
	// IDProtocolo é o id interno do documento. Obtido a partir do campo
	// `id_documento` retornado por `documento_consultar_soap`. NÃO confundir
	// com o protocoloFormatado exibido ao usuário (ex.: 0107523).
	IDProtocolo int `json:"id_protocolo" jsonschema:"id interno do documento externo (anexo). Equivalente ao campo id_documento devolvido por documento_consultar_soap. NÃO usar o protocolo formatado exibido ao usuário (ex.: 0107523)."`
}

// BaixarAnexoOutput é a resposta da tool — apenas indica que o job foi
// enfileirado. A URL real chega via webhook quando o download termina.
type BaixarAnexoOutput struct {
	Status   string `json:"status" jsonschema:"sempre 'enfileirado' quando o download foi aceito"`
	Mensagem string `json:"mensagem" jsonschema:"mensagem amigável para o usuário"`
}

func (app *application) toolBaixarAnexo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in BaixarAnexoInput,
) (*mcp.CallToolResult, BaixarAnexoOutput, error) {
	// Garante usuário cadastrado antes de enfileirar (evita job descartado).
	if _, err := app.chatbotauth.GetUsuario(ctx, in.Plataforma, in.PlataformaID); err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), BaixarAnexoOutput{}, nil
		}
		return nil, BaixarAnexoOutput{}, fmt.Errorf("get usuario: %w", err)
	}

	if in.IDProtocolo <= 0 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "id_protocolo inválido"}},
		}, BaixarAnexoOutput{}, nil
	}

	_, err := app.river.Insert(ctx, tasks.BaixarAnexoArgs{
		Plataforma:   in.Plataforma,
		PlataformaID: in.PlataformaID,
		IDProtocolo:  in.IDProtocolo,
	}, nil)
	if err != nil {
		return nil, BaixarAnexoOutput{}, fmt.Errorf("enfileirar download: %w", err)
	}

	return nil, BaixarAnexoOutput{
		Status:   "enfileirado",
		Mensagem: "O download foi enfileirado. Você receberá uma mensagem quando o arquivo estiver pronto.",
	}, nil
}

// ListarDocumentosProcessoInput agrupa as entradas da tool
// [toolListarDocumentosProcesso].
//
// Os campos opcionais com valor zero são omitidos da requisição ao WSSEI.
type ListarDocumentosProcessoInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Procedimento int    `json:"procedimento" jsonschema:"id interno do processo (procedimento) no SEI"`
	Limit        int    `json:"limit,omitempty" jsonschema:"limite de registros da paginação"`
	Start        int    `json:"start,omitempty" jsonschema:"página de início da paginação"`
}

// ListarDocumentosProcessoOutput devolve a lista de documentos e o total de
// registros retornados pelo WSSEI.
type ListarDocumentosProcessoOutput struct {
	Total      int               `json:"total" jsonschema:"total de registros disponíveis no WSSEI"`
	Documentos []wssei.Documento `json:"documentos" jsonschema:"documentos retornados pela página atual"`
}

func (app *application) toolListarDocumentosProcesso(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ListarDocumentosProcessoInput,
) (*mcp.CallToolResult, ListarDocumentosProcessoOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), ListarDocumentosProcessoOutput{}, nil
		}
		return nil, ListarDocumentosProcessoOutput{}, err
	}

	docs, total, err := client.ListarDocumentosProcessos(ctx, wssei.ListarDocumentosParams{
		Limit:        in.Limit,
		Start:        in.Start,
		Procedimento: in.Procedimento,
	})
	if err != nil {
		return nil, ListarDocumentosProcessoOutput{}, fmt.Errorf("listar documentos processo: %w", err)
	}

	return nil, ListarDocumentosProcessoOutput{
		Total:      total,
		Documentos: docs,
	}, nil
}

// ConsultarDocumentoSOAPInput agrupa as entradas da tool
// [toolConsultarDocumentoSOAP].
type ConsultarDocumentoSOAPInput struct {
	Protocolo string `json:"protocolo" jsonschema:"protocolo formatado do documento (ex.: 0000000.00000.0000000/0000-00)"`
}

// ConsultarDocumentoSOAPOutput devolve os metadados do documento retornados
// pela API SOAP legada do SEI.
type ConsultarDocumentoSOAPOutput struct {
	Documento seiws.RetornoConsultaDocumento `json:"documento" jsonschema:"metadados do documento retornados pela API SOAP do SEI"`
}

// toolSEIWSNotConfiguredResult devolve um resultado MCP de erro quando a
// integração com a API SOAP legada do SEI não está configurada
// (SEI_WS_URL / SEI_SIGLA_SISTEMA / SEI_IDENTIFICACAO_SERVICO).
func toolSEIWSNotConfiguredResult() *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{
			Text: "Integração com a API SOAP do SEI não configurada (SEI_WS_URL/SEI_SIGLA_SISTEMA/SEI_IDENTIFICACAO_SERVICO).",
		}},
	}
}

func (app *application) toolConsultarDocumentoSOAP(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ConsultarDocumentoSOAPInput,
) (*mcp.CallToolResult, ConsultarDocumentoSOAPOutput, error) {
	if app.seiws == nil {
		return toolSEIWSNotConfiguredResult(), ConsultarDocumentoSOAPOutput{}, nil
	}

	protocolo := strings.TrimSpace(in.Protocolo)
	if protocolo == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: "protocolo é obrigatório"}},
		}, ConsultarDocumentoSOAPOutput{}, nil
	}

	resp, err := app.seiws.ConsultarDocumento(ctx, protocolo)
	if err != nil {
		var soapErr *soap.Error
		if errors.As(err, &soapErr) {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: soapErr.Error()}},
			}, ConsultarDocumentoSOAPOutput{}, nil
		}
		return nil, ConsultarDocumentoSOAPOutput{}, fmt.Errorf("consultar documento soap: %w", err)
	}

	return nil, ConsultarDocumentoSOAPOutput{Documento: resp.Parametros}, nil
}
