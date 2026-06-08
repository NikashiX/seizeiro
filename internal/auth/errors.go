package auth

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUsuarioNotFound é o erro retornado quando um usuário não é encontrado.
	ErrUsuarioNotFound = errors.New("usuario not found")

	// ErrCPFTaken é o erro retornado quando há um CPF duplicado no banco de dados.
	ErrCPFTaken = errors.New("cpf is already taken")

	// ErrEmailTaken é o erro retornado quando há um email duplicado no banco de dados.
	ErrEmailTaken = errors.New("email is already taken")

	// ErrInvalidToken é o erro retornado para tokens inválidos ou expirados.
	ErrInvalidToken = errors.New("token is invalid or expired")

	// ErrNoSenha é retornado quando há tentativa de uso de senha em usuários que não possuem uma cadastrada.
	ErrNoSenha = errors.New("usuario has no password")

	// ErrInvalidCredentials é o erro retornado quando o CPF ou senha de um usuário são inválidos.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrForbiddenDomain é o erro retornado para tentativa de criar usuários com domínios não permitidos.
	ErrForbiddenDomain = errors.New("email domain is fobidden")
)

// WeakPasswordError é o erro retornado quando a senha informado por um usuário é fraca.
type WeakPasswordError struct {
	Violations []string
}

// Error implementa a interface [error].
func (w *WeakPasswordError) Error() string {
	return "password is too weak"
}

// Description retorna uma descrição amigável para o usuário final com os erros.
func (w *WeakPasswordError) Description() string {
	n := len(w.Violations)
	if n == 0 {
		return "A senha informada é inválida."
	}

	var description string
	switch n {
	case 1:
		description = w.Violations[0]
	case 2:
		description = w.Violations[0] + " e " + w.Violations[1]
	default:
		description = strings.Join(w.Violations[:n-1], ", ")
		description = description + " e " + w.Violations[n-1]
	}

	return fmt.Sprintf("A senha é muito fraca. Ela deve ter %s.", description)
}
