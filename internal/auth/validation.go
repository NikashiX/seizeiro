package auth

import (
	"unicode"
	"unicode/utf8"
)

// ValidatePassword valida a força de uma senha. A senha deve possuir entre 8 a 50 caracteres,
// um dígito, uma letra minúscula e uma maiúscula. Retorna [*WeakPasswordError] caso a senha seja fraca.
func ValidatePassword(senha string) error {
	var (
		lower bool
		upper bool
		digit bool
	)

	var violations []string

	n := utf8.RuneCountInString(senha)
	if n < 8 || n > 50 {
		violations = append(violations, "entre 8 e 50 caracteres")
	}

	for _, r := range senha {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		}
	}

	if !lower {
		violations = append(violations, "uma letra minúscula")
	}
	if !upper {
		violations = append(violations, "uma letra maiúscula")
	}
	if !digit {
		violations = append(violations, "um dígito")
	}

	if len(violations) > 0 {
		return &WeakPasswordError{
			Violations: violations,
		}
	}

	return nil
}
