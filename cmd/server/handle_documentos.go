package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/seiws"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/automatiza-mg/seizeiro/internal/soap"
	"github.com/danielgtaylor/huma/v2"
)


func resolveWSSEIClient(
	ctx context.Context,
	app *application,
	authorization string,
) (*wssei.Client, error) {
	token, ok := strings.CutPrefix(authorization, "Bearer ")
	token = strings.TrimSpace(token)
	if !ok || token == "" {
		return nil, huma.Error401Unauthorized("authorization bearer token obrigatório")
	}

	tokenData, err := app.chatbotauth.GetTokenData(ctx, token)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrInvalidToken) {
			return nil, huma.Error401Unauthorized("token inválido ou expirado")
		}
		return nil, fmt.Errorf("get token data: %w", err)
	}

	usuario, err := app.chatbotauth.GetUsuario(ctx, tokenData.Plataforma, tokenData.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return nil, huma.Error404NotFound("usuário do chatbot não cadastrado")
		}
		return nil, fmt.Errorf("get usuario: %w", err)
	}

	return app.wsseiClients.Get(usuario), nil
}

// registerDocumentos registra os endpoints HTTP que expõem as operações de
// documento do módulo WSSEI.
func registerDocumentos(api huma.API, pathPrefix string, app *application) {
	registerGetDocumentoInterno(api, pathPrefix, app)
	registerGetDocumentoExterno(api, pathPrefix, app)
	registerGetDocumentoVisualizar(api, pathPrefix, app)
	registerGetDocumentoAnexo(api, pathPrefix, app)
	registerGetDocumentoTemplate(api, pathPrefix, app)
	registerListDocumentosProcesso(api, pathPrefix, app)
	registerGetDocumentoSOAP(api, pathPrefix, app)
}

// GetDocumentoSOAPRequest contém os parâmetros para consulta de um documento
// pela API SOAP legada do SEI (SeiWS.php).
type GetDocumentoSOAPRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Protocolo     string `path:"protocolo" doc:"Protocolo do documento (ex.: 0000000.00000.0000000/0000-00)"`
}

// GetDocumentoSOAPResponse devolve os metadados de um documento retornados
// pela API SOAP legada do SEI.
type GetDocumentoSOAPResponse struct {
	Body seiws.RetornoConsultaDocumento
}

// registerGetDocumentoSOAP registra o endpoint de consulta de documento pela
// API SOAP legada do SEI (SeiWS.php). Equivalente ao endpoint do projeto
// `automatiza` que usa a versão antiga da API.
//
// Exige o mesmo Bearer token do chatbot usado nas demais rotas de documento:
// o usuário precisa estar cadastrado para chamar a rota, mesmo a chamada SOAP
// em si usando credenciais globais (SiglaSistema/IdentificacaoServico).
func registerGetDocumentoSOAP(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-soap",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/soap/{protocolo}",
		Tags:        []string{"Documentos"},
		Summary:     "Consulta metadados de um documento via API SOAP legada do SEI",
		Description: "Usa o endpoint SeiWS.php (API antiga) para consultar metadados de um documento pelo protocolo. " +
			"Autenticação via Bearer com token do chatbot — exige usuário cadastrado.",
	}, func(ctx context.Context, in *GetDocumentoSOAPRequest) (*GetDocumentoSOAPResponse, error) {
		// Valida o token do chatbot e garante que o usuário está cadastrado.
		// O cliente WSSEI retornado não é usado: a chamada SOAP usa
		// credenciais globais (SiglaSistema/IdentificacaoServico).
		if _, err := resolveWSSEIClient(ctx, app, in.Authorization); err != nil {
			return nil, err
		}

		if app.seiws == nil {
			return nil, huma.Error503ServiceUnavailable(
				"integração com a API SOAP do SEI não configurada (SEI_WS_URL/SEI_SIGLA_SISTEMA/SEI_IDENTIFICACAO_SERVICO)",
			)
		}

		protocolo := strings.TrimSpace(in.Protocolo)
		if protocolo == "" {
			return nil, huma.Error400BadRequest("protocolo obrigatório")
		}

		resp, err := app.seiws.ConsultarDocumento(ctx, protocolo)
		if err != nil {
			var soapErr *soap.Error
			if errors.As(err, &soapErr) {
				return nil, huma.Error400BadRequest(soapErr.Error())
			}
			return nil, fmt.Errorf("consultar documento soap: %w", err)
		}

		return &GetDocumentoSOAPResponse{Body: resp.Parametros}, nil
	})
}

// GetDocumentoInternoRequest contém os parâmetros para consulta de um documento
// interno.
type GetDocumentoInternoRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Protocolo     int    `path:"protocolo" minimum:"1" doc:"Protocolo do documento interno"`
}

// GetDocumentoInternoResponse devolve os metadados de um documento interno.
type GetDocumentoInternoResponse struct {
	Body wssei.DocumentoInterno
}

func registerGetDocumentoInterno(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-interno",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/internos/{protocolo}",
		Tags:        []string{"Documentos"},
		Summary:     "Consulta metadados de um documento interno",
	}, func(ctx context.Context, in *GetDocumentoInternoRequest) (*GetDocumentoInternoResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		doc, err := client.ConsultarDocumentoInterno(ctx, in.Protocolo)
		if err != nil {
			return nil, fmt.Errorf("consultar documento interno: %w", err)
		}

		return &GetDocumentoInternoResponse{Body: *doc}, nil
	})
}

// GetDocumentoExternoRequest contém os parâmetros para consulta de um documento
// externo.
type GetDocumentoExternoRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Protocolo     int    `path:"protocolo" minimum:"1" doc:"Protocolo do documento externo"`
}

// GetDocumentoExternoResponse devolve os metadados de um documento externo.
type GetDocumentoExternoResponse struct {
	Body wssei.DocumentoExterno
}

func registerGetDocumentoExterno(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-externo",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/externos/{protocolo}",
		Tags:        []string{"Documentos"},
		Summary:     "Consulta metadados de um documento externo",
	}, func(ctx context.Context, in *GetDocumentoExternoRequest) (*GetDocumentoExternoResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		doc, err := client.ConsultarDocumentoExterno(ctx, in.Protocolo)
		if err != nil {
			return nil, fmt.Errorf("consultar documento externo: %w", err)
		}

		return &GetDocumentoExternoResponse{Body: *doc}, nil
	})
}

// GetDocumentoVisualizarRequest contém os parâmetros para obter o HTML de
// visualização de um documento interno.
type GetDocumentoVisualizarRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Documento     int    `path:"documento" minimum:"1" doc:"ID do documento interno"`
}

// GetDocumentoVisualizarResponse devolve o HTML renderizado de um documento
// interno.
type GetDocumentoVisualizarResponse struct {
	Body []byte `contentType:"text/html; charset=utf-8"`
}

func registerGetDocumentoVisualizar(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-visualizar",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/internos/{documento}/visualizar",
		Tags:        []string{"Documentos"},
		Summary:     "Retorna o HTML de visualização de um documento interno",
	}, func(ctx context.Context, in *GetDocumentoVisualizarRequest) (*GetDocumentoVisualizarResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		html, err := client.VisualizarDocumento(ctx, in.Documento)
		if err != nil {
			return nil, fmt.Errorf("visualizar documento: %w", err)
		}

		return &GetDocumentoVisualizarResponse{Body: []byte(html)}, nil
	})
}

// GetDocumentoAnexoRequest contém os parâmetros para baixar o conteúdo binário
// de um documento externo.
type GetDocumentoAnexoRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Protocolo     int    `path:"protocolo" minimum:"1" doc:"ID interno do documento externo (idProtocolo). Não usar o protocolo formatado (ex.: 0107523)."`
}

func registerGetDocumentoAnexo(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-anexo",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/anexos/{protocolo}/conteudo",
		Tags:        []string{"Documentos"},
		Summary:     "Baixa o conteúdo binário de um documento externo (anexo)",
	}, func(ctx context.Context, in *GetDocumentoAnexoRequest) (*huma.StreamResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		body, contentType, err := client.BaixarAnexo(ctx, in.Protocolo)
		if err != nil {
			return nil, fmt.Errorf("baixar anexo: %w", err)
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

// GetDocumentoTemplateRequest contém os parâmetros para pesquisa do template de
// tipo de documento.
type GetDocumentoTemplateRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	ID            int    `query:"id" doc:"ID do tipo de documento"`
	Procedimento  int    `query:"procedimento" doc:"ID do procedimento"`
}

// GetDocumentoTemplateResponse devolve o template do tipo de documento.
type GetDocumentoTemplateResponse struct {
	Body wssei.TemplateDocumento
}

func registerGetDocumentoTemplate(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "get-documento-template",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/documentos/tipos/template",
		Tags:        []string{"Documentos"},
		Summary:     "Pesquisa o template de tipo de documento",
	}, func(ctx context.Context, in *GetDocumentoTemplateRequest) (*GetDocumentoTemplateResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		tpl, err := client.PesquisarTipoTemplateDocumento(ctx, in.ID, in.Procedimento)
		if err != nil {
			return nil, fmt.Errorf("pesquisar tipo template documento: %w", err)
		}

		return &GetDocumentoTemplateResponse{Body: *tpl}, nil
	})
}

// ListDocumentosProcessoRequest contém os parâmetros para listagem paginada dos
// documentos de um processo.
type ListDocumentosProcessoRequest struct {
	Authorization string `header:"Authorization" required:"true" doc:"Bearer <token-do-chatbot>"`
	Procedimento  int    `path:"procedimento" minimum:"1" doc:"ID do processo"`
	Limit         int    `query:"limit" minimum:"0" doc:"Limite de registros por página"`
	Start         int    `query:"start" minimum:"0" doc:"Página inicial (offset)"`
}

// ListDocumentosProcessoResponse devolve a página de documentos e o total
// reportado pelo WSSEI.
type ListDocumentosProcessoResponse struct {
	Body struct {
		Documentos []wssei.Documento `json:"documentos"`
		Total      int               `json:"total"`
	}
}

func registerListDocumentosProcesso(api huma.API, pathPrefix string, app *application) {
	huma.Register(api, huma.Operation{
		OperationID: "list-documentos-processo",
		Method:      http.MethodGet,
		Path:        pathPrefix + "/processos/{procedimento}/documentos",
		Tags:        []string{"Documentos"},
		Summary:     "Lista os documentos de um processo",
	}, func(ctx context.Context, in *ListDocumentosProcessoRequest) (*ListDocumentosProcessoResponse, error) {
		client, err := resolveWSSEIClient(ctx, app, in.Authorization)
		if err != nil {
			return nil, err
		}

		docs, total, err := client.ListarDocumentosProcessos(ctx, wssei.ListarDocumentosParams{
			Limit:        in.Limit,
			Start:        in.Start,
			Procedimento: in.Procedimento,
		})
		if err != nil {
			return nil, fmt.Errorf("listar documentos processo: %w", err)
		}

		var resp ListDocumentosProcessoResponse
		resp.Body.Documentos = docs
		resp.Body.Total = total
		return &resp, nil
	})
}
