package wssei

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// AuthResponse representa os dados retornados pelo endpoint POST /autenticar.
type AuthResponse struct {
	LoginData     LoginData `json:"loginData"`
	Perfis        []Perfil  `json:"perfis"`
	Unidades      []Unidade `json:"unidades"`
	Identificador string    `json:"identificador"`
	Token         string    `json:"token"`
}

// LoginData reúne os dados de login do usuário autenticado.
type LoginData struct {
	IdSistema               string `json:"IdSistema"`
	IdContexto              string `json:"IdContexto"`
	IdUsuario               string `json:"IdUsuario"`
	IdLogin                 string `json:"IdLogin"`
	HashAgente              string `json:"HashAgente"`
	IdUnidadeAtual          string `json:"IdUnidadeAtual"`
	Sigla                   string `json:"sigla"`
	Nome                    string `json:"nome"`
	IdUltimoCargoAssinatura string `json:"idUltimoCargoAssinatura"`
	UltimoCargoAssinatura   string `json:"ultimoCargoAssinatura"`
}

// Perfil representa um perfil do usuário autenticado.
type Perfil struct {
	IdPerfil string `json:"idPerfil"`
	Nome     string `json:"nome"`
	StAtivo  string `json:"stAtivo"`
}

// Unidade representa uma unidade à qual o usuário autenticado tem acesso.
type Unidade struct {
	Id        string `json:"id"`
	Sigla     string `json:"sigla"`
	Descricao string `json:"descricao"`
}

// Auth é responsável por gerar tokens de autenticação do WSSEI.
type Auth struct {
	endpoint string
}

// NewAuth cria um Auth para a URL base do SEI.
func NewAuth(baseURL string) *Auth {
	return &Auth{
		endpoint: apiBaseURL(baseURL),
	}
}

// Autenticar autentica as credenciais informadas e retorna os dados de autenticação, incluindo o token.
func (a *Auth) Autenticar(ctx context.Context, usuario, senha string, orgao int) (*AuthResponse, error) {
	form := make(url.Values)
	form.Set("usuario", usuario)
	form.Set("senha", senha)
	form.Set("orgao", strconv.Itoa(orgao))

	endpoint := a.endpoint + "/autenticar"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[AuthResponse]
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return nil, fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	if env.Data.Token == "" {
		return nil, fmt.Errorf("no token")
	}

	return &env.Data, nil
}
