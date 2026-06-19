package wssei

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ErrDocumentoNaoEncontrado é retornado por [Client.ResolverIDDocumentoPorNumero]
// quando nenhum documento corresponde ao número formatado pesquisado.
var ErrDocumentoNaoEncontrado = errors.New("documento não encontrado")

// ErrDocumentoAmbiguo é retornado por [Client.ResolverIDDocumentoPorNumero]
// quando mais de um documento corresponde ao número formatado pesquisado e
// não é possível decidir com segurança qual baixar.
var ErrDocumentoAmbiguo = errors.New("documento ambíguo")

// ConsultarDocumentoInterno retorna os metadados do Documento Interno.
func (c *Client) ConsultarDocumentoInterno(ctx context.Context, protocolo int) (*DocumentoInterno, error) {
	if protocolo <= 0 {
		return nil, fmt.Errorf("protocolo inválido: %d", protocolo)
	}

	url := fmt.Sprintf(
		"%s/documento/interno/consultar/%d",
		c.endpoint,
		protocolo,
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

	var result Envelope[DocumentoInterno]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("erro json decoder: %w", err)
	}
	if result.Sucesso != true {
		return nil, fmt.Errorf("erro consultar: %d : %s", protocolo, result.Mensagem)
	}

	return &result.Data, nil

}

// DocumentoInterno tipo utilizado na funcao "ConsultarDocumentoInterno".
type DocumentoInterno struct {
	NomeDocumento            string `json:"nomeDocumento"`
	Protocolo                string `json:"protocolo"`
	IDDocumento              string `json:"idDocumento"`
	IDSerie                  string `json:"idSerie"`
	NomeSerie                string `json:"nomeSerie"`
	Numero                   string `json:"numero"`
	IDTipoConferencia        string `json:"idTipoConferencia"`
	DescricaoTipoConferencia string `json:"descricaoTipoConferencia"`
	NivelAcesso              string `json:"nivelAcesso"`
	IDHipoteseLegal          string `json:"idHipoteseLegal"`
	NomeHipoteseLegal        string `json:"nomeHipoteseLegal"`
	BaseLegal                string `json:"baseLegal"`
	GrauSigilo               string `json:"grauSigilo"`
	Descricao                string `json:"descricao"`
	DataElaboracao           string `json:"dataElaboracao"`
	Observacao               string `json:"observacao"`

	Assuntos     []Assunto     `json:"assuntos"`
	Interessados []Interessado `json:"interessados"`
	//Destinatarios documentado como string, mas é identico ao Interessados
	Destinatarios       []Interessado `json:"destinatarios"`
	ObservacoesUnidades Slice[string] `json:"observacoesUnidades"`
}

// VisualizarDocumento retorna o HTML do Documento para visualização.
func (c *Client) VisualizarDocumento(ctx context.Context, documento int) (string, error) {
	if documento <= 0 {
		return "", fmt.Errorf("protocolo inválido: %d", documento)
	}

	url := fmt.Sprintf(
		"%s/documento/%d/interno/visualizar",
		c.endpoint,
		documento,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("erro request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro response: %w", err)
	}
	defer resp.Body.Close()

	var result Envelope[string]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return "", fmt.Errorf("falhou %d: %s", documento, result.Mensagem)
	}

	return result.Data, nil
}

// BaixarAnexo baixa um documento externo. Retorna o corpo da requisicao e o Content-Type.
func (c *Client) BaixarAnexo(ctx context.Context, protocolo int) (io.ReadCloser, string, error) {
	if protocolo <= 0 {
		return nil, "", fmt.Errorf("protocolo inválido: %d", protocolo)
	}

	url := fmt.Sprintf(
		"%s/documento/baixar/anexo/%d",
		c.endpoint,
		protocolo,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("erro request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("erro response: %w", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("status error: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	return resp.Body, contentType, nil
}

// PesquisarGeralParams reúne os parâmetros opcionais de [Client.PesquisarGeral].
//
// Campos com valor zero ("" ou 0) são omitidos da requisição.
type PesquisarGeralParams struct {
	// BuscaRapida replica o comportamento da busca rápida da interface web:
	// quando informado, os demais filtros são ignorados pelo WSSEI.
	BuscaRapida string
	// PalavrasChave é o texto da pesquisa avançada.
	PalavrasChave string
	// Limit é o limite de registros da paginação.
	Limit int
	// Start é a página de início da paginação.
	Start int
}

// Converte os parâmetros em [url.Values], omitindo os campos zerados.
func (p PesquisarGeralParams) values() url.Values {
	q := make(url.Values)
	if p.BuscaRapida != "" {
		q.Set("buscaRapida", p.BuscaRapida)
	}
	if p.PalavrasChave != "" {
		q.Set("palavrasChave", p.PalavrasChave)
	}
	if p.Limit != 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Start != 0 {
		q.Set("start", strconv.Itoa(p.Start))
	}
	return q
}

// ResultadoPesquisaGeral representa um item retornado por
// [Client.PesquisarGeral].
//
// O WSSEI pode enviar `documento` como objeto único, como array de objetos ou
// como string vazia. Por isso o campo é uma [ResultadoPesquisaGeralDocumentos]
// que normaliza essas variantes em uma lista.
type ResultadoPesquisaGeral struct {
	IDProcedimento                 string                          `json:"idProcedimento"`
	IDTipoProcedimento             string                          `json:"idTipoProcedimento"`
	NomeTipoProcedimento           string                          `json:"nomeTipoProcedimento"`
	SiglaUnidadeGeradora           string                          `json:"siglaUnidadeGeradora"`
	IDUnidadeGeradora              string                          `json:"idUnidadeGeradora"`
	ProtocoloFormatadoProcedimento string                          `json:"protocoloFormatadoProcedimento"`
	IDUsuarioGerador               string                          `json:"idUsuarioGerador"`
	NomeUsuarioGerador             string                          `json:"nomeUsuarioGerador"`
	SiglaUsuarioGerador            string                          `json:"siglaUsuarioGerador"`
	DataGeracao                    string                          `json:"dataGeracao"`
	Documento                      ResultadoPesquisaGeralDocumentos `json:"documento"`
}

// ResultadoPesquisaGeralDocumentos normaliza o campo `documento` dos resultados
// de pesquisa, que o WSSEI envia ora como objeto único, ora como array, ora
// como string vazia.
type ResultadoPesquisaGeralDocumentos []ResultadoPesquisaGeralDocumento

// UnmarshalJSON aceita objeto único `{...}`, array `[...]` ou as formas vazias
// `""`/`null`/`[]`/`{}`. Sempre produz uma lista (possivelmente vazia).
func (d *ResultadoPesquisaGeralDocumentos) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	switch string(trimmed) {
	case "", `""`, "null", "[]", "{}":
		*d = nil
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var items []ResultadoPesquisaGeralDocumento
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		*d = items
		return nil
	}
	var item ResultadoPesquisaGeralDocumento
	if err := json.Unmarshal(trimmed, &item); err != nil {
		return err
	}
	*d = []ResultadoPesquisaGeralDocumento{item}
	return nil
}

// ResultadoPesquisaGeralDocumento reúne os dados do documento referenciado
// por um [ResultadoPesquisaGeral].
type ResultadoPesquisaGeralDocumento struct {
	IDDocumento                 string `json:"idDocumento"`
	IDSerieDocumento            string `json:"idSerieDocumento"`
	NomeSerieDocumento          string `json:"nomeSerieDocumento"`
	ProtocoloFormatadoDocumento string `json:"protocoloFormatadoDocumento"`
	NumeroDocumento             string `json:"numeroDocumento"`
	StaDocumento                string `json:"staDocumento"`
	DtaGeracao                  string `json:"dtaGeracao"`
}

// PesquisarGeral chama o endpoint `/processo/pesquisar` do WSSEI e retorna a
// lista de resultados e o total.
func (c *Client) PesquisarGeral(ctx context.Context, params PesquisarGeralParams) ([]ResultadoPesquisaGeral, int, error) {
	endpoint := c.endpoint + "/processo/pesquisar"
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

	var env Envelope[[]ResultadoPesquisaGeral]
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return nil, 0, fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	total, err := env.getTotal()
	if err != nil {
		return nil, 0, fmt.Errorf("parse total %q: %w", env.Total, err)
	}

	return env.Data, total, nil
}

// ResolverIDDocumentoPorNumero busca o id interno do documento (`idDocumento`)
// a partir do número formatado exibido na interface do SEI (ex: "0107523").
//
// Internamente consulta [Client.PesquisarGeral] usando `buscaRapida` e filtra
// pelos itens cujo `protocoloFormatadoDocumento` bate exatamente com o número
// informado.
//
// Retorna [ErrDocumentoNaoEncontrado] quando a pesquisa não devolve nenhum
// resultado correspondente e [ErrDocumentoAmbiguo] quando existem múltiplos
// `idDocumento` distintos para o mesmo número.
func (c *Client) ResolverIDDocumentoPorNumero(ctx context.Context, numero string) (int, error) {
	numero = strings.TrimSpace(numero)
	if numero == "" {
		return 0, fmt.Errorf("numero vazio")
	}

	resultados, _, err := c.PesquisarGeral(ctx, PesquisarGeralParams{PalavrasChave: numero})
	if err != nil {
		return 0, fmt.Errorf("pesquisar geral: %w", err)
	}

	// Coleta os ids distintos de documentos cujo protocolo formatado bate
	// exatamente. Vários itens podem repetir o mesmo idDocumento (um documento
	// presente em mais de um processo do resultado), por isso deduplicamos
	// antes de decidir se a busca é ambígua.
	idsDistintos := make(map[string]struct{})
	for _, r := range resultados {
		for _, doc := range r.Documento {
			if doc.IDDocumento == "" {
				continue
			}
			if strings.TrimSpace(doc.ProtocoloFormatadoDocumento) != numero {
				continue
			}
			idsDistintos[doc.IDDocumento] = struct{}{}
		}
	}

	switch len(idsDistintos) {
	case 0:
		return 0, ErrDocumentoNaoEncontrado
	case 1:
		var id string
		for k := range idsDistintos {
			id = k
		}
		parsed, err := strconv.Atoi(id)
		if err != nil {
			return 0, fmt.Errorf("parse idDocumento %q: %w", id, err)
		}
		return parsed, nil
	default:
		return 0, ErrDocumentoAmbiguo
	}
}

// PesquisarTipoTemplateDocumento retorna a lista de Templates do Documento.
func (c *Client) PesquisarTipoTemplateDocumento(ctx context.Context, id int, procedimento int) (*TemplateDocumento, error) {
	url := fmt.Sprintf(
		"%s/documento/tipo/template",
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

	var result Envelope[TemplateDocumento]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return nil, fmt.Errorf("erro ao pesquisar %d : %s", procedimento, result.Mensagem)
	}

	return &result.Data, nil
}

// TemplateDocumento tipo utilizado na funcao "PesquisarTipoTemplateDocumento".
type TemplateDocumento struct {
	Assuntos                     Assuntos              `json:"assuntos"`
	Interessados                 string                `json:"interessados"`
	NivelAcessoPermitido         NivelAcessoPermitido1 `json:"nivelAcessoPermitido"`
	PermiteInteressados          bool                  `json:"permiteInteressados"`
	PermiteDestinatarios         bool                  `json:"permiteDestinatarios"`
	ObrigatoriedadeHipoteseLegal string                `json:"obrigatoriedadeHipoteseLegal"`
	ObrigatoriedadeGrauSigilo    string                `json:"obrigatoriedadeGrauSigilo"`
}

// ConsultarDocumentoExterno consulta o Documento Externo.
func (c *Client) ConsultarDocumentoExterno(ctx context.Context, protocolo int) (*DocumentoExterno, error) {
	if protocolo <= 0 {
		return nil, fmt.Errorf("protocolo inválido: %d", protocolo)
	}

	url := fmt.Sprintf(
		"%s/documento/externo/consultar/%d",
		c.endpoint,
		protocolo,
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

	var result Envelope[DocumentoExterno]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return nil, fmt.Errorf("erro contultar %d : %s", protocolo, result.Mensagem)
	}

	return &result.Data, nil
}

// DocumentoExterno tipo utilizado na funcao "ConsultarDocumentoExterno".
type DocumentoExterno struct {
	NomeDocumento            string  `json:"nomeDocumento"`
	Protocolo                string  `json:"protocolo"`
	IDocumento               string  `json:"idDocumento"`
	IdSerie                  string  `json:"idSerie"`
	NomeSerie                string  `json:"nomeSerie"`
	Numero                   string  `json:"numero"`
	IdTipoConferencia        string  `json:"idTipoConferencia"`
	DescricaoTipoConferencia string  `json:"descricaoTipoConferencia"`
	NivelAcesso              string  `json:"nivelAcesso"`
	IdHipoteseLegal          string  `json:"idHipoteseLegal"`
	NomeHipoteseLegal        string  `json:"nomeHipoteseLegal"`
	GrauSigilo               string  `json:"grauSigilo"`
	Descricao                string  `json:"descricao"`
	DataElaboracao           string  `json:"dataElaboracao"`
	Observacao               string  `json:"observacao"`
	Assuntos                 string  `json:"assuntos"`
	Remetente                string  `json:"remetente"`
	Interessados             string  `json:"interessados"`
	Destinatarios            string  `json:"destinatarios"`
	ObservacoesUnidades      string  `json:"observacoesUnidades"`
	Anexo                    []Anexo `json:"anexo"`
}

// ListarDocumentosParams reúne os parâmetros opcionais de [Client.ListarProcessos].
//
// Campos com valor zero (0, "" ou false) são omitidos da requisição.
type ListarDocumentosParams struct {
	// Limit é o limite de registros da paginação.
	Limit int
	// Start é a página de início da paginação.
	Start int
	//Procedimento é o ID do processo. OBRIGATORIO
	Procedimento int
}

// Converte os parâmetros em [url.Values], omitindo os campos zerados.
func (p ListarDocumentosParams) values() url.Values {
	q := make(url.Values)
	if p.Limit != 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Start != 0 {
		q.Set("start", strconv.Itoa(p.Start))
	}
	return q
}

// ListarDocumentosProcessos retorna a lista de Documentos do Processo.
func (c *Client) ListarDocumentosProcessos(ctx context.Context, params ListarDocumentosParams) ([]Documento, int, error) {
	if params.Procedimento == 0 {
		return nil, 0, fmt.Errorf("procedimento e obrigatorio")
	}
	endpoint := fmt.Sprintf("%s/documento/listar/%d", c.endpoint, params.Procedimento)
	if q := params.values().Encode(); q != "" {
		endpoint += "?" + q
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("erro request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("erro response: %w", err)
	}
	defer resp.Body.Close()

	var result Envelope[[]Documento]

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, 0, fmt.Errorf("erro json decoder: %w", err)
	}

	if result.Sucesso != true {
		return nil, 0, fmt.Errorf("erro listar %d : %s", params.Procedimento, result.Mensagem)
	}

	total, err := result.getTotal()
	if err != nil {
		return nil, 0, fmt.Errorf("total invalido")
	}

	return result.Data, total, nil

}

// Documento tipo utilizado na funcao "ListarDocumentosProcessos"
type Documento struct {
	ID        string             `json:"id"`
	Atributos AtributosDocumento `json:"atributos"`
}
