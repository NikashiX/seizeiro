-- +goose Up
CREATE EXTENSION IF NOT EXISTS "vector";

CREATE TABLE "arquivos" (
    "id" BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "hash_sha256" BYTEA NOT NULL UNIQUE,
    "chave_storage" TEXT NOT NULL UNIQUE,
    "mime_type" TEXT NOT NULL,
    "tamanho_bytes" BIGINT NOT NULL,
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "arquivos_conteudo" (
    "id" BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "arquivo_id" BIGINT NOT NULL REFERENCES "arquivos"("id"),
    "metodo" TEXT NOT NULL, -- ocr, html_markdown, plain, etc.
    "formato" TEXT NOT NULL, -- plain, markdown.
    "conteudo" TEXT NOT NULL, -- texto completo
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE ("arquivo_id", "metodo")
);

CREATE TABLE "arquivos_conteudo_chunks" (
    "id" BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "conteudo_id" BIGINT NOT NULL REFERENCES "arquivos_conteudo"("id"),
    "indice" INT NOT NULL,
    "conteudo" TEXT NOT NULL,
    "tokens" INT,
    "embedding" VECTOR(1536),
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE ("conteudo_id", "indice")
);
CREATE INDEX ON "arquivos_conteudo_chunks" ("conteudo_id", "indice");
CREATE INDEX ON "arquivos_conteudo_chunks" USING hnsw ("embedding" vector_cosine_ops);

-- +goose Down
DROP TABLE "arquivos_conteudo_chunks";
DROP TABLE "arquivos_conteudo";
DROP TABLE "arquivos";
DROP EXTENSION IF EXISTS "vector";
