package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword gera o hash de uma senha usando um algoritmo adequado.
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("generate from password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compara o hash de uma senha com uma senha e retorna se as duas são equivalentes.
//
//	ok, _ := VerifyPassword(pwdHash, pwd)
//	if !ok {
//		// Senha inválida
//	}
func VerifyPassword(pwdHash, pwd string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(pwdHash), []byte(pwd))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("compare hash and password: %w", err)
	}
	return true, nil
}
