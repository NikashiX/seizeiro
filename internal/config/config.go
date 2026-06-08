package config

import "github.com/caarlos0/env/v11"

// Config contém as configurações da aplicação.
type Config struct {
	PostgresURL string `env:"POSTGRES_URL,notEmpty"`
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
