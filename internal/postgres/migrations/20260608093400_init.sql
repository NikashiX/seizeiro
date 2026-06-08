-- +goose Up
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE "usuarios" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "nome" TEXT NOT NULL,
    "cpf" TEXT NOT NULL UNIQUE CHECK ("cpf" ~ '^[0-9]{11}$'),
    "email" CITEXT NOT NULL UNIQUE,
    "email_verificado" BOOLEAN NOT NULL DEFAULT false,
    "hash_senha" TEXT,
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "atualizado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE "usuarios";
