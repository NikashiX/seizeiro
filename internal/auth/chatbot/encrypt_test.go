package chatbot

import (
	"bytes"
	"testing"
)

// newEncryptService cria um Service apenas para os testes de criptografia, sem banco de dados.
func newEncryptService(tb testing.TB, key []byte) *Service {
	tb.Helper()

	service, err := NewService(nil, key)
	if err != nil {
		tb.Fatal(err)
	}
	return service
}

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()
	s := newEncryptService(t, testKey)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{name: "empty", plaintext: []byte{}},
		{name: "short", plaintext: []byte("abc")},
		{name: "senha", plaintext: []byte("SenhaSecreta123")},
		{name: "binary", plaintext: []byte{0x00, 0xff, 0x10, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ciphertext, err := s.encrypt(tt.plaintext)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(ciphertext, tt.plaintext) && len(tt.plaintext) > 0 {
				t.Error("ciphertext contains plaintext")
			}

			plaintext, err := s.decrypt(ciphertext)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(tt.plaintext, plaintext) {
				t.Errorf("expected %q, got %q", tt.plaintext, plaintext)
			}
		})
	}
}

func TestEncrypt_DistinctCiphertexts(t *testing.T) {
	t.Parallel()
	s := newEncryptService(t, testKey)

	// O nonce aleatório deve garantir que o mesmo plaintext produza ciphertexts diferentes.
	a, err := s.encrypt([]byte("SenhaSecreta123"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.encrypt([]byte("SenhaSecreta123"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Error("expected distinct ciphertexts for the same plaintext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	t.Parallel()
	s := newEncryptService(t, testKey)
	other := newEncryptService(t, []byte("ffffffffffffffffffffffffffffffff"))

	ciphertext, err := s.encrypt([]byte("SenhaSecreta123"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := other.decrypt(ciphertext); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDecrypt_Invalid(t *testing.T) {
	t.Parallel()
	s := newEncryptService(t, testKey)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{name: "empty", ciphertext: []byte{}},
		{name: "shorter than nonce", ciphertext: []byte("short")},
		{name: "nonce only", ciphertext: bytes.Repeat([]byte{0x01}, 12)},
		{name: "tampered", ciphertext: bytes.Repeat([]byte{0x01}, 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := s.decrypt(tt.ciphertext); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
