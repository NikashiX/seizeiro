-- name: SaveToken :exec
INSERT INTO tokens (hash, usuario_id, escopo, expira_em)
VALUES ($1, $2, $3, $4);

-- GetUsuarioForToken retorna o usuário dono de um determinado token que ainda não tenha expirado.
-- name: GetUsuarioForToken :one
SELECT u.*
FROM usuarios u
JOIN tokens t ON t.usuario_id = u.id
WHERE t.hash = $1
AND t.escopo = $2
AND t.expira_em > CURRENT_TIMESTAMP;
