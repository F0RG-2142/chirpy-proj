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

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET
    updated_at = NOW(),
    revoked_at = NOW()
WHERE
    token = $1;

-- name: UpdateUser :exec
UPDATE users
SET
    updated_at = NOW(),
    email = $1,
    hashed_password= $2
WHERE
    id = $3;

-- name: DeleteYap :exec
DELETE FROM yaps WHERE id = $1;