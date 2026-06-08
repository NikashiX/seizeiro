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

	service := newTestService(t)

	usuario, err := service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-01",
		Email: "fulano.silva@planejamento.mg.gov.br",
	})
	if err != nil {
		t.Fatal(err)
	}

	token, err := service.CreateToken(t.Context(), CreateTokenParams{
		UsuarioID: usuario.ID,
		Escopo:    EscopoAuth,
		TTL:       time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	read, err := service.GetTokenOwner(t.Context(), token.PlainText, EscopoAuth)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(usuario, read); diff != "" {
		t.Fatalf("mismatch:\n%s", diff)
	}
}

func TestGetTokenOwner_Expired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service := newTestService(t)

		usuario, err := service.CreateUsuario(t.Context(), CreateUsuarioParams{
			Nome:  "Fulano da Silva",
			CPF:   "123.456.789-01",
			Email: "fulano.silva@planejamento.mg.gov.br",
		})
		if err != nil {
			t.Fatal(err)
		}

		ttl := time.Hour
		token, err := service.CreateToken(t.Context(), CreateTokenParams{
			UsuarioID: usuario.ID,
			Escopo:    EscopoAuth,
			TTL:       ttl,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Tempo em que o token já vai estar expirado.
		time.Sleep(ttl + 5*time.Second)

		_, err = service.GetTokenOwner(t.Context(), token.PlainText, EscopoAuth)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})
}
