package config

import (
	"encoding/base64"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// SEI contém as configurações de acesso às APIs do SEI.
type SEI struct {
	BaseURL string `env:"SEI_BASE_URL,notEmpty"`
	// WSURL é o endpoint completo da API SOAP legada (SeiWS.php). Quando vazio,
	// as operações que usam essa API devolvem erro de configuração.
	WSURL string `env:"SEI_WS_URL"`
	// SiglaSistema é o identificador do sistema chamador cadastrado no SEI,
	// usado pela API SOAP legada.
	SiglaSistema string `env:"SEI_SIGLA_SISTEMA"`
	// IdentificacaoServico é o token/segredo do sistema cadastrado no SEI,
	// usado pela API SOAP legada.
	IdentificacaoServico string `env:"SEI_IDENTIFICACAO_SERVICO" json:"-"`
}

// ChatbotWebhook contém as configurações do webhook notificado quando o
// cadastro de um usuário do chatbot é concluído com sucesso.
//
// Quando URL está vazia, nenhuma notificação é disparada. Secret, quando
// definido, é enviado no header `key` para o receptor validar a origem da
// chamada (formato esperado pelo credential "Header Auth" do n8n).
type ChatbotWebhook struct {
	URL    string `env:"CHATBOT_WEBHOOK_URL"`
	Secret string `env:"CHATBOT_WEBHOOK_SECRET"`
}

// Config contém as configurações da aplicação.
type Config struct {
	// BaseURL é a URL base do servidor.
	BaseURL string `env:"BASE_URL,notEmpty" envDefault:"http://localhost:4000"`
	// PostgresURL é a URL de conexão com o banco de dados PostgreSQL.
	PostgresURL string `env:"POSTGRES_URL,notEmpty"`
	// SecretKey é a chave secreta para realização de cryptografia simétrica.
	// Deve possuir 32 bytes e usar encoding base64.
	//
	// TODO: Adicionar um gerenciador de chaves com suporte para Azure Key Vault.
	SecretKey string `env:"SECRET_KEY,notEmpty"`

	Production bool `env:"PRODUCTION" envDefault:"true"`

	SEI            SEI
	ChatbotWebhook ChatbotWebhook
}

// NewFromEnv cria uma nova [Config] com base nas variáveis de ambiente definidas no sistema operacional.
func NewFromEnv() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Key retorna o valor de SecretKey sem o encoding.
func (c *Config) Key() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(c.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("b64 decode: %w", err)
	}
	return key, nil
}
