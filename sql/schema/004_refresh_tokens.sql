-- +goose Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT PK_refresh_tokens
        PRIMARY KEY (token),
    CONSTRAINT FK_refresh_tokens_to_users
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE refresh_tokens;