// Package wssei contém um client HTTP para o módulo WSSEI.
//
// Referência: https://pengovbr.github.io/mod-wssei/#/
package wssei

import (
	"net/http"
	"strconv"
	"strings"
)

// O caminho da API do módulo WSSEI relativo à URL base do SEI.
const apiBasePath = "/sei/modulos/wssei/controlador_ws.php/api/v2"

// Monta a URL base da API do WSSEI a partir da URL base do SEI.
func apiBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + apiBasePath
}

// Envelope é o formato padrão de resposta do WSSEI, presente em todas as
// chamadas HTTP.
// O conteúdo útil fica em Data, cujo tipo varia por endpoint.
type Envelope[T any] struct {
	Sucesso  bool   `json:"sucesso"`
	Mensagem string `json:"mensagem"`
	Total    string `json:"total"`
	Data     T      `json:"data"`
}

// Se total vazio, return 0  e sem erro
func (e *Envelope[T]) getTotal() (int, error) {
	if e.Total == "" {
		return 0, nil
	}
	return strconv.Atoi(e.Total)
}

// Config reúne os dados necessários para autenticar e acessar o WSSEI.
type Config struct {
	// BaseURL é a URL base do SEI (ex: https://www.sei.mg.gov.br).
	BaseURL string
	// Usuario é o login do usuário usado na autenticação.
	Usuario string
	// Senha é a senha do usuário usada na autenticação.
	Senha string
	// Orgao é o id do órgão da autenticação.
	Orgao int
}

type Client struct {
	endpoint string
	http     *http.Client
}

// NewClient cria um Client que autentica no WSSEI com usuário e senha,
// gerando e reaproveitando o token automaticamente em cada requisição.
func NewClient(cfg Config) *Client {
	return &Client{
		endpoint: apiBaseURL(cfg.BaseURL),
		http: &http.Client{
			Transport: &tokenTransport{
				RoundTripper: http.DefaultTransport,
				auth:         NewAuth(cfg.BaseURL),
				usuario:      cfg.Usuario,
				senha:        cfg.Senha,
				orgao:        cfg.Orgao,
			},
		},
	}
}
