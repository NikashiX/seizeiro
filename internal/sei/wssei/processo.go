package wssei

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Processo representa um processo retornado pelo WSSEI.
type Processo struct {
	Id                 string            `json:"id"`
	Status             string            `json:"status"`
	SeiNumMaxDocsPasta string            `json:"seiNumMaxDocsPasta"`
	Atributos          ProcessoAtributos `json:"atributos"`
}

// ProcessoAtributos reúne os atributos detalhados de um [Processo].
type ProcessoAtributos struct {
	IdProcedimento   string                           `json:"idProcedimento"`
	IdProtocolo      string                           `json:"idProtocolo"`
	Numero           string                           `json:"numero"`
	TipoProcesso     string                           `json:"tipoProcesso"`
	Descricao        string                           `json:"descricao"`
	UsuarioAtribuido Object[ProcessoUsuarioAtribuido] `json:"usuarioAtribuido"`
	Unidade          Object[ProcessoUnidade]          `json:"unidade"`
	Ciencias         Slice[ProcessoCiencia]           `json:"ciencias"`
	Marcador         Object[ProcessoMarcador]         `json:"marcador"`
	DadosAbertura    Object[ProcessoDadosAbertura]    `json:"dadosAbertura"`
	Anotacoes        Slice[ProcessoAnotacao]          `json:"anotacoes"`
	Status           ProcessoStatusFlags              `json:"status"`
}

// ProcessoUsuarioAtribuido representa o usuário atribuído a um [Processo].
type ProcessoUsuarioAtribuido struct {
	IdAtividade   string `json:"idAtividade"`
	IdUsuario     string `json:"idUsuario"`
	Sigla         string `json:"sigla"`
	Nome          string `json:"nome"`
	Nomeformatado string `json:"nomeformatado"`
}

// ProcessoUnidade representa a unidade atual de um [Processo].
type ProcessoUnidade struct {
	IdUnidade string `json:"idUnidade"`
	Sigla     string `json:"sigla"`
}

// ProcessoCiencia representa um registro de ciência em um [Processo].
type ProcessoCiencia struct {
	IdProtocolo  string `json:"idProtocolo"`
	IdAtividade  string `json:"idAtividade"`
	Data         string `json:"data"`
	IdUnidade    string `json:"idUnidade"`
	Unidade      string `json:"unidade"`
	SiglaUnidade string `json:"siglaUnidade"`
	IdUsuario    string `json:"idUsuario"`
	SiglaUsuario string `json:"siglaUsuario"`
	NomeUsuario  string `json:"nomeUsuario"`
	Descricao    string `json:"descricao"`
}

// ProcessoMarcador representa o marcador aplicado a um [Processo].
type ProcessoMarcador struct {
	IdMarcador   string `json:"idMarcador"`
	Nome         string `json:"nome"`
	Texto        string `json:"texto"`
	IdCor        string `json:"idCor"`
	DescricaoCor string `json:"descricaoCor"`
	ArquivoCor   string `json:"arquivoCor"`
}

// ProcessoDadosAbertura reúne os dados de abertura de um [Processo].
type ProcessoDadosAbertura struct {
	Info     string                                `json:"info"`
	Unidades Slice[ProcessoDadosAberturaUnidade]   `json:"unidades"`
	Lista    Slice[ProcessoDadosAberturaListaItem] `json:"lista"`
}

// ProcessoDadosAberturaUnidade representa uma unidade nos dados de abertura de um [Processo].
type ProcessoDadosAberturaUnidade struct {
	Id   string `json:"id"`
	Nome string `json:"nome"`
}

// ProcessoDadosAberturaListaItem representa um item da lista nos dados de abertura de um [Processo].
type ProcessoDadosAberturaListaItem struct {
	Sigla string `json:"sigla"`
}

// ProcessoAnotacao representa uma anotação de um [Processo].
type ProcessoAnotacao struct {
	IdAnotacao    string `json:"idAnotacao"`
	IdProtocolo   string `json:"idProtocolo"`
	Descricao     string `json:"descricao"`
	IdUnidade     string `json:"idUnidade"`
	IdUsuario     string `json:"idUsuario"`
	DthAnotacao   string `json:"dthAnotacao"`
	SinPrioridade string `json:"sinPrioridade"`
	StaAnotacao   string `json:"staAnotacao"`
}

// ProcessoStatusFlags reúne os indicadores de situação de um [Processo].
type ProcessoStatusFlags struct {
	DocumentoSigiloso                 string `json:"documentoSigiloso"`
	DocumentoRestrito                 string `json:"documentoRestrito"`
	DocumentoNovo                     string `json:"documentoNovo"`
	DocumentoPublicado                string `json:"documentoPublicado"`
	Anotacao                          string `json:"anotacao"`
	AnotacaoPrioridade                string `json:"anotacaoPrioridade"`
	Ciencia                           string `json:"ciencia"`
	RetornoProgramado                 string `json:"retornoProgramado"`
	RetornoData                       any    `json:"retornoData"`
	RetornoAtrasado                   string `json:"retornoAtrasado"`
	ProcessoAcessadoUsuario           string `json:"processoAcessadoUsuario"`
	ProcessoAcessadoUnidade           string `json:"processoAcessadoUnidade"`
	ProcessoRemocaoSobrestamento      string `json:"processoRemocaoSobrestamento"`
	ProcessoBloqueado                 string `json:"processoBloqueado"`
	ProcessoDocumentoIncluidoAssinado string `json:"processoDocumentoIncluidoAssinado"`
	ProcessoPublicado                 string `json:"processoPublicado"`
	NivelAcessoGlobal                 string `json:"nivelAcessoGlobal"`
	PodeGerenciarCredenciais          string `json:"podeGerenciarCredenciais"`
	ProcessoAberto                    string `json:"processoAberto"`
	ProcessoEmTramitacao              string `json:"processoEmTramitacao"`
	ProcessoSobrestado                string `json:"processoSobrestado"`
	ProcessoAnexado                   string `json:"processoAnexado"`
	PodeReabrirProcesso               string `json:"podeReabrirProcesso"`
	PodeRegistrarAnotacao             string `json:"podeRegistrarAnotacao"`
	PodeRemoverSobrestamento          bool   `json:"podeRemoverSobrestamento"`
	Tipo                              string `json:"tipo"`
	ProcessoGeradoRecebido            string `json:"processoGeradoRecebido"`
}

// TipoBusca representa o tipo de busca usado em [Client.ListarProcessos].
type TipoBusca string

// Tipos de busca aceitos pelo endpoint de listagem de processos.
const (
	TipoBuscaTotal         TipoBusca = "T"
	TipoBuscaParcial       TipoBusca = "P"
	TipoBuscaResumido      TipoBusca = "R"
	TipoBuscaExterno       TipoBusca = "E"
	TipoBuscaAuditoria     TipoBusca = "A"
	TipoBuscaUnidade       TipoBusca = "U"
	TipoBuscaPersonalizado TipoBusca = "Z"
)

// ListarProcessosParams reúne os parâmetros opcionais de [Client.ListarProcessos].
//
// Campos com valor zero (0, "" ou false) são omitidos da requisição.
type ListarProcessosParams struct {
	// Limit é o limite de registros da paginação.
	Limit int
	// Start é a página de início da paginação.
	Start int
	// Filter é a palavra-chave da pesquisa.
	Filter string
	// ID é o id do processo para detalhamento.
	ID int
	// Usuario é o id do usuário de atribuição.
	Usuario int
	// Tipo é o tipo de busca.
	Tipo TipoBusca
	// ApenasMeus, quando verdadeiro, retorna apenas os processos do usuário.
	ApenasMeus bool
}

// Converte os parâmetros em [url.Values], omitindo os campos zerados.
func (p ListarProcessosParams) values() url.Values {
	q := make(url.Values)
	if p.Limit != 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Start != 0 {
		q.Set("start", strconv.Itoa(p.Start))
	}
	if p.Filter != "" {
		q.Set("filter", p.Filter)
	}
	if p.ID != 0 {
		q.Set("id", strconv.Itoa(p.ID))
	}
	if p.Usuario != 0 {
		q.Set("usuario", strconv.Itoa(p.Usuario))
	}
	if p.Tipo != "" {
		q.Set("tipo", string(p.Tipo))
	}
	if p.ApenasMeus {
		q.Set("apenasMeus", "S")
	}
	return q
}

// ListarProcessos retorna a lista de processos e o total de registros,
// aplicando os filtros e a paginação informados em params.
func (c *Client) ListarProcessos(ctx context.Context, params ListarProcessosParams) ([]Processo, int, error) {
	endpoint := c.endpoint + "/processo/listar"
	if q := params.values().Encode(); q != "" {
		endpoint += "?" + q
	}

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

	var env Envelope[[]Processo]
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
