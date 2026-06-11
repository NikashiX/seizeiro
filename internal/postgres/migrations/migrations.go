// Package migrations contém funções para aplicar as migrações do banco de dados PostgreSQL
// de forma programática.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
)

//go:embed *.sql
var fs embed.FS

// Up aplica todas as migrações da aplicação ao banco de dados
// PostgreSQL.
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

// RiverUp aplica as migrações do schema do River (river_job, etc) ao banco de
// dados PostgreSQL.
func RiverUp(ctx context.Context, db *sql.DB) error {
	migrator, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	if err != nil {
		return fmt.Errorf("new river migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrate up: %w", err)
	}

	return nil
}
