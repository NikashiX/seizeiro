package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/automatiza-mg/seizeiro/internal/postgres/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v4"
)

const (
	testUser     = "testuser"
	testPassword = "testpw"
	testDB       = "testdb"
)

// TestInstance é um wrapper em um banco de dados PostgreSQL baseado no Docker.
type TestInstance struct {
	skipReason string

	pool     dockertest.ClosablePool
	resource dockertest.ClosableResource
	dbURL    *url.URL
	db       *sql.DB
}

func MustTestInstance() *TestInstance {
	ti, err := NewTestInstance()
	if err != nil {
		log.Fatal(err)
	}
	return ti
}

// NewTestInstance cria uma nova instância do banco de dados PostgreSQL baseado no Docker.
// Cria também um banco de dados inicial e aplica todas as migrações.
//
// Essa função não deve ser usada fora de testes, mas é exposta publicamente para facilitar
// o reuso da lógica.
//
// Os testes podem ser pulados usando a flag `-short` ao executar os testes ou definindo
// a variável de ambiente `SKIP_DATABASE_TESTS`.
func NewTestInstance() (*TestInstance, error) {
	if !flag.Parsed() {
		flag.Parse()
	}

	if testing.Short() {
		return &TestInstance{
			skipReason: "Pulando testes de banco de dados (flag -short definida)",
		}, nil
	}

	if skip, _ := strconv.ParseBool(os.Getenv("SKIP_DATABASE_TESTS")); skip {
		return &TestInstance{
			skipReason: "Pulando testes de banco de dados (SKIP_DATABASE_TESTS definido)",
		}, nil
	}

	ctx := context.Background()

	pool, err := dockertest.NewPool(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	resource, err := pool.Run(ctx,
		"pgvector/pgvector",
		dockertest.WithTag("pg16"),
		dockertest.WithEnv([]string{
			"POSTGRES_USER=" + testUser,
			"POSTGRES_PASSWORD=" + testPassword,
			"POSTGRES_DB=" + testDB,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("pool run: %w", err)
	}

	dbURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(testUser, testPassword),
		Host:   resource.GetHostPort("5432/tcp"),
		Path:   testDB,
	}

	var db *sql.DB
	err = dockertest.Retry(ctx, time.Minute, time.Second, func() error {
		db, err = sql.Open("pgx", dbURL.String())
		if err != nil {
			return err
		}
		// Evita o acesso concorrente ao testdb
		db.SetMaxOpenConns(1)
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if err := migrations.Up(ctx, db); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	if err := migrations.RiverUp(ctx, db); err != nil {
		return nil, fmt.Errorf("apply river migrations: %w", err)
	}

	return &TestInstance{
		pool:     pool,
		resource: resource,
		dbURL:    dbURL,
		db:       db,
	}, nil
}

func (ti *TestInstance) NewPool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	if ti.skipReason != "" {
		tb.Skip(ti.skipReason)
	}

	dbName := rand.Text()
	q := fmt.Sprintf(`CREATE DATABASE "%s" WITH TEMPLATE "%s"`, dbName, testDB)
	_, err := ti.db.ExecContext(tb.Context(), q)
	if err != nil {
		tb.Fatal(err)
	}

	dbURL := ti.dbURL.ResolveReference(&url.URL{
		Path: dbName,
	})

	pool, err := New(tb.Context(), dbURL.String())
	if err != nil {
		tb.Fatal(err)
	}

	tb.Cleanup(func() {
		pool.Close()

		q := fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, dbName)
		_, err := ti.db.Exec(q)
		if err != nil {
			tb.Errorf("Failed to drop database: %v", err)
		}
	})

	return pool
}

// Close fecha os recursos utilizados por TestInstance.
func (ti *TestInstance) Close(ctx context.Context) error {
	if ti.skipReason != "" {
		return nil
	}

	return errors.Join(
		ti.db.Close(),
		ti.resource.Close(ctx),
		ti.pool.Close(ctx),
	)
}
