package auth

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestGetTokenOwner(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	usuario := f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
	})

	token, err := f.service.CreateToken(t.Context(), CreateTokenParams{
		UsuarioID: usuario.ID,
		Escopo:    EscopoAuth,
		TTL:       time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	read, err := f.service.GetTokenOwner(t.Context(), token.PlainText, EscopoAuth)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(usuario, read); diff != "" {
		t.Fatalf("mismatch:\n%s", diff)
	}
}

func TestGetTokenOwner_Expired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFixture(t)

		usuario := f.createUsuario(t, CreateUsuarioParams{
			Nome:  "Fulano da Silva",
			CPF:   "123.456.789-09",
			Email: "fulano.silva@planejamento.mg.gov.br",
		})

		ttl := time.Hour
		token, err := f.service.CreateToken(t.Context(), CreateTokenParams{
			UsuarioID: usuario.ID,
			Escopo:    EscopoAuth,
			TTL:       ttl,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Tempo em que o token já vai estar expirado.
		time.Sleep(ttl + 5*time.Second)

		_, err = f.service.GetTokenOwner(t.Context(), token.PlainText, EscopoAuth)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})
}
