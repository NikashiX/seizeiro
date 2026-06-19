package wssei

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Marcador representa o marcador do processo retornada pelo WSSEI.
type Marcador struct {
	IDMarcador   string `json:"idMarcador"`
	IDProtocolo  string `json:"idProtocolo"`
	Texto        string `json:"texto"`
	IDCor        string `json:"idCor"`
	DescricaoCor string `json:"descricaoCor"`
	ArquivoCor   string `json:"arquivoCor"`
}

// MarcadorCor representa a cor de marcador retornada pelo WSSEI.
type MarcadorCor struct {
	ID        string `json:"id"`
	Descricao string `json:"descricao"`
	Arquivo   string `json:"arquivo"`
}

// MarcadorHistorico representa o histórico de marcador do processo pelo WSSEI.
type MarcadorHistorico struct {
	MarcadorAtivo string `json:"marcadorAtivo"`
	Data          string `json:"data"`
	Texto         string `json:"texto"`
	NomeMarcador  string `json:"nomeMarcador"`
	NomeUsuario   string `json:"nomeUsuario"`
	SiglaUsuario  string `json:"siglaUsuario"`
}

// ConsultarMarcador retorna o marcador associado ao processo identificado por protocolo.
func (c *Client) ConsultarMarcador(ctx context.Context, protocolo int) (*Marcador, error) {
	endpoint := fmt.Sprintf("%s/marcador/processo/%d/consultar", c.endpoint, protocolo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[Marcador]
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return nil, fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	return &env.Data, nil
}

// ListarCores retorna a lista de [MarcadorCor] disponíveis e o total de registros.
func (c *Client) ListarCores(ctx context.Context) ([]MarcadorCor, int, error) {
	endpoint := fmt.Sprintf("%s/marcador/cores/listar", c.endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http do: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[[]MarcadorCor]
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return nil, 0, fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	total, err := strconv.Atoi(env.Total)
	if err != nil {
		return nil, 0, fmt.Errorf("parse total %q: %w", env.Total, err)
	}

	return env.Data, total, nil
}
