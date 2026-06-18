package wssei

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListarAtividades retorna a lista de Atividades do Processo.
func (c *Client) ListarAtividades(
	ctx context.Context,
	procedimento int,
	limit int,
	start int,
) (*Atividades, error) {
	url := fmt.Sprintf(
		"%s/atividade/listar",
		c.endpoint,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro response: %w", err)
	}
	defer resp.Body.Close()

	var result Envelope[Atividades]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return nil, fmt.Errorf("Erro ao Listar %d : %s", procedimento, result.Mensagem)
	}

	return &result.Data, nil

}

// Atividades tipo utilizado na funcao "ListarAtividades".
type Atividades struct {
	Id         string `json:"id"`
	Atributos  string `json:"atributos"`
	IdProcesso string `json:"idProcesso"`
	Usuario    string `json:"usuario"`
	Data       string `json:"data"`
	Hora       string `json:"hora"`
	Unidade    string `json:"unidade"`
	Informacao string `json:"informacao"`
	Total      string `json:"total"`
}
