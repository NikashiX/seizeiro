package config

import "github.com/caarlos0/env/v11"

// DocumentIntelligence contém as configurações necessárias para o client do pacote docintel.
type DocumentIntelligence struct {
	Key      string `env:"AZURE_DOCINTEL_KEY"`
	Endpoint string `env:"AZURE_DOCINTEL_ENDPOINT"`
}

// OpenAI contém as configurações necessárias para o embedder do pacote llm.
type OpenAI struct {
	APIKey         string `env:"OPENAI_API_KEY,notEmpty"`
	EmbeddingModel string `env:"OPENAI_EMBEDDING_MODEL" envDefault:"text-embedding-3-small"`
	// EmbeddingDimensions deve casar com a dimensão da coluna VECTOR usada no schema.
	EmbeddingDimensions int `env:"OPENAI_EMBEDDING_DIMENSIONS" envDefault:"1536"`
	// EmbeddingBatchSize limita a quantidade de textos enviados por requisição.
	EmbeddingBatchSize int `env:"OPENAI_EMBEDDING_BATCH_SIZE" envDefault:"256"`
}

// Config contém as configurações da aplicação.
type Config struct {
	PostgresURL string `env:"POSTGRES_URL,notEmpty"`

	DocIntel DocumentIntelligence
	OpenAI   OpenAI
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
