package arquivo

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("arquivo not found")
	ErrAlreadyExists = errors.New("arquivo already exists")
)

// UnsupportedError é o erro retornado quando não há suporte para um Content-Type.
type UnsupportedError struct {
	ContentType string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("unsupported content-type: %q", e.ContentType)
}
