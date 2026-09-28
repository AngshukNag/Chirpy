-- +goose Up

ALTER TABLE users
ADD COLUMN hashed_password TEXT;

ALTER TABLE users
ALTER COLUMN hashed_password SET DEFAULT 'unset';

-- +goose Down

ALTER TABLE users
DROP COLUMN hashed_password;

ALTER TABLE users
ALTER COLUMN hashed_password DROP DEFAULT;