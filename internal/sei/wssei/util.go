package wssei

import (
	"bytes"
	"encoding/json"
)

// Object embrulha um valor que o WSSEI pode enviar como objeto ou, quando
// ausente, como string vazia "", array vazio [] ou null.
//
// Segue o padrão dos tipos sql.Null: Valid indica se há um valor presente e,
// em caso negativo, Value permanece zerado.
type Object[T any] struct {
	// Value é o valor decodificado. Só é significativo quando Valid é true.
	Value T
	// Valid indica se o WSSEI enviou um valor de fato, em vez de uma das formas
	// vazias.
	Valid bool
}

// UnmarshalJSON decodifica o objeto em Value e marca Valid como true. Se o
// WSSEI enviar uma das formas vazias ("", [] ou null), Value é zerado e Valid
// fica false.
func (o *Object[T]) UnmarshalJSON(data []byte) error {
	var zero T
	switch string(bytes.TrimSpace(data)) {
	case "", `""`, "[]", "null":
		o.Value = zero
		o.Valid = false
		return nil
	default:
		if err := json.Unmarshal(data, &o.Value); err != nil {
			return err
		}
		o.Valid = true
		return nil
	}
}

// Slice é uma lista que o WSSEI pode enviar como array ou, quando vazia, como
// string vazia "" ou null. Nesses casos vazios decodifica para um slice nil.
type Slice[T any] []T

// UnmarshalJSON decodifica o array nos elementos ou, se o WSSEI enviar uma das
// formas vazias ("", {} ou null), mantém o slice nil.
func (s *Slice[T]) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "", `""`, "{}", "null":
		*s = nil
		return nil
	default:
		return json.Unmarshal(data, (*[]T)(s))
	}
}
