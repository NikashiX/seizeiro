package arquivo

type ExtractArgs struct {
	ArquivoID int64 `json:"arquivo_id"`
}

func (args ExtractArgs) Kind() string {
	return "arquivo:extract"
}

type ChunkArgs struct {
	ConteudoID int64 `json:"conteudo_id"`
}

func (args ChunkArgs) Kind() string {
	return "arquivo:chunk"
}
