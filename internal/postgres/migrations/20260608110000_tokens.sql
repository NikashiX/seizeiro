-- +goose Up
CREATE TABLE "tokens" (
    "hash" BYTEA PRIMARY KEY,
    "usuario_id" UUID NOT NULL REFERENCES "usuarios"("id") ON DELETE CASCADE,
    "escopo" TEXT NOT NULL,
    "expira_em" TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE "tokens";
