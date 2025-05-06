-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_uuid (),
    NOW(),
    NOW(),
    $1
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

-- name: GetAllYaps :many
SELECT * FROM yaps ORDER BY created_at ASC;