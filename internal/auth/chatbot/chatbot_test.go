package chatbot

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/auth"
	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ti *database.TestInstance

func TestMain(m *testing.M) {
	ti = database.MustTestInstance()
	code := m.Run()

	if err := ti.Close(context.Background()); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

// testKey é uma chave de criptografia determinística de 32 bytes usada nos testes.
var testKey = []byte("0123456789abcdef0123456789abcdef")

type fixture struct {
	pool    *pgxpool.Pool
	service *Service
}

func newFixture(tb testing.TB) *fixture {
	tb.Helper()

	pool := ti.NewPool(tb)

	service, err := NewService(pool, testKey)
	if err != nil {
		tb.Fatal(err)
	}

	return &fixture{
		pool:    pool,
		service: service,
	}
}

// createToken cria um token de teste, falhando o teste em caso de erro.
func (f *fixture) createToken(tb testing.TB, plataforma, plataformaID string) *Token {
	tb.Helper()

	token, err := f.service.CreateToken(tb.Context(), plataforma, plataformaID)
	if err != nil {
		tb.Fatal(err)
	}
	return token
}

func TestNewService_InvalidKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  []byte
	}{
		{name: "nil", key: nil},
		{name: "too short", key: []byte("short")},
		{name: "too long", key: append([]byte("0123456789abcdef0123456789abcdef"), 'x')},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewService(nil, tt.key)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	token := f.createToken(t, "whatsapp", "5531999999999")

	if token.Plataforma != "whatsapp" {
		t.Errorf("expected plataforma %q, got %q", "whatsapp", token.Plataforma)
	}
	if token.PlataformaID != "5531999999999" {
		t.Errorf("expected plataforma_id %q, got %q", "5531999999999", token.PlataformaID)
	}
	if token.PlainText == "" {
		t.Fatal("expected non-empty token")
	}

	// O token deve ser um base64url válido de 32 bytes aleatórios.
	b, err := base64.RawURLEncoding.DecodeString(token.PlainText)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
	if len(b) != 32 {
		t.Errorf("expected 32 random bytes, got %d", len(b))
	}

	// O token deve expirar em aproximadamente 12 horas.
	ttl := time.Until(token.ExpiraEm)
	if ttl <= 11*time.Hour || ttl > 12*time.Hour {
		t.Errorf("expected ttl of ~12h, got %s", ttl)
	}
}

func TestCreateUsuario(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	token := f.createToken(t, "whatsapp", "5531999999999")

	err := f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      token.PlainText,
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	})
	if err != nil {
		t.Fatal(err)
	}

	usuario, err := f.service.GetUsuario(t.Context(), "whatsapp", "5531999999999")
	if err != nil {
		t.Fatal(err)
	}

	want := &Usuario{
		Plataforma:   "whatsapp",
		PlataformaID: "5531999999999",
		SEIUsuario:   "fulano.silva",
		SEISenha:     "SenhaSecreta123",
	}
	if diff := cmp.Diff(want, usuario, cmpopts.IgnoreFields(Usuario{}, "CriadoEm")); diff != "" {
		t.Fatalf("usuario mismatch:\n%s", diff)
	}
	if usuario.CriadoEm.IsZero() {
		t.Error("expected non-zero criado_em")
	}

	// A senha não deve ser armazenada em texto puro no banco de dados.
	var seiSenha []byte
	row := f.pool.QueryRow(t.Context(), "SELECT sei_senha FROM usuarios_chatbot WHERE plataforma = $1 AND plataforma_id = $2", "whatsapp", "5531999999999")
	if err := row.Scan(&seiSenha); err != nil {
		t.Fatal(err)
	}
	if string(seiSenha) == "SenhaSecreta123" {
		t.Error("sei_senha stored in plaintext")
	}
}

func TestCreateToken_ReplacesPrevious(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// A criação de um novo token deve invalidar o token anterior do mesmo usuário.
	old := f.createToken(t, "whatsapp", "5531999999999")
	f.createToken(t, "whatsapp", "5531999999999")

	err := f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      old.PlainText,
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	})
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestCreateToken_SweepsExpired(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// Token expirado de outro usuário, inserido diretamente para controlar expira_em.
	hash := sha256.Sum256([]byte("token-expirado"))
	err := postgres.New(f.pool).SaveTokenChatbot(t.Context(), postgres.SaveTokenChatbotParams{
		Hash:         hash[:],
		Plataforma:   "telegram",
		PlataformaID: "outro-usuario",
		ExpiraEm:     pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	f.createToken(t, "whatsapp", "5531999999999")

	// O token expirado deve ser removido de forma oportunista.
	var count int
	row := f.pool.QueryRow(t.Context(), "SELECT count(*) FROM tokens_chatbot WHERE hash = $1", hash[:])
	if err := row.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected expired token to be deleted, found %d", count)
	}
}

func TestCreateUsuario_UpdatesExisting(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	token := f.createToken(t, "whatsapp", "5531999999999")
	err := f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      token.PlainText,
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Um novo registro com outro token deve atualizar as credenciais existentes.
	token = f.createToken(t, "whatsapp", "5531999999999")
	err = f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      token.PlainText,
		SEIUsuario: "fulano.santos",
		SEISenha:   "NovaSenha456",
	})
	if err != nil {
		t.Fatal(err)
	}

	usuario, err := f.service.GetUsuario(t.Context(), "whatsapp", "5531999999999")
	if err != nil {
		t.Fatal(err)
	}

	want := &Usuario{
		Plataforma:   "whatsapp",
		PlataformaID: "5531999999999",
		SEIUsuario:   "fulano.santos",
		SEISenha:     "NovaSenha456",
	}
	if diff := cmp.Diff(want, usuario, cmpopts.IgnoreFields(Usuario{}, "CriadoEm")); diff != "" {
		t.Fatalf("usuario mismatch:\n%s", diff)
	}
}

func TestCreateUsuario_TokenSingleUse(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	token := f.createToken(t, "whatsapp", "5531999999999")

	params := CreateUsuarioParams{
		Token:      token.PlainText,
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	}
	if err := f.service.CreateUsuario(t.Context(), params); err != nil {
		t.Fatal(err)
	}

	// O token deve ser consumido após o primeiro uso.
	err := f.service.CreateUsuario(t.Context(), params)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestCreateUsuario_InvalidToken(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	err := f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      "token-inexistente",
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	})
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestCreateUsuario_ExpiredToken(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	// A expiração é verificada pelo relógio do banco de dados, então o token
	// expirado é inserido diretamente, sem passar por CreateToken.
	plainText := "token-expirado"
	hash := sha256.Sum256([]byte(plainText))

	err := postgres.New(f.pool).SaveTokenChatbot(t.Context(), postgres.SaveTokenChatbotParams{
		Hash:         hash[:],
		Plataforma:   "whatsapp",
		PlataformaID: "5531999999999",
		ExpiraEm:     pgtype.Timestamptz{Time: time.Now().Add(-time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = f.service.CreateUsuario(t.Context(), CreateUsuarioParams{
		Token:      plainText,
		SEIUsuario: "fulano.silva",
		SEISenha:   "SenhaSecreta123",
	})
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestGetUsuario_NotFound(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.GetUsuario(t.Context(), "whatsapp", "id-inexistente")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
