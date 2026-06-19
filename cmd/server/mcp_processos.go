package main

import (
	"context"
	"errors"
	"fmt"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/automatiza-mg/seizeiro/internal/sei/wssei"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerProcessosTools adiciona as tools MCP que expõem operações de
// processo do WSSEI. As tools recebem `plataforma` e `plataforma_id` como
// argumentos; o usuário/senha/órgão do SEI são lidos do cadastro persistido.
func registerProcessosTools(server *mcp.Server, app *application) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "processo_listar",
		Description: "Lista processos do SEI aplicando os filtros e a paginação informados. Devolve a lista de processos e o total de registros disponíveis.",
	}, app.toolListarProcessos)
}

// ListarProcessosInput agrupa as entradas da tool [toolListarProcessos].
//
// Os campos opcionais com valor zero são omitidos da requisição ao WSSEI.
type ListarProcessosInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
	Limit        int    `json:"limit,omitempty" jsonschema:"limite de registros da paginação"`
	Start        int    `json:"start,omitempty" jsonschema:"página de início da paginação"`
	Filter       string `json:"filter,omitempty" jsonschema:"palavra-chave da pesquisa"`
	ID           int    `json:"id,omitempty" jsonschema:"id do processo para detalhamento"`
	Usuario      int    `json:"usuario,omitempty" jsonschema:"id do usuário de atribuição"`
	Tipo         string `json:"tipo,omitempty" jsonschema:"tipo de busca (T=total, P=parcial, R=resumido, E=externo, A=auditoria, U=unidade, Z=personalizado)"`
	ApenasMeus   bool   `json:"apenas_meus,omitempty" jsonschema:"quando true, retorna apenas os processos do usuário"`
	Unidade      int    `json:"unidade,omitempty" jsonschema:"id da unidade"`
}

// ListarProcessosOutput devolve a lista de processos e o total de registros
// retornados pelo WSSEI.
type ListarProcessosOutput struct {
	Total     int              `json:"total" jsonschema:"total de registros disponíveis no WSSEI"`
	Processos []wssei.Processo `json:"processos" jsonschema:"processos retornados pela página atual"`
}

func (app *application) toolListarProcessos(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in ListarProcessosInput,
) (*mcp.CallToolResult, ListarProcessosOutput, error) {
	client, err := resolveWSSEIClientByPlataforma(ctx, app, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return toolNotFoundResult(), ListarProcessosOutput{}, nil
		}
		return nil, ListarProcessosOutput{}, err
	}

	processos, total, err := client.ListarProcessos(ctx, wssei.ListarProcessosParams{
		Limit:      in.Limit,
		Start:      in.Start,
		Filter:     in.Filter,
		ID:         in.ID,
		Usuario:    in.Usuario,
		Tipo:       wssei.TipoBusca(in.Tipo),
		ApenasMeus: in.ApenasMeus,
		Unidade:    in.Unidade,
	})
	if err != nil {
		return nil, ListarProcessosOutput{}, fmt.Errorf("listar processos: %w", err)
	}

	return nil, ListarProcessosOutput{
		Total:     total,
		Processos: processos,
	}, nil
}
