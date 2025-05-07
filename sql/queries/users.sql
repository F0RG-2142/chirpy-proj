-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid (),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: NewYap :one
INSERT INTO yaps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid (),
    NOW(),
    NOW(),
    $1,
    $2 
)
RETURNING *;

-- name: NewRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    NOW() + INTERTVAL '60 days',
    NULL
)
RETURNING *;

-- name: GetAllYaps :many
SELECT * FROM yaps ORDER BY created_at ASC;

-- name: GetYapByID :one
SELECT * FROM yaps WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1;