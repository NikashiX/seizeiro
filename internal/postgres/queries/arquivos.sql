-- GetArquivo busca o registro de arquivo pelo hash.
-- name: GetArquivo :one
SELECT * FROM arquivos WHERE hash = $1;

-- SaveArquivo insere um novo arquivo deduplicado por hash.
-- name: SaveArquivo :exec
INSERT INTO arquivos (hash, storage_key, content_type, tamanho_bytes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (hash) DO NOTHING;

-- GetDocumentoAnexo retorna o último hash conhecido para um id_protocolo.
-- name: GetDocumentoAnexo :one
SELECT * FROM documentos_anexo WHERE id_protocolo = $1;

-- SaveDocumentoAnexo registra (ou atualiza) o hash atual associado a um
-- id_protocolo. O timestamp `baixado_em` é renovado a cada chamada.
-- name: SaveDocumentoAnexo :exec
INSERT INTO documentos_anexo (id_protocolo, hash, baixado_em)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (id_protocolo) DO UPDATE SET
    hash = EXCLUDED.hash,
    baixado_em = EXCLUDED.baixado_em;
