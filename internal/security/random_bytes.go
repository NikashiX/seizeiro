package security

import "crypto/rand"

// RandomBytes retorns n bytes aleatórios criptograficamente seguros, podendo ser usado para gerar tokens ou chaves
// de API.
//
// O valor de n deve ser maior que 0 ou um panic será causado.
func RandomBytes(n int) []byte {
	if n <= 0 {
		panic("invalid random bytes length")
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
