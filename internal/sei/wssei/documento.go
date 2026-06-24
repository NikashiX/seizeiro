package wssei

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

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

// AssinarDocParams tipo utilizado na funcao AssinarDocumentos
type AssinarDocParams struct {
	// Todos os parametros sao obrigatorios
	documento int
	orgao     int
	cargo     string
	login     string
	senha     string
	usuario   int
}

// AssinarDocumento realiza a assinatura de um documento
func (c *Client) AssinarDocumento(ctx context.Context, params AssinarDocParams) error {
	if params.documento <= 0 {
		return fmt.Errorf("documento invalido: %d", params.documento)
	}
	if params.orgao <= 0 {
		return fmt.Errorf("orgao invalido: %d", params.orgao)
	}
	if strings.TrimSpace(params.cargo) == "" {
		return fmt.Errorf("texto required")
	}
	if strings.TrimSpace(params.login) == "" {
		return fmt.Errorf("texto required")
	}
	if strings.TrimSpace(params.senha) == "" {
		return fmt.Errorf("texto required")
	}
	if params.usuario <= 0 {
		return fmt.Errorf("usuario invalido: %d", params.usuario)
	}

	payload := map[string]interface{}{
		"documento": params.documento,
		"orgao":     params.orgao,
		"cargo":     params.cargo,
		"login":     params.login,
		"senha":     params.senha,
		"usuario":   params.usuario,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/assinar", c.endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		fmt.Errorf("erro response: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[struct{}]
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	return nil
}

// DocExterno tipo utilizado na funcao CadastrarDocumentoExterno
type DocExternoParams struct {
	// procedimento obrigatorio
	procedimento               int
	idUnidadeGeradoraProtocolo int
	numero                     string
	// idSerie obrigatorio
	idSerie int
	// dataElaboracao  obrigatotio
	dataElaboracao    string
	idTipoConferencia int
	assuntos          string
	interessados      string
	remetente         int
	observacao        string
	// nivelAcesso obrigatorio
	nivelAcesso     int
	idHipoteseLegal int
	// anexo obrigatorio
	anexo string
	// grauSigilo obrigatotio
	grauSigilo string
}

// CadastrarDocumentoExterno cadastra um novo documento externo
func (c *Client) CadastrarDocumentoExterno(ctx context.Context, params DocExternoParams) error {
	if params.procedimento <= 0 {
		return fmt.Errorf("procedimento invalido: %d", params.procedimento)
	}
	if params.idSerie <= 0 {
		return fmt.Errorf("id do documento invalido: %d", params.idSerie)
	}
	if strings.TrimSpace(params.dataElaboracao) == "" {
		return fmt.Errorf("texto required")
	}
	if params.nivelAcesso <= 0 {
		return fmt.Errorf("nivel de acesso invalido: %d", params.nivelAcesso)
	}
	if strings.TrimSpace(params.anexo) == "" {
		return fmt.Errorf("texto required")
	}
	if strings.TrimSpace(params.grauSigilo) == "" {
		return fmt.Errorf("texto required")
	}

	payload := map[string]interface{}{
		"procedimento":               params.procedimento,
		"idUnidadeGeradoraProtocolo": params.idUnidadeGeradoraProtocolo,
		"numero":                     params.numero,
		"idSerie":                    params.idSerie,
		"dataElaboracao":             params.dataElaboracao,
		"idTipoConferencia":          params.idTipoConferencia,
		"assuntos":                   params.assuntos,
		"interessados":               params.interessados,
		"remetente":                  params.remetente,
		"observacao":                 params.observacao,
		"nivelAcesso":                params.nivelAcesso,
		"idHipoteseLegal":            params.idHipoteseLegal,
		"anexo":                      params.anexo,
		"grauSigilo":                 params.grauSigilo,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/%d/externo/criar", c.endpoint, params.procedimento)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		fmt.Errorf("erro response: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[struct{}]
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	return nil
}

// DocExterno tipo utilizado na funcao CadastrarDocumentoInterno
type DocInternoParams struct {
	// procedimento obrigatorio
	procedimento               int
	idUnidadeGeradoraProtocolo int
	// idSerie obrigatorio
	idSerie      int
	assuntos     string
	interessados string
	// observacao obrigatorio
	observacao string
	// nivelAcesso obrigatorio
	nivelAcesso              int
	idHipoteseLegal          int
	idTextoPadraoInterno     int
	protocoloDocumentoModelo string
	descricao                string
	destinatarios            string
}

// CadastrarDocumentoInterno cadastra um novo documento interno
func (c *Client) CadastrarDocumentoInterno(ctx context.Context, params DocInternoParams) error {
	if params.procedimento <= 0 {
		return fmt.Errorf("procedimento invalido: %d", params.procedimento)
	}
	if params.idSerie <= 0 {
		return fmt.Errorf("id do documento invalido: %d", params.idSerie)
	}
	if strings.TrimSpace(params.observacao) == "" {
		return fmt.Errorf("texto required")
	}
	if params.nivelAcesso <= 0 {
		return fmt.Errorf("nivel de acesso invalido: %d", params.nivelAcesso)
	}

	payload := map[string]interface{}{
		"procedimento":               params.procedimento,
		"idUnidadeGeradoraProtocolo": params.idUnidadeGeradoraProtocolo,
		"idSerie":                    params.idSerie,
		"assuntos":                   params.assuntos,
		"interessados":               params.interessados,
		"observacao":                 params.observacao,
		"nivelAcesso":                params.nivelAcesso,
		"idHipoteseLegal":            params.idHipoteseLegal,
		"idTextoPadraoInterno":       params.idTextoPadraoInterno,
		"protocoloDocumentoModelo":   params.protocoloDocumentoModelo,
		"descricao":                  params.descricao,
		"destinatarios":              params.destinatarios,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/%d/externo/criar", c.endpoint, params.procedimento)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		fmt.Errorf("erro response: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var env Envelope[struct{}]
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	if !env.Sucesso {
		return fmt.Errorf("invalid response: %s", env.Mensagem)
	}

	return nil
}
