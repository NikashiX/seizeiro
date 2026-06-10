package llm

import (
	"errors"
	"fmt"
)

// ErrInvalidEmbedderConfig é retornado quando as configurações de um embedder são inválidas.
var ErrInvalidEmbedderConfig = errors.New("llm: invalid embedder config")

// DimensionMismatchError é o erro retornado quando a dimensão do embedding gerado
// difere da dimensão esperada, configurada para casar com o schema do banco.
type DimensionMismatchError struct {
	Expected int
	Got      int
}

func (e *DimensionMismatchError) Error() string {
	return fmt.Sprintf("llm: embedding dimension mismatch: expected %d, got %d", e.Expected, e.Got)
}
