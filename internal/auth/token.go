package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/automatiza-mg/seizeiro/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	tokenLength = 64
)

const (
	// EscopoAuth é o escopo usado para tokens de autenticação.
	EscopoAuth Escopo = "auth"
	// EscopoResetSenha é o escopo usado para tokens de redefinição de senha.
	EscopoResetSenha Escopo = "reset-senha"
)

// Escopo define o uso de um [Token].
type Escopo string

// Token é um valor com expiração e finalidade específica emitido a um usuário.
type Token struct {
	// PlainText é o valor do token e deve ser retornado ao usuário apenas uma vez.
	PlainText string    `json:"token"`
	ExpiraEm  time.Time `json:"expira_em"`
}

type CreateTokenParams struct {
	UsuarioID uuid.UUID
	Escopo    Escopo
	// TTL é o tempo de duração (time-to-live) de um token.
	// Deve ser maior que zero.
	TTL time.Duration
}

// CreateToken cria um novo token com escopo e duração para um determinado usuário.
func (s *Service) CreateToken(ctx context.Context, params CreateTokenParams) (*Token, error) {
	if params.TTL <= 0 {
		return nil, fmt.Errorf("invalid token ttl: %s", params.TTL)
	}

	b := security.RandomBytes(tokenLength)
	plainText := base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(plainText))
	expiraEm := time.Now().Add(params.TTL)

	err := s.q.SaveToken(ctx, postgres.SaveTokenParams{
		Hash:      hash[:],
		UsuarioID: pgtype.UUID{Bytes: params.UsuarioID, Valid: true},
		Escopo:    string(params.Escopo),
		ExpiraEm:  pgtype.Timestamptz{Time: expiraEm, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return &Token{
		PlainText: plainText,
		ExpiraEm:  expiraEm,
	}, nil
}

// GetTokenOwner retorna o dono de um determinado token.
// Se o token não for encontrado ou tenha expirado, retorna [ErrInvalidToken].
func (s *Service) GetTokenOwner(ctx context.Context, token string, escopo Escopo) (*Usuario, error) {
	hash := sha256.Sum256([]byte(token))

	row, err := s.q.GetUsuarioForToken(ctx, postgres.GetUsuarioForTokenParams{
		Hash:   hash[:],
		Escopo: string(escopo),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("get usuario for token: %w", err)
	}

	usuario := usuarioFromDB(row)
	return &usuario, nil
}
