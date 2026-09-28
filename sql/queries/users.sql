-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(), Now(), Now(), $1, $2
)
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT * FROM users WHERE email=$1;

-- name: UpdateUser :one
UPDATE users
SET email=$2,
hashed_password=$3
WHERE id=$1
RETURNING *;

-- name: AddChirpyRed :one
UPDATE users
SET is_chirpy_red=true
WHERE id=$1
RETURNING *;