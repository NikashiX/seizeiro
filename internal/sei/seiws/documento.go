package seiws

import (
	"context"
	"encoding/xml"
)

// ConsultarDocumentoRequest é o payload da operação consultarDocumento do
// SeiWS.php.
type ConsultarDocumentoRequest struct {
	XMLName                     xml.Name `xml:"Sei consultarDocumento"`
	SiglaSistema                string
	IdentificacaoServico        string
	ProtocoloDocumento          string
	IDUnidade                   int    `xml:"IdUnidade,omitempty"`
	SinRetornarAndamentoGeracao string `xml:",omitempty"`
	SinRetornarAssinaturas      string `xml:",omitempty"`
	SinRetornarPublicacao       string `xml:",omitempty"`
	SinRetornarCampos           string `xml:",omitempty"`
}

// ConsultarDocumentoResponse é o envelope de resposta da operação
// consultarDocumento.
type ConsultarDocumentoResponse struct {
	XMLName    xml.Name                 `xml:"Sei consultarDocumentoResponse"`
	Parametros RetornoConsultaDocumento `xml:"parametros" json:"parametros"`
}

// RetornoConsultaDocumento contém os metadados de um documento retornados
// pela API SOAP do SEI.
type RetornoConsultaDocumento struct {
	IDProcedimento        string             `xml:"IdProcedimento" json:"id_procedimento"`
	ProcedimentoFormatado string             `xml:"ProcedimentoFormatado" json:"procedimento_formatado"`
	IDDocumento           string             `xml:"IdDocumento" json:"id_documento"`
	DocumentoFormatado    string             `xml:"DocumentoFormatado" json:"documento_formatado"`
	NivelAcessoLocal      int                `xml:"NivelAcessoLocal" json:"nivel_acesso_local"`
	NivelAcessoGlobal     int                `xml:"NivelAcessoGlobal" json:"nivel_acesso_global"`
	LinkAcesso            string             `xml:"LinkAcesso" json:"link_acesso"`
	Serie                 Serie              `xml:"Serie" json:"serie"`
	Numero                string             `xml:"Numero" json:"numero"`
	Data                  string             `xml:"Data" json:"data"`
	Descricao             string             `xml:"Descricao" json:"descricao"`
	UnidadeElaboradora    UnidadeElaboradora `xml:"UnidadeElaboradora" json:"unidade_elaboradora"`
	Assinaturas           Assinaturas        `xml:"Assinaturas" json:"assinaturas"`
}

// ConsultarDocumento consulta os metadados de um documento no SEI pela API
// SOAP legada (SeiWS.php).
func (c *Client) ConsultarDocumento(ctx context.Context, protocolo string) (*ConsultarDocumentoResponse, error) {
	return doReq[ConsultarDocumentoRequest, ConsultarDocumentoResponse](ctx, c, ConsultarDocumentoRequest{
		SiglaSistema:           c.cfg.SiglaSistema,
		IdentificacaoServico:   c.cfg.IdentificacaoServico,
		ProtocoloDocumento:     protocolo,
		SinRetornarAssinaturas: "S",
	})
}
