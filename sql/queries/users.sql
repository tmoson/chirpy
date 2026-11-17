-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password)
VALUES (
  gen_random_uuid(),
  $1,
  $2
)
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: UpdateUserEmailAndPassword :one
UPDATE users
SET hashed_password = $2, email = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: UpgradeUserToRed :exec
UPDATE users
SET is_chirpy_red = true, updated_at = now()
WHERE id = $1;

-- name: ResetUsers :exec
DELETE FROM users *;
