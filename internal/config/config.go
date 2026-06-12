package config

import "github.com/caarlos0/env/v11"

// DocumentIntelligence contém as configurações necessárias para o client do pacote docintel.
type DocumentIntelligence struct {
	Key      string `env:"AZURE_DOCINTEL_KEY"`
	Endpoint string `env:"AZURE_DOCINTEL_ENDPOINT"`
}

// Storage contém as configurações de armazenamento de objetos.
//
// Quando AzureAccount está definido, usa o Azure Blob Storage; caso contrário,
// usa o sistema de arquivos no diretório FilesystemRoot.
type Storage struct {
	AzureAccount   string `env:"STORAGE_AZURE_ACCOUNT"`
	AzureContainer string `env:"STORAGE_AZURE_CONTAINER"`
	FilesystemRoot string `env:"STORAGE_FILESYSTEM_ROOT" envDefault:".blob"`
}

// OpenAI contém as configurações necessárias para o embedder do pacote llm.
type OpenAI struct {
	BaseURL        string `env:"OPENAI_BASE_URL,notEmpty"`
	APIKey         string `env:"OPENAI_API_KEY,notEmpty"`
	EmbeddingModel string `env:"OPENAI_EMBEDDING_MODEL" envDefault:"text-embedding-3-small"`
	// EmbeddingDimensions deve casar com a dimensão da coluna VECTOR usada no schema.
	EmbeddingDimensions int `env:"OPENAI_EMBEDDING_DIMENSIONS" envDefault:"1536"`
	// EmbeddingBatchSize limita a quantidade de textos enviados por requisição.
	EmbeddingBatchSize int `env:"OPENAI_EMBEDDING_BATCH_SIZE" envDefault:"256"`
}

// SMTP contém as configurações do servidor de e-mail.
type SMTP struct {
	Host        string `env:"SMTP_HOST" envDefault:"localhost"`
	Port        int    `env:"SMTP_PORT" envDefault:"1025"`
	User        string `env:"SMTP_USER"`
	Password    string `env:"SMTP_PASSWORD"`
	FromAddress string `env:"SMTP_FROM_ADDRESS" envDefault:"notificacoes@planejamento.mg.gov.br"`
}

// Config contém as configurações da aplicação.
type Config struct {
	// ClientURL é a URL base do frontend da aplicação.
	ClientURL string `env:"CLIENT_URL,notEmpty" envDefault:"http://localhost:5173"`
	// PostgresURL é a URL de conexão com o banco de dados PostgreSQL.
	PostgresURL string `env:"POSTGRES_URL,notEmpty"`

	DocIntel DocumentIntelligence
	OpenAI   OpenAI
	SMTP     SMTP
	Storage  Storage
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
