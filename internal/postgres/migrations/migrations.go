// Package migrations contém funções para aplicar as migrações do banco de dados PostgreSQL
// de forma programática.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var fs embed.FS

// Up aplica todas as migrações ao banco de dados PostgreSQL.
func Up(ctx context.Context, db *sql.DB) error {
	m, err := goose.NewProvider(goose.DialectPostgres, db, fs)
	if err != nil {
		return fmt.Errorf("new provider: %w", err)
	}

	_, err = m.Up(ctx)
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}
