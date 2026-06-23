package chatbot

import "errors"

// ErrInvalidToken é o erro retornado quando um token de cadastro do chatbot é
// inválido ou está expirado.
var ErrInvalidToken = errors.New("token is invalid or expired")
