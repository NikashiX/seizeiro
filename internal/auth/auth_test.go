package auth

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/automatiza-mg/seizeiro/internal/database"
)

var ti *database.TestInstance

func newTestService(tb testing.TB) *Service {
	tb.Helper()
	pool := ti.NewPool(tb)
	return NewService(pool)
}

func TestMain(m *testing.M) {
	ti = database.MustTestInstance()
	code := m.Run()

	if err := ti.Close(context.Background()); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}
