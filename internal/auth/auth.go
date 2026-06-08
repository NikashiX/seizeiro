package auth

import (
	"github.com/automatiza-mg/seizeiro/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
	q    *postgres.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool: pool,
		q:    postgres.New(pool),
	}
}
