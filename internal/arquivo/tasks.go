package arquivo

type ExtractArgs struct {
	ArquivoID int64 `json:"arquivo_id"`
}

func (args ExtractArgs) Kind() string {
	return "arquivo:extract"
}
