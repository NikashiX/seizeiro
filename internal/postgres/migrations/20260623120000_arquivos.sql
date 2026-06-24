-- +goose Up
-- Tabela `arquivos` armazena, deduplicado por SHA-256, todo conteúdo
-- binário baixado do SEI (anexos, PDFs, HTML, XML, etc.).
CREATE TABLE "arquivos" (
    "hash" TEXT PRIMARY KEY,
    "storage_key" TEXT NOT NULL,
    "content_type" TEXT NOT NULL,
    "tamanho_bytes" BIGINT NOT NULL,
    "criado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Tabela `documentos_anexo` mapeia id_protocolo -> hash do último arquivo
-- baixado. Permite descobrir o conteúdo atual de um anexo sem refazer o
-- download quando o hash não mudou.
CREATE TABLE "documentos_anexo" (
    "id_protocolo" BIGINT PRIMARY KEY,
    "hash" TEXT NOT NULL REFERENCES "arquivos"("hash"),
    "baixado_em" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_documentos_anexo_hash ON "documentos_anexo"("hash");

-- +goose Down
DROP TABLE "documentos_anexo";
DROP TABLE "arquivos";
