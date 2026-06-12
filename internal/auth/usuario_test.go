package auth

import (
	"errors"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/security"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

func TestCreateUsuario(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	usuario := f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da   Silva",
		CPF:   "123.456.789-09",
		Email: "Fulano.Silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	})

	if want := "Fulano da Silva"; usuario.Nome != want {
		t.Fatalf("want %q, got %q", want, usuario.Nome)
	}
	if want := "12345678909"; usuario.CPF != want {
		t.Fatalf("want %q, got %q", want, usuario.CPF)
	}
	if want := "fulano.silva@planejamento.mg.gov.br"; usuario.Email != want {
		t.Fatalf("want %q, got %q", want, usuario.Email)
	}
	if usuario.HashSenha == nil {
		t.Fatal("usuario should have a password")
	}

	// A notificação de ativação de conta deve ser enviada.
	if f.notifier.calls != 1 {
		t.Fatalf("want 1 notifier call, got %d", f.notifier.calls)
	}
	if f.notifier.emailAddress != usuario.Email {
		t.Fatalf("want notification to %q, got %q", usuario.Email, f.notifier.emailAddress)
	}
	if f.notifier.token == "" {
		t.Fatal("expected non-empty activation token")
	}
}

func TestCreateUsuario_ErrEmailTaken(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	params := CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	}
	f.createUsuario(t, params)

	params.CPF = "529.988.310-28"
	_, err := f.service.CreateUsuario(t.Context(), params)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestCreateUsuario_ErrCPFTaken(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	params := CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	}
	f.createUsuario(t, params)

	params.Email = "fulano.silva2@planejamento.mg.gov.br"
	_, err := f.service.CreateUsuario(t.Context(), params)
	if !errors.Is(err, ErrCPFTaken) {
		t.Fatalf("expected ErrCPFTaken, got %v", err)
	}
}

func TestGetUsuario(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	usuario := f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
	})

	read, err := f.service.GetUsuario(t.Context(), usuario.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(usuario, read); diff != "" {
		t.Fatalf("mismatch:\n%s", diff)
	}
}

func TestGetUsuario_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.GetUsuario(t.Context(), uuid.New())
	if !errors.Is(err, ErrUsuarioNotFound) {
		t.Fatalf("expected ErrUsuarioNotFound, got %v", err)
	}
}

func TestChangeSenha(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	params := CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	}
	usuario := f.createUsuario(t, params)

	// Atualiza senha do usuário
	senhaParams := UpdateSenhaParams{
		UsuarioID:   usuario.ID,
		SenhaAntiga: params.Senha,
		SenhaNova:   "123123Abc",
	}
	err := f.service.ChangeSenha(t.Context(), senhaParams)
	if err != nil {
		t.Fatal(err)
	}

	// Lê os dados do usuário atualizado
	usuario, err = f.service.GetUsuario(t.Context(), usuario.ID)
	if err != nil {
		t.Fatal(err)
	}
	if usuario.HashSenha == nil {
		t.Fatal("expected HashSenha != nil")
	}

	// Verifica se a senha nova equivale ao hash.
	ok, err := security.VerifyPassword(*usuario.HashSenha, senhaParams.SenhaNova)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected hash and senha nova to match")
	}
}

func TestChangeSenha_ErrNoSenha(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	params := CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
	}
	usuario := f.createUsuario(t, params)

	// Atualiza senha do usuário
	senhaParams := UpdateSenhaParams{
		UsuarioID:   usuario.ID,
		SenhaAntiga: params.Senha,
		SenhaNova:   "123123Abc",
	}
	err := f.service.ChangeSenha(t.Context(), senhaParams)
	if !errors.Is(err, ErrNoSenha) {
		t.Fatalf("expected ErrNoSenha, got %v", err)
	}
}
