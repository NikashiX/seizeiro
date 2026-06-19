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
//
// Algumas respostas do WSSEI também enviam o objeto embrulhado em um array
// (ex: [{...}]). Nesses casos o primeiro elemento é usado como Value; se o
// array vier vazio, Valid permanece false.
func (o *Object[T]) UnmarshalJSON(data []byte) error {
	var zero T
	trimmed := bytes.TrimSpace(data)
	switch string(trimmed) {
	case "", `""`, "[]", "null":
		o.Value = zero
		o.Valid = false
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var items []T
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		if len(items) == 0 {
			o.Value = zero
			o.Valid = false
			return nil
		}
		o.Value = items[0]
		o.Valid = true
		return nil
	}
	if err := json.Unmarshal(trimmed, &o.Value); err != nil {
		return err
	}
	o.Valid = true
	return nil
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
