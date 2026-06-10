-- name: SaveArquivo :one
INSERT INTO arquivos (hash_sha256, chave_storage, mime_type, tamanho_bytes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SaveArquivoConteudo :one
INSERT INTO arquivos_conteudo (arquivo_id, metodo, formato, conteudo)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SaveArquivoConteudoChunk :exec
INSERT INTO arquivos_conteudo_chunks (conteudo_id, indice, conteudo, tokens, embedding)
VALUES ($1, $2, $3, $4, $5);

-- name: GetArquivo :one
SELECT * FROM arquivos WHERE id = $1;

-- name: GetArquivoConteudoLatest :one
SELECT * FROM arquivos_conteudo
WHERE arquivo_id = $1
ORDER BY criado_em DESC
LIMIT 1;
