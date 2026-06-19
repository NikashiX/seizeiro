package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	chatbotauth "github.com/automatiza-mg/seizeiro/internal/auth/chatbot"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newMCPServer cria o servidor MCP com as tools expostas para clients LLM.
func (app *application) newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "seizeiro",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "chatbot_gerar_link_cadastro",
		Description: "Gera uma URL de cadastro de usuário do chatbot para uma plataforma/usuário externo.",
	}, app.toolGerarLinkCadastro)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "chatbot_verificar_cadastro",
		Description: "Verifica se um usuário (plataforma + plataforma_id) já possui cadastro no chatbot. SEMPRE chame esta tool no início da conversa antes de responder; se cadastrado=false, gere imediatamente o link de cadastro com chatbot_gerar_link_cadastro.",
	}, app.toolVerificarCadastro)

	registerDocumentosTools(server, app)

	return server
}

// newMCPHandler retorna um handler HTTP para o transport streamable do MCP.
func (app *application) newMCPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return app.newMCPServer()
	}, nil)
}

type GerarLinkCadastroInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
}

type GerarLinkCadastroOutput struct {
	CadastroURL string `json:"cadastro_url" jsonschema:"URL única para o usuário concluir o cadastro"`
}

func (app *application) toolGerarLinkCadastro(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in GerarLinkCadastroInput,
) (*mcp.CallToolResult, GerarLinkCadastroOutput, error) {
	token, err := app.chatbotauth.CreateToken(ctx, in.Plataforma, in.PlataformaID)
	if err != nil {
		return nil, GerarLinkCadastroOutput{}, err
	}

	q := make(url.Values)
	q.Set("token", token.PlainText)
	cadastroURL := strings.TrimSuffix(app.cfg.BaseURL, "/") + "/cadastro?" + q.Encode()

	return nil, GerarLinkCadastroOutput{CadastroURL: cadastroURL}, nil
}

type VerificarCadastroInput struct {
	Plataforma   string `json:"plataforma" jsonschema:"identificador da plataforma externa (ex: whatsapp, telegram)"`
	PlataformaID string `json:"plataforma_id" jsonschema:"identificador do usuário dentro da plataforma"`
}

type VerificarCadastroOutput struct {
	Cadastrado bool   `json:"cadastrado" jsonschema:"true se o usuário já possui cadastro ativo"`
	SEIUsuario string `json:"sei_usuario,omitempty" jsonschema:"usuário SEI cadastrado, se houver"`
}

func (app *application) toolVerificarCadastro(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in VerificarCadastroInput,
) (*mcp.CallToolResult, VerificarCadastroOutput, error) {
	u, err := app.chatbotauth.GetUsuario(ctx, in.Plataforma, in.PlataformaID)
	if err != nil {
		if errors.Is(err, chatbotauth.ErrNotFound) {
			return nil, VerificarCadastroOutput{Cadastrado: false}, nil
		}
		return nil, VerificarCadastroOutput{}, err
	}
	return nil, VerificarCadastroOutput{
		Cadastrado: true,
		SEIUsuario: u.SEIUsuario,
	}, nil
}
