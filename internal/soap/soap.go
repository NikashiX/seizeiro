// Package soap contém tipos auxiliares para serializar e deserializar
// envelopes SOAP utilizados pelas APIs legadas do SEI.
package soap

import (
	"encoding/xml"
	"fmt"
)

// Envelope representa o envelope SOAP raiz.
type Envelope[T any] struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    Body[T]  `xml:"Body"`
}

// Body é o corpo do envelope SOAP, contendo o payload tipado T.
type Body[T any] struct {
	XMLName xml.Name `xml:"Body"`
	Content T
}

// Fault representa um erro retornado pelo servidor SOAP.
type Fault struct {
	XMLName xml.Name    `xml:"Fault"`
	Code    string      `xml:"faultcode"`
	Message string      `xml:"faultstring"`
	Detail  FaultDetail `xml:"detail"`
}

// FaultDetail contém os detalhes adicionais de um [Fault].
type FaultDetail struct {
	Items []FaultDetailItem `xml:"item"`
}

// FaultDetailItem é um par chave/valor de detalhes do erro.
type FaultDetailItem struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

// Error é o tipo de erro retornado pelo cliente SOAP quando a resposta possui
// status diferente de 200 ou contém um Fault.
type Error struct {
	Status int
	Fault  Envelope[Fault]
}

// Error implementa a interface error.
func (e *Error) Error() string {
	return fmt.Sprintf("falha ao executar ação (%d): %s", e.Status, e.Fault.Body.Content.Message)
}

// NewError cria um novo [*Error] a partir do status HTTP e do envelope de
// Fault decodificado.
func NewError(status int, fault Envelope[Fault]) *Error {
	return &Error{
		Status: status,
		Fault:  fault,
	}
}
