-- +goose Up
CREATE TABLE "tokens_chatbot" (
    "hash" BYTEA PRIMARY KEY,
    "plataforma" TEXT NOT NULL,
    "plataforma_id" TEXT NOT NULL,
    "expira_em" TIMESTAMPTZ NOT NULL
);

CREATE TABLE "usuarios_chatbot" (
    "plataforma" TEXT NOT NULL,
    "plataforma_id" TEXT NOT NULL,
    "sei_usuario" TEXT NOT NULL,
    "sei_senha" BYTEA NOT NULL,
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("plataforma", "plataforma_id")
);

-- +goose Down
DROP TABLE "tokens_chatbot";
DROP TABLE "usuarios_chatbot";
