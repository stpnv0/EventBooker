-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY,
    username         VARCHAR(255) NOT NULL UNIQUE,
    telegram_chat_id BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS users;
