-- +goose Up
ALTER TABLE "usuarios_chatbot"
    ADD COLUMN "sei_orgao" INTEGER NOT NULL DEFAULT 0;

ALTER TABLE "usuarios_chatbot"
    ALTER COLUMN "sei_orgao" DROP DEFAULT;

-- +goose Down
ALTER TABLE "usuarios_chatbot"
    DROP COLUMN "sei_orgao";
