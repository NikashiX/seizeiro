// Package seiws contém um client HTTP/SOAP para a API legada do SEI
// (SeiWS.php), usada apenas para operações que ainda não têm equivalente no
// módulo WSSEI (REST).
package seiws

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/automatiza-mg/seizeiro/internal/soap"
)

// Config contém as credenciais e endpoint necessários para acessar a API SOAP
// do SEI. Estas credenciais são fixas por aplicação (servidor) e não por
// usuário.
type Config struct {
	// URL é o endpoint completo do SeiWS.php (ex.:
	// https://www.sei.mg.gov.br/sei/ws/SeiWS.php).
	URL string
	// SiglaSistema é o identificador do sistema chamador cadastrado no SEI.
	SiglaSistema string
	// IdentificacaoServico é o token/segredo fornecido pelo SEI para o sistema
	// cadastrado.
	IdentificacaoServico string
}

// Client executa chamadas SOAP contra a API legada do SEI.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient cria um novo [*Client] usando o [http.DefaultClient].
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: http.DefaultClient,
	}
}

// Parametros é o wrapper XML usado pelas respostas SOAP do SEI que retornam
// listas (ex.: parametros/item).
type Parametros[T any] struct {
	Items []T `xml:"item"`
}

// makeSoapError decodifica o corpo da resposta como um envelope SOAP de Fault
// e devolve o erro tipado correspondente.
func makeSoapError(status int, r io.Reader) error {
	var fault soap.Envelope[soap.Fault]
	if err := xml.NewDecoder(r).Decode(&fault); err != nil {
		return err
	}
	return soap.NewError(status, fault)
}

// doReq monta o envelope SOAP a partir de Req, envia para c.cfg.URL e
// decodifica a resposta em Res.
func doReq[Req any, Res any](ctx context.Context, c *Client, req Req) (*Res, error) {
	body, err := xml.Marshal(soap.Envelope[Req]{
		Body: soap.Body[Req]{
			Content: req,
		},
	})
	if err != nil {
		return nil, err
	}

	reqBody := xml.Header + string(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	res, err := c.http.Do(r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	res.Body = io.NopCloser(bytes.NewReader(b))
	if res.StatusCode != http.StatusOK {
		return nil, makeSoapError(res.StatusCode, res.Body)
	}

	var resp soap.Envelope[Res]
	if err := xml.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp.Body.Content, nil
}
