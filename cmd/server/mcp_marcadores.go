package main

import (
	"context"
	"errors"
	"fmt"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerMarcadoresTools adiciona as tools MCP que expõem operações de
// marcador do WSSEI. As tools recebem `plataforma` e `plataforma_id` como
// argumentos; o usuário/senha/órgão do SEI são lidos do cadastro persistido.
func registerMarcadoresTools(server *mcp.Server, app *application) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "marcador_consultar",
		Description: "Consulta o marcador associado a um processo do SEI a partir do protocolo.",
	}, app.toolConsultarMarcador)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "marcador_listar_cores",
		Description: "Lista as cores de marcador disponíveis no SEI e o total de registros.",
	}, app.toolListarCoresMarcador)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "marcador_listar_historico",
		Description: "Lista o histórico de marcadores aplicados a um processo do SEI a partir do protocolo.",
	}, app.toolListarHistoricoMarcador)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "marcador_marcar_processo",
		Description: "Vincula um marcador (id de cor) ao processo do SEI identificado pelo protocolo, com um texto descritivo.",
	}, app.toolMarcarProcesso)
}

// ConsultarMarcadorInput agrupa as entradas da tool [toolConsultarMarcador].
type ConsultarMarcadorInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Protocolo    int    `json:"protocolo" jsonschema:"protocolo do processo no SEI"`
}

// ConsultarMarcadorOutput devolve o marcador associado ao processo consultado.
type ConsultarMarcadorOutput struct {
	Marcador wssei.Marcador `json:"marcador" jsonschema:"marcador atualmente associado ao processo"`
}

func (app *application) toolConsultarMarcador(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ConsultarMarcadorInput,
) (*mcp.CallToolResult, ConsultarMarcadorOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), ConsultarMarcadorOutput{}, nil
		}
		return nil, ConsultarMarcadorOutput{}, err
	}

	marcador, err := client.ConsultarMarcador(ctx, in.Protocolo)
	if err != nil {
		return nil, ConsultarMarcadorOutput{}, fmt.Errorf("consultar marcador: %w", err)
	}

	return nil, ConsultarMarcadorOutput{Marcador: *marcador}, nil
}

// ListarCoresMarcadorInput agrupa as entradas da tool [toolListarCoresMarcador].
type ListarCoresMarcadorInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
}

// ListarCoresMarcadorOutput devolve a lista de cores e o total de registros.
type ListarCoresMarcadorOutput struct {
	Total int                  `json:"total" jsonschema:"total de cores disponíveis no WSSEI"`
	Cores []wssei.MarcadorCor `json:"cores" jsonschema:"cores de marcador retornadas pelo WSSEI"`
}

func (app *application) toolListarCoresMarcador(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ListarCoresMarcadorInput,
) (*mcp.CallToolResult, ListarCoresMarcadorOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), ListarCoresMarcadorOutput{}, nil
		}
		return nil, ListarCoresMarcadorOutput{}, err
	}

	cores, total, err := client.ListarCores(ctx)
	if err != nil {
		return nil, ListarCoresMarcadorOutput{}, fmt.Errorf("listar cores: %w", err)
	}

	return nil, ListarCoresMarcadorOutput{
		Total: total,
		Cores: cores,
	}, nil
}

// ListarHistoricoMarcadorInput agrupa as entradas da tool
// [toolListarHistoricoMarcador].
type ListarHistoricoMarcadorInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Protocolo    int    `json:"protocolo" jsonschema:"protocolo do processo no SEI"`
}

// ListarHistoricoMarcadorOutput devolve o histórico de marcadores do processo.
type ListarHistoricoMarcadorOutput struct {
	Historico []wssei.MarcadorHistorico `json:"historico" jsonschema:"itens do histórico de marcadores do processo"`
}

func (app *application) toolListarHistoricoMarcador(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ListarHistoricoMarcadorInput,
) (*mcp.CallToolResult, ListarHistoricoMarcadorOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), ListarHistoricoMarcadorOutput{}, nil
		}
		return nil, ListarHistoricoMarcadorOutput{}, err
	}

	historico, err := client.ListarHistoricoMarcador(ctx, in.Protocolo)
	if err != nil {
		return nil, ListarHistoricoMarcadorOutput{}, fmt.Errorf("listar histórico de marcador: %w", err)
	}

	return nil, ListarHistoricoMarcadorOutput{Historico: historico}, nil
}

// MarcarProcessoInput agrupa as entradas da tool [toolMarcarProcesso].
type MarcarProcessoInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Protocolo    int    `json:"protocolo" jsonschema:"protocolo do processo no SEI"`
	Texto        string `json:"texto" jsonschema:"texto descritivo associado ao marcador"`
	Marcador     int    `json:"marcador" jsonschema:"id da cor do marcador a ser vinculada ao processo"`
}

// MarcarProcessoOutput sinaliza o sucesso da operação de marcação.
type MarcarProcessoOutput struct {
	Sucesso bool `json:"sucesso" jsonschema:"true quando o marcador foi vinculado ao processo com sucesso"`
}

func (app *application) toolMarcarProcesso(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in MarcarProcessoInput,
) (*mcp.CallToolResult, MarcarProcessoOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), MarcarProcessoOutput{}, nil
		}
		return nil, MarcarProcessoOutput{}, err
	}

	err = client.MarcarProcesso(ctx, in.Protocolo, wssei.MarcadorProcessoParams{
		Texto:    in.Texto,
		Marcador: in.Marcador,
	})
	if err != nil {
		return nil, MarcarProcessoOutput{}, fmt.Errorf("marcar processo: %w", err)
	}

	return nil, MarcarProcessoOutput{Sucesso: true}, nil
}
