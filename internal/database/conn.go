// Package database contém funções para conexão com banco de dados e utilidades para testes integrados.
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New cria uma nova pool de conexões com o banco de dados PostgreSQL.
func New(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}

// IsUniqueError verifica se o erro informado é uma unique_violation para a constraint informada.
//
// O nome da constraint gerada pelo PostgreSQL segue o seguinte padrão: {tabela}_{coluna}_key.
//
//	ok := IsUniqueError(err, "usuarios_cpf_key") // Tabela: "usuarios" Coluna: "cpf".
func IsUniqueError(err error, constraintName string) bool {
	pgError, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return false
	}
	return pgError.Code == "23505" && pgError.ConstraintName == constraintName
}
