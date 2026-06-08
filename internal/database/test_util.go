package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/url"
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

type TestInstance struct {
	skipReason string

	pool     dockertest.ClosablePool
	resource dockertest.ClosableResource
	dbURL    *url.URL
	db       *sql.DB
}

func MustTestInstance() *TestInstance {
	ti, err := NewTestInstance(context.Background())
	if err != nil {
		panic(err)
	}
	return ti
}

func NewTestInstance(ctx context.Context) (*TestInstance, error) {
	if !flag.Parsed() {
		flag.Parse()
	}

	if testing.Short() {
		return &TestInstance{
			skipReason: "Pulando testes com PostgreSQL (flag -short)",
		}, nil
	}

	pool, err := dockertest.NewPool(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	resource, err := pool.Run(ctx,
		"postgres",
		dockertest.WithTag("17-alpine"),
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
