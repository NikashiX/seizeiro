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
	if id <= 0 {
		return nil, fmt.Errorf("id invalido: %d", id)
	}
	if procedimento <= 0 {
		return nil, fmt.Errorf("procedimento invalido: %d", procedimento)
	}

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
	Documento int
	Orgao     int
	Cargo     string
	Login     string
	Senha     string
	Usuario   int
}

// AssinarDocumento realiza a assinatura de um documento
func (c *Client) AssinarDocumento(ctx context.Context, params AssinarDocParams) error {
	if params.Documento <= 0 {
		return fmt.Errorf("documento invalido: %d", params.Documento)
	}
	if params.Orgao <= 0 {
		return fmt.Errorf("orgao invalido: %d", params.Orgao)
	}
	if strings.TrimSpace(params.Cargo) == "" {
		return fmt.Errorf("cargo: texto required")
	}
	if strings.TrimSpace(params.Login) == "" {
		return fmt.Errorf("login: texto required")
	}
	if strings.TrimSpace(params.Senha) == "" {
		return fmt.Errorf("senha: texto required")
	}
	if params.Usuario <= 0 {
		return fmt.Errorf("usuario invalido: %d", params.Usuario)
	}

	payload := map[string]interface{}{
		"documento": params.Documento,
		"orgao":     params.Orgao,
		"cargo":     params.Cargo,
		"login":     params.Login,
		"senha":     params.Senha,
		"usuario":   params.Usuario,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/assinar", c.endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro response: %w", err)
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
	Procedimento               int
	IdUnidadeGeradoraProtocolo int
	Numero                     string
	// IdSerie obrigatorio
	IdSerie int
	// DataElaboracao  obrigatotio
	DataElaboracao    string
	IdTipoConferencia int
	Assuntos          string
	Interessados      string
	Remetente         int
	Observacao        string
	// NivelAcesso obrigatorio
	NivelAcesso     int
	IdHipoteseLegal int
	// Anexo obrigatorio
	Anexo string
	// GrauSigilo obrigatotio
	GrauSigilo string
}

// CadastrarDocumentoExterno cadastra um novo documento externo
func (c *Client) CadastrarDocumentoExterno(ctx context.Context, params DocExternoParams) error {
	if params.Procedimento <= 0 {
		return fmt.Errorf("Procedimento invalido: %d", params.Procedimento)
	}
	if params.IdSerie <= 0 {
		return fmt.Errorf("id do documento invalido: %d", params.IdSerie)
	}
	if strings.TrimSpace(params.DataElaboracao) == "" {
		return fmt.Errorf("data: texto required")
	}
	if params.NivelAcesso <= 0 {
		return fmt.Errorf("nivel de acesso invalido: %d", params.NivelAcesso)
	}
	if strings.TrimSpace(params.Anexo) == "" {
		return fmt.Errorf("anexo: texto required")
	}
	if strings.TrimSpace(params.GrauSigilo) == "" {
		return fmt.Errorf("graua sigilo: texto required")
	}

	payload := map[string]interface{}{
		"procedimento":               params.Procedimento,
		"idUnidadeGeradoraProtocolo": params.IdUnidadeGeradoraProtocolo,
		"numero":                     params.Numero,
		"idSerie":                    params.IdSerie,
		"dataElaboracao":             params.DataElaboracao,
		"idTipoConferencia":          params.IdTipoConferencia,
		"assuntos":                   params.Assuntos,
		"interessados":               params.Interessados,
		"remetente":                  params.Remetente,
		"observacao":                 params.Observacao,
		"nivelAcesso":                params.NivelAcesso,
		"idHipoteseLegal":            params.IdHipoteseLegal,
		"anexo":                      params.Anexo,
		"grauSigilo":                 params.GrauSigilo,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/%d/externo/criar", c.endpoint, params.Procedimento)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro response: %w", err)
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
	// Procedimento obrigatorio
	Procedimento               int
	IdUnidadeGeradoraProtocolo int
	// idSerie obrigatorio
	IdSerie      int
	Assuntos     string
	Interessados string
	// Observacao obrigatorio
	Observacao string
	// NivelAcesso obrigatorio
	NivelAcesso              int
	IdHipoteseLegal          int
	IdTextoPadraoInterno     int
	ProtocoloDocumentoModelo string
	Descricao                string
	Destinatarios            string
}

// CadastrarDocumentoInterno cadastra um novo documento interno
func (c *Client) CadastrarDocumentoInterno(ctx context.Context, params DocInternoParams) error {
	if params.Procedimento <= 0 {
		return fmt.Errorf("procedimento invalido: %d", params.Procedimento)
	}
	if params.IdSerie <= 0 {
		return fmt.Errorf("id do documento invalido: %d", params.IdSerie)
	}
	if strings.TrimSpace(params.Observacao) == "" {
		return fmt.Errorf("observacao: texto required")
	}
	if params.NivelAcesso <= 0 {
		return fmt.Errorf("nivel de acesso invalido: %d", params.NivelAcesso)
	}

	payload := map[string]interface{}{
		"procedimento":               params.Procedimento,
		"idUnidadeGeradoraProtocolo": params.IdUnidadeGeradoraProtocolo,
		"idSerie":                    params.IdSerie,
		"assuntos":                   params.Assuntos,
		"interessados":               params.Interessados,
		"observacao":                 params.Observacao,
		"nivelAcesso":                params.NivelAcesso,
		"idHipoteseLegal":            params.IdHipoteseLegal,
		"idTextoPadraoInterno":       params.IdTextoPadraoInterno,
		"protocoloDocumentoModelo":   params.ProtocoloDocumentoModelo,
		"descricao":                  params.Descricao,
		"destinatarios":              params.Destinatarios,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/%d/interno/criar", c.endpoint, params.Procedimento)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro response: %w", err)
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

// usuario tipo utilizado na funcao AssinarBlocoDocumentos
type BlocoDocumentosParams struct {
	// Todos os parametros sao obrigatorios
	arrDocumento string
	orgao        int
	cargo        string
	login        string
	senha        string
	usuario      int
}

// AssinarBlocoDocumentos realiza a assinatura de um ou mais documentos
func (c *Client) AssinarBlocoDocumentos(ctx context.Context, params BlocoDocumentosParams) error {
	if strings.TrimSpace(params.arrDocumento) == "" {
		return fmt.Errorf("arrDocumento: texto required")
	}
	if params.orgao <= 0 {
		return fmt.Errorf("orgao invalido: %d", params.orgao)
	}
	if strings.TrimSpace(params.cargo) == "" {
		return fmt.Errorf("cargo: texto required")
	}
	if strings.TrimSpace(params.login) == "" {
		return fmt.Errorf("login: texto required")
	}
	if strings.TrimSpace(params.senha) == "" {
		return fmt.Errorf("senha: texto required")
	}
	if params.usuario <= 0 {
		return fmt.Errorf("usuario invalido: %d", params.usuario)
	}

	payload := map[string]interface{}{
		"arrDocumento": params.arrDocumento,
		"orgao":        params.orgao,
		"cargo":        params.cargo,
		"login":        params.login,
		"senha":        params.senha,
		"usuario":      params.usuario,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao gerar payload: %w", err)
	}

	url := fmt.Sprintf("%s/documento/assinar/bloco", c.endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("erro request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro response: %w", err)
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
