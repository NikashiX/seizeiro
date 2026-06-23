package seiws

import "encoding/xml"

// Serie identifica o tipo de documento (série) no SEI.
type Serie struct {
	IDSerie string `xml:"IdSerie" json:"id_serie"`
	Nome    string `xml:"Nome" json:"nome"`
}

// UnidadeElaboradora é a unidade do SEI responsável pela elaboração do
// documento.
type UnidadeElaboradora struct {
	IDUnidade string `xml:"IdUnidade" json:"id_unidade"`
	Sigla     string `xml:"Sigla" json:"sigla"`
	Descricao string `xml:"Descricao" json:"descricao"`
}

// Assinaturas é a coleção de [Assinatura] que constam em um documento.
type Assinaturas struct {
	XMLName xml.Name     `xml:"Assinaturas" json:"-"`
	Itens   []Assinatura `xml:"item" json:"itens"`
}

// Assinatura é uma assinatura registrada em um documento do SEI.
type Assinatura struct {
	Nome        string `xml:"Nome" json:"nome"`
	CargoFuncao string `xml:"CargoFuncao" json:"cargo_funcao"`
	DataHora    string `xml:"DataHora" json:"data_hora"`
	IDUsuario   string `xml:"IdUsuario" json:"id_usuario"`
	IDOrigem    string `xml:"IdOrigem" json:"id_origem"`
	IDOrgao     string `xml:"IdOrgao" json:"id_orgao"`
	Sigla       string `xml:"Sigla" json:"sigla"`
}
