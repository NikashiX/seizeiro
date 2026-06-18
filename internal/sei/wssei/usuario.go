package wssei

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListarUsuarios retorna a lista de Usuários
func (c *Client) ListarUsuarios(
	ctx context.Context,
	limit int,
	start int,
	unidade int,
) (*Usuarios, int, error) {
	url := fmt.Sprintf(
		"%s/usuario/listar",
		c.endpoint,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("erro request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("erro response: %w", err)
	}
	defer resp.Body.Close()

	var result Envelope[Usuarios]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, 0, fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return nil, 0, fmt.Errorf("erro listar ususarios : %s", result.Mensagem)
	}

	total, err := result.getTotal()
	if err != nil {
		return nil, 0, fmt.Errorf("total invalido")
	}

	return &result.Data, total, nil

}

// Usuarios tipo utilizado na funcao "ListarUsuarios"
type Usuarios struct {
	IDUsuario string `json:"id_usuario"`
	Sigla     string `json:"sigla"`
	Nome      string `json:"nome"`
	IDContato string `json:"id_contato"`
	Total     string `json:"total"`
}
