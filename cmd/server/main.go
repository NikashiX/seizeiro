package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	<-ctx.Done()
	return nil
}
