package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/database"
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/automatiza-mg/seizeiro/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ativarContaTokenTTL = 3 * 24 * time.Hour // 72 horas
)

// Anonymous representa um usuário não autenticado.
var Anonymous = &Usuario{}

type Usuario struct {
	ID              uuid.UUID `json:"id"`
	Nome            string    `json:"nome"`
	CPF             string    `json:"cpf"`
	Email           string    `json:"email"`
	EmailVerificado bool      `json:"email_verificado"`
	HashSenha       *string   `json:"-"`
	CriadoEm        time.Time `json:"criado_em"`
	AtualizadoEm    time.Time `json:"atualizado_em"`
}

// IsAnonymous reporta se o usuário é anônimo (não autenticado).
func (u *Usuario) IsAnonymous() bool {
	return u == Anonymous
}

func usuarioFromDB(record postgres.Usuario) Usuario {
	var hashSenha *string
	if record.HashSenha.Valid {
		hashSenha = &record.HashSenha.String
	}
	return Usuario{
		ID:              record.ID.Bytes,
		Nome:            record.Nome,
		CPF:             record.CPF,
		Email:           record.Email,
		EmailVerificado: record.EmailVerificado,
		HashSenha:       hashSenha,
		CriadoEm:        record.CriadoEm.Time,
		AtualizadoEm:    record.AtualizadoEm.Time,
	}
}

// GetUsuario retorna um usuário pelo ID. Se nenhum usuário for encontrado, retorna [ErrUsuarioNotFound].
func (s *Service) GetUsuario(ctx context.Context, usuarioID uuid.UUID) (*Usuario, error) {
	row, err := s.q.GetUsuario(ctx, pgtype.UUID{Bytes: usuarioID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUsuarioNotFound
		}
		return nil, fmt.Errorf("get usuario: %w", err)
	}

	usuario := usuarioFromDB(row)
	return &usuario, nil
}

type CreateUsuarioParams struct {
	Nome  string
	CPF   string
	Email string
	// Senha é um campo opcional. Um hash só será gerado se for estiver presente.
	Senha string
}

// CreateUsuario adiciona um novo usuário ao banco de dados. Retorna [ErrCPFTaken] ou [ErrEmailTaken] caso
// haja dados duplicados.
//
// Se o CPF for inválido, retorna [ErrInvalidCPF].
//
// Se o domínio do email não for permitido, retorna [ErrForbiddenDomain].
//
// Se uma senha for informada, ela será validada pela função [ValidatePassword],
// podendo retornar [*WeakPasswordError].
func (s *Service) CreateUsuario(ctx context.Context, params CreateUsuarioParams) (*Usuario, error) {
	nome := normalizeNome(params.Nome)
	cpf := normalizeCPF(params.CPF)
	email := normalizeEmail(params.Email)

	if err := ValidateCPF(cpf); err != nil {
		return nil, err
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return nil, fmt.Errorf("parse address: %w", err)
	}
	if !strings.HasSuffix(email, ".mg.gov.br") {
		return nil, ErrForbiddenDomain
	}

	var hashSenha pgtype.Text
	if params.Senha != "" {
		if err := ValidatePassword(params.Senha); err != nil {
			return nil, fmt.Errorf("validate password: %w", err)
		}

		pwHash, err := security.HashPassword(params.Senha)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}

		hashSenha = pgtype.Text{
			String: string(pwHash),
			Valid:  true,
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.q.WithTx(tx)

	row, err := q.SaveUsuario(ctx, postgres.SaveUsuarioParams{
		Nome:      nome,
		CPF:       cpf,
		Email:     email,
		HashSenha: hashSenha,
	})
	if err != nil {
		switch {
		case database.IsUniqueError(err, "usuarios_email_key"):
			return nil, ErrEmailTaken
		case database.IsUniqueError(err, "usuarios_cpf_key"):
			return nil, ErrCPFTaken
		default:
			return nil, fmt.Errorf("save usuario: %w", err)
		}
	}

	token, err := s.createToken(ctx, q, CreateTokenParams{
		UsuarioID: row.ID.Bytes,
		Escopo:    EscopoAtivarConta,
		TTL:       ativarContaTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("create token: %w", err)
	}

	err = s.notifier.SendAtivarConta(ctx, tx, row.Email, token.PlainText)
	if err != nil {
		return nil, fmt.Errorf("send ativar conta: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	usuario := usuarioFromDB(row)
	return &usuario, nil
}

type UpdateSenhaParams struct {
	UsuarioID   uuid.UUID
	SenhaAntiga string
	SenhaNova   string
}

// ChangeSenha troca a senha de um usuário.
// Se o usuário não possuir uma senha cadastrada, retorna [ErrNoSenha].
// Se a senha antiga informada for inválida, retorna [ErrInvalidCredentials].
// A nova senha será validada pela função [ValidatePassword], podendo retornar [*WeakPasswordError].
func (s *Service) ChangeSenha(ctx context.Context, params UpdateSenhaParams) error {
	usuario, err := s.GetUsuario(ctx, params.UsuarioID)
	if err != nil {
		return err
	}
	if usuario.HashSenha == nil {
		return ErrNoSenha
	}

	ok, err := security.VerifyPassword(*usuario.HashSenha, params.SenhaAntiga)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return ErrInvalidCredentials
	}

	if err := ValidatePassword(params.SenhaNova); err != nil {
		return fmt.Errorf("validate password: %w", err)
	}

	pwHash, err := security.HashPassword(params.SenhaNova)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	err = s.q.UpdateSenhaUsuario(ctx, postgres.UpdateSenhaUsuarioParams{
		ID:        pgtype.UUID{Bytes: usuario.ID, Valid: true},
		HashSenha: pgtype.Text{String: pwHash, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update senha usuario: %w", err)
	}

	return nil
}
