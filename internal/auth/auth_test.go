package auth

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ti *database.TestInstance

// fakeNotifier implementa Notifier para testes, sem chamadas externas.
// Registra a última notificação enviada para permitir asserções.
type fakeNotifier struct {
	calls        int
	emailAddress string
	token        string
}

func (f *fakeNotifier) SendAtivarConta(ctx context.Context, tx pgx.Tx, emailAddress string, token string) error {
	f.calls++
	f.emailAddress = emailAddress
	f.token = token
	return nil
}

type fixture struct {
	pool     *pgxpool.Pool
	service  *Service
	notifier *fakeNotifier
}

func newFixture(tb testing.TB) *fixture {
	tb.Helper()

	pool := ti.NewPool(tb)
	notifier := &fakeNotifier{}

	return &fixture{
		pool:     pool,
		service:  NewService(pool, notifier),
		notifier: notifier,
	}
}

// createUsuario cria um usuário de teste, falhando o teste em caso de erro.
func (f *fixture) createUsuario(tb testing.TB, params CreateUsuarioParams) *Usuario {
	tb.Helper()

	usuario, err := f.service.CreateUsuario(tb.Context(), params)
	if err != nil {
		tb.Fatal(err)
	}
	return usuario
}

func TestMain(m *testing.M) {
	ti = database.MustTestInstance()
	code := m.Run()

	if err := ti.Close(context.Background()); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestLogin(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	usuario := f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	})

	principal, err := f.service.Login(t.Context(), LoginParams{
		CPF:   "123.456.789-09",
		Senha: "Abc123123",
	})
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(usuario, principal.Usuario); diff != "" {
		t.Fatalf("usuario mismatch:\n%s", diff)
	}
	if principal.Token == nil {
		t.Fatal("expected token, got nil")
	}
	if principal.Token.PlainText == "" {
		t.Fatal("expected non-empty token")
	}

	// O token emitido deve ser válido e pertencer ao usuário autenticado.
	owner, err := f.service.GetTokenOwner(t.Context(), principal.Token.PlainText, EscopoAuth)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(usuario, owner); diff != "" {
		t.Fatalf("token owner mismatch:\n%s", diff)
	}
}

func TestLogin_InvalidCPF(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Login(t.Context(), LoginParams{
		CPF:   "000.000.000-00",
		Senha: "Abc123123",
	})
	if !errors.Is(err, ErrInvalidCPF) {
		t.Fatalf("expected ErrInvalidCPF, got %v", err)
	}
}

func TestLogin_InvalidCredentials_UnknownCPF(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// CPF válido, porém não cadastrado.
	_, err := f.service.Login(t.Context(), LoginParams{
		CPF:   "529.988.310-28",
		Senha: "Abc123123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_InvalidCredentials_WrongPassword(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
		Senha: "Abc123123",
	})

	_, err := f.service.Login(t.Context(), LoginParams{
		CPF:   "123.456.789-09",
		Senha: "SenhaErrada123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_ErrNoSenha(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// Usuário criado sem senha cadastrada.
	f.createUsuario(t, CreateUsuarioParams{
		Nome:  "Fulano da Silva",
		CPF:   "123.456.789-09",
		Email: "fulano.silva@planejamento.mg.gov.br",
	})

	_, err := f.service.Login(t.Context(), LoginParams{
		CPF:   "123.456.789-09",
		Senha: "Abc123123",
	})
	if !errors.Is(err, ErrNoSenha) {
		t.Fatalf("expected ErrNoSenha, got %v", err)
	}
}
