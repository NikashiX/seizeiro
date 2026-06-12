-- name: GetUsuarioChatbot :one
SELECT *
FROM usuarios_chatbot
WHERE plataforma = $1
AND plataforma_id = $2;

-- name: SaveUsuarioChatbot :exec
INSERT INTO usuarios_chatbot (plataforma, plataforma_id, sei_usuario, sei_senha)
VALUES ($1, $2, $3, $4)
ON CONFLICT (plataforma, plataforma_id) DO UPDATE SET
    sei_usuario = EXCLUDED.sei_usuario,
    sei_senha = EXCLUDED.sei_senha;

-- name: SaveTokenChatbot :exec
INSERT INTO tokens_chatbot (hash, plataforma, plataforma_id, expira_em)
VALUES ($1, $2, $3, $4);

-- name: GetTokenChatbot :one
SELECT *
FROM tokens_chatbot
WHERE hash = $1
AND expira_em > CURRENT_TIMESTAMP;

-- name: DeleteTokenChatbot :exec
DELETE FROM tokens_chatbot
WHERE hash = $1;

-- name: DeleteTokensChatbot :exec
DELETE FROM tokens_chatbot
WHERE (plataforma = $1 AND plataforma_id = $2)
OR expira_em <= CURRENT_TIMESTAMP;
