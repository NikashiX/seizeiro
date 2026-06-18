package wssei

// Tipos compartilhados pelo Client
type Assunto struct {
	ID                         string `json:"id"`
	CodigoEstruturadoFormatado string `json:"codigoEstruturadoFormatado"`
	Descricao                  string `json:"descricao"`
	Sequencia                  string `json:"sequencia"`
}

type Assuntos struct {
	ID                         string `json:"id"`
	CodigoEstruturadoFormatado string `json:"codigoEstruturadoFormatado"`
	Descricao                  string `json:"descricao"`
	CodigoEstruturado          string `json:"codigoEstruturado"`
}

type Interessado struct {
	ID            string `json:"id"`
	Nome          string `json:"nome"`
	NomeFormatado string `json:"nomeFormatado"`
	Sigla         string `json:"sigla"`
}

type NivelAcessoPermitido1 struct {
	Publico  bool `json:"publico"`
	Restrito bool `josn:"restrito"`
	Sigiloso bool `json:"sigiloso"`
}

type Anexo struct {
	ID           string `json:"id"`
	Unidade      string `json:"unidade"`
	SiglaUnidade string `json:"siglaUnidade"`
	Nome         string `json:"nome"`
	DataInclusao string `json:"dataInclusao"`
	Tamanho      string `json:"tamanho"`
	SiglaUsuario string `json:"siglaUsuario"`
	PodeExcluir  bool   `json:"podeExcluir"`
}

type Status struct {
	SinBloqueado            string `json:"sinBloqueado"`
	DocumentoSigiloso       string `json:"documentoSigiloso"`
	DocumentoRestrito       string `json:"documentoRestrito"`
	DocumentoPublicado      string `json:"documentoPublicado"`
	DocumentoAssinado       string `json:"documentoAssinado"`
	Ciencia                 string `json:"ciencia"`
	DocumentoCancelado      string `json:"documentoCancelado"`
	PodeVisualizarDocumento string `json:"podeVisualizarDocumento"`
	PermiteAssinatura       bool   `json:"permiteAssinatura"` // Booleano sem aspas no JSON
	PermiteAlterar          bool   `json:"permiteAlterar"`    // Booleano sem aspas no JSON
	PodeVisualizarMetadados string `json:"podeVisualizarMetadados"`
	PodeAlterarMetadados    string `json:"podeAlterarMetadados"`
}

type AtributosDocumento struct {
	IDProcedimento     string `json:"idProcedimento"`
	IDProtocolo        string `json:"idProtocolo"`
	ProtocoloFormatado string `json:"protocoloFormatado"`
	Nome               string `json:"nome"`
	Titulo             string `json:"titulo"`
	Tipo               string `json:"tipo"`
	TipoDocumento      string `json:"tipoDocumento"`
	MimeType           string `json:"mimeType"`
	Informacao         string `json:"informacao"`
	Tamanho            string `json:"tamanho"`
	IDUnidade          string `json:"idUnidade"`
	SiglaUnidade       string `json:"siglaUnidade"`
	NomeComposto       string `json:"nomeComposto"`
	TipoConferencia    string `json:"tipoConferencia"`
	Status             Status `json:"status"` // Referenciando a struct Status criada abaixo
}
