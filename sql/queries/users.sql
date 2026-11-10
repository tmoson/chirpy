-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password)
VALUES (
  gen_random_uuid(),
  $1,
  $2
)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE email = $1;

-- name: ResetUsers :exec
DELETE FROM users *;
