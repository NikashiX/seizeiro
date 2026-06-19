package config

import (
	"encoding/base64"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// SEI contém as configurações de acesso às APIs do SEI.
type SEI struct {
	BaseURL string `env:"SEI_BASE_URL,notEmpty"`
}

// Config contém as configurações da aplicação.
type Config struct {
	// BaseURL é a URL base do servidor.
	BaseURL string `env:"BASE_URL,notEmpty" envDefault:"http://localhost:4000"`
	// ClientURL é a URL base do frontend da aplicação.
	ClientURL string `env:"CLIENT_URL,notEmpty" envDefault:"http://localhost:5173"`
	// PostgresURL é a URL de conexão com o banco de dados PostgreSQL.
	PostgresURL string `env:"POSTGRES_URL,notEmpty"`
	// SecretKey é a chave secreta para realização de cryptografia simétrica.
	// Deve possuir 32 bytes e usar encoding base64.
	//
	// TODO: Adicionar um gerenciador de chaves com suporte para Azure Key Vault.
	SecretKey string `env:"SECRET_KEY,notEmpty"`

	Production bool `env:"PRODUCTION" envDefault:"true"`
	SEI      SEI
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
